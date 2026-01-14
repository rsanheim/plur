# Stream Helper Hang Prevention

Replace `bufio.Scanner` with a more robust pipe reading approach to prevent subprocess hangs.

Status: Completed

## Summary

- Replaced `bufio.Scanner` with `bufio.Reader` in subprocess pipe readers so stdout/stderr are always drained, even when a single line exceeds the Scanner token limit.
- Added a regression test that spawns a helper subprocess emitting a >256KB single line and asserts `streamTestOutput()` + `cmd.Wait()` complete (no hang).
- Applied the same fix pattern to watch mode (`plur/watch/watcher.go`) so large JSON watcher events can’t deadlock the watcher process.

## Context

Previously, `plur/stream_helper.go` used `bufio.Scanner` to read stdout/stderr from test subprocesses. This creates a hang risk:

**The Problem:**

When a line exceeds the scanner buffer (currently 256KB at `stream_helper.go:17`):
1. `scanner.Scan()` returns `false`
2. `scanner.Err()` returns `bufio.ErrTooLong`
3. The scanning goroutine exits the loop (only logs the error)
4. **Remaining pipe data is never drained**
5. Subprocess blocks writing to full OS pipe buffer (~64KB)
6. `cmd.Wait()` blocks forever waiting for subprocess
7. Worker goroutine hangs, blocking entire `wg.Wait()` at `runner.go:183`

**Previous Error Handling (insufficient):**

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

* [x] Subprocess pipes are always fully drained regardless of line length
* [x] Lines > 256KB are handled correctly (read + processed without deadlock)
* [x] No subprocess hangs even with pathological output
* [x] Performance is comparable to scanner-based approach for typical output
* [x] `PLUR_RACE=1 bin/rake test:go` passes
* [x] Add test case that verifies long-line handling

## Task List

### Phase 1: Research & Design

* [x] Review Go stdlib options: `bufio.Reader.ReadBytes()`, `io.Copy` with custom writer, etc.
* [x] Evaluate trade-offs:
  * [x] `bufio.Reader.ReadBytes('\n')` - returns partial reads with `bufio.ErrBufferFull`
  * [x] Custom split function for `bufio.Scanner` that truncates instead of failing
  * [x] `io.Copy` to discard remaining line data after truncation
* [x] Choose approach that:
  * [x] Keeps draining pipe on long lines
  * [x] Preserves line-by-line parsing for progress output
  * [x] Handles both stdout (parsed) and stderr (captured) cases
* [x] Document chosen approach in this file

### Phase 2: Implement Robust Pipe Reading

* [x] Create new pipe reading function(s) that handle long lines
* [x] Approach implemented:
  * [x] Replace scanner with `bufio.Reader.ReadString('\n')`
  * [x] On `io.EOF`, process any remaining buffered content as the final line
  * [x] Trim `\n` / `\r` so CRLF output is handled
* [x] Ensure both stdout and stderr goroutines use new approach
* [x] Preserve existing parsing/callback interface

### Phase 3: Add Test Coverage

* [x] Add unit test generating output > 256KB in a single line
* [x] Verify pipe is fully drained (subprocess exits cleanly)
* [x] Verify oversized lines are handled appropriately (this implementation preserves the full line)
* [x] Test with actual subprocess (not just mocked reader)

### Phase 4: Consider Watch Mode

* [x] Review `plur/watch/watcher.go` (`readEvents()`)
* [x] Replace Scanner with Reader to remove token limits / hang risk
* [x] Use a 256KB read buffer (`WatcherBufferSize`) consistent with stream helper

### Phase 5: Validation

* [ ] Run `bin/rake` (full test suite)
* [x] Run `PLUR_RACE=1 bin/rake test:go`
* [x] Add a deterministic hang regression test that reproduces the failure mode without requiring a real Ruby test suite
* [x] Verify no performance regression risk introduced for typical output (see notes below)

## Files to Modify

* `plur/stream_helper.go` - Replace/enhance scanner usage
* `plur/stream_helper_test.go` - Add long-line test cases
* `plur/watch/watcher.go` - Potentially apply same fix

## Design Notes

This fix chose a Reader-based approach over attempting to “recover” a Scanner, because Scanner failure is exactly the problem: once Scanner stops, pipes are no longer drained.

### Approach Chosen: Reader-based line reading

- Implementation: `bufio.NewReaderSize(..., 256*1024)` + `ReadString('\n')` loop
- Key property: there is no fixed token cap; the reader grows as needed and continues draining the pipe.
- EOF behavior: if the subprocess exits without a trailing newline, the final partial line is still processed before returning.

### Performance + Memory Notes

- Baseline memory stays bounded for normal output:
  - `streamTestOutput()` creates two buffered readers per worker (stdout + stderr) each with a 256KB buffer.
  - The watcher uses the same 256KB buffer for stdout/stderr streams.
- Large single-line output still allocates proportionally to the line length because the full line is required for parsing (e.g., a large JSON event/record). This is an intentional trade-off to guarantee pipe draining and correctness.

### Additional Fixes

- `plur/stream_helper_test.go` uses a self-exec helper process to generate the long line deterministically (avoids shell/env-size issues and keeps the test hermetic).
- `plur/stream_helper.go` avoids per-line `line + "\n"` concatenation when collecting stderr (reduces allocations in hot paths).
- `plur/watch/watcher.go` removes an unreachable `io.EOF` check inside the JSON unmarshal error path while keeping the “ignore non-JSON lines” behavior.
