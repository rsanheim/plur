# Stream Helper Hang Prevention

Replace `bufio.Scanner` with a more robust pipe reading approach to prevent subprocess hangs.

## Context

`plur/stream_helper.go` uses `bufio.Scanner` to read stdout/stderr from test subprocesses. This creates a hang risk:

**The Problem:**

When a line exceeds the scanner buffer (currently 256KB at `stream_helper.go:17`):
1. `scanner.Scan()` returns `false`
2. `scanner.Err()` returns `bufio.ErrTooLong`
3. The scanning goroutine exits the loop (only logs the error)
4. **Remaining pipe data is never drained**
5. Subprocess blocks writing to full OS pipe buffer (~64KB)
6. `cmd.Wait()` blocks forever waiting for subprocess
7. Worker goroutine hangs, blocking entire `wg.Wait()` at `runner.go:183`

**Current Error Handling (insufficient):**

```go
// stream_helper.go:104-106
if err := scanner.Err(); err != nil {
    logger.Logger.Error("error reading stdout", "error", err, "worker", workerIndex)
}
```

This logs the error but doesn't continue draining the pipe.

**Likelihood:**

* 256KB buffer handles typical test output
* Pathological cases: single lines > 256KB from large data structures, binary output, malformed JSON, huge stack traces
* When it happens, entire test run hangs silently

**Related Code Paths:**

* `plur/runner.go:209-229` - Creates pipes, calls `streamTestOutput()`, then `cmd.Wait()`
* `plur/runner.go:173-181` - Worker goroutines that would hang
* `plur/watch/watcher.go:127-151` - Also uses scanner (default 64KB buffer)

## Success Criteria

* [ ] Subprocess pipes are always fully drained regardless of line length
* [ ] Lines > 256KB are handled gracefully (truncate content but keep draining)
* [ ] No subprocess hangs even with pathological output
* [ ] Performance is comparable to current scanner approach
* [ ] `PLUR_RACE=1 bin/rake test:go` passes
* [ ] Add test case that verifies long-line handling

## Task List

### Phase 1: Research & Design

* [ ] Review Go stdlib options: `bufio.Reader.ReadBytes()`, `io.Copy` with custom writer, etc.
* [ ] Evaluate trade-offs:
  * `bufio.Reader.ReadBytes('\n')` - returns partial reads with `bufio.ErrBufferFull`
  * Custom split function for `bufio.Scanner` that truncates instead of failing
  * `io.Copy` to discard remaining line data after truncation
* [ ] Choose approach that:
  * Keeps draining pipe on long lines
  * Preserves line-by-line parsing for progress output
  * Handles both stdout (parsed) and stderr (captured) cases
* [ ] Document chosen approach in this file

### Phase 2: Implement Robust Pipe Reading

* [ ] Create new pipe reading function(s) that handle long lines
* [ ] Options to consider:
  * Replace scanner with `bufio.Reader.ReadString('\n')` + truncation logic
  * Custom `SplitFunc` for scanner that truncates oversized tokens
  * Wrapper that catches `ErrTooLong` and drains remainder
* [ ] Ensure both stdout and stderr goroutines use new approach
* [ ] Preserve existing parsing/callback interface

### Phase 3: Add Test Coverage

* [ ] Add unit test generating output > 256KB in a single line
* [ ] Verify pipe is fully drained (subprocess exits cleanly)
* [ ] Verify truncated content is logged/handled appropriately
* [ ] Test with actual subprocess (not just mocked reader)

### Phase 4: Consider Watch Mode

* [ ] Review `plur/watch/watcher.go:127-151` (`readEvents()`)
* [ ] Watcher uses default 64KB buffer
* [ ] JSON events > 64KB could cause same issue
* [ ] Apply same fix pattern if appropriate

### Phase 5: Validation

* [ ] Run `bin/rake` (full test suite)
* [ ] Run `PLUR_RACE=1 bin/rake test:go`
* [ ] Test with real-world large output (e.g., test that dumps large object)
* [ ] Verify no performance regression

## Files to Modify

* `plur/stream_helper.go` - Replace/enhance scanner usage
* `plur/stream_helper_test.go` - Add long-line test cases
* `plur/watch/watcher.go` - Potentially apply same fix

## Design Notes

**Approach Option A: Custom SplitFunc**

```go
// SplitFunc that truncates instead of failing
func truncatingSplitFunc(maxLen int) bufio.SplitFunc {
    return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
        // Standard ScanLines logic, but cap token length
        // Continue advancing through oversized lines
    }
}
```

**Approach Option B: Reader-based draining**

```go
reader := bufio.NewReader(pipe)
for {
    line, err := reader.ReadString('\n')
    if err == bufio.ErrBufferFull {
        // Truncate and drain remainder
        io.Copy(io.Discard, &lineDrainer{reader, '\n'})
    }
    // Process truncated line
}
```

**Approach Option C: Post-error recovery**

```go
for scanner.Scan() {
    // Process line
}
if scanner.Err() == bufio.ErrTooLong {
    // Drain remaining pipe data
    io.Copy(io.Discard, stdout)
}
```
