# Logger Concurrency Safety

Add thread-safety to `CustomTextHandler` to prevent log interleaving.

## Context

The `slog.Handler` interface requires implementations to be safe for concurrent use. The current `CustomTextHandler` lacks synchronization around writes.

**The Problem:**

`plur/logger/logger.go:58-86` - `Handle()` method:
```go
func (h *CustomTextHandler) Handle(_ context.Context, r slog.Record) error {
    // ... builds string in sb ...
    _, err := io.WriteString(h.writer, sb.String())
    return err
    // No mutex! Multiple goroutines can interleave at byte level
}
```

**Why It Matters:**

* `runner.go:173-181` spawns multiple worker goroutines
* Each worker logs via `logger.Logger` (e.g., errors in `stream_helper.go:104-106`)
* Without synchronization, concurrent `io.WriteString()` calls can produce garbled output
* Individual bytes from different log lines can interleave

**What's Already Safe:**

* `slog.LevelVar` (line 16) is internally thread-safe for atomic level changes
* The concurrency test at `logger_test.go:125-151` only tests `logLevel`, not write safety

**Severity:**

Low impact in practice:
* Most logging happens during single-threaded initialization
* Worker logging is rare (only on errors)
* But when it happens, output corruption is confusing to debug

## Success Criteria

* [ ] `CustomTextHandler.Handle()` is thread-safe
* [ ] No log interleaving under concurrent writes
* [ ] Minimal performance impact (mutex only around write)
* [ ] `PLUR_RACE=1 bin/rake test:go` passes
* [ ] Add test verifying concurrent log safety

## Task List

### Implementation

* [ ] Add `mu sync.Mutex` field to `CustomTextHandler` struct
* [ ] Lock around `io.WriteString()` call in `Handle()`:
  ```go
  h.mu.Lock()
  _, err := io.WriteString(h.writer, sb.String())
  h.mu.Unlock()
  return err
  ```
* [ ] Alternative: use `defer h.mu.Unlock()` for safety (slightly more overhead)

### Testing

* [ ] Add test in `logger_test.go` that:
  * Spawns multiple goroutines logging concurrently
  * Captures output to buffer
  * Verifies no interleaved/corrupted lines
* [ ] Run with race detector: `PLUR_RACE=1 go test ./plur/logger/...`

### Alternative Approach (not recommended for this case)

* Use `slog.NewTextHandler` with `HandlerOptions.ReplaceAttr`
* Built-in concurrency guarantees
* But loses custom formatting (would need to reimplement)

## Validation

* [ ] Run `PLUR_RACE=1 bin/rake test:go`
* [ ] Run `bin/rake` (full test suite)
* [ ] Manual testing with high parallelism: `plur -n 8`

## Files to Modify

* `plur/logger/logger.go` - Add mutex, lock around writes
* `plur/logger/logger_test.go` - Add concurrent write test

## Implementation Snippet

```go
// logger.go

type CustomTextHandler struct {
    opts   slog.HandlerOptions
    writer io.Writer
    mu     sync.Mutex  // Add this
}

func (h *CustomTextHandler) Handle(ctx context.Context, r slog.Record) error {
    // ... existing string building ...

    h.mu.Lock()
    _, err := io.WriteString(h.writer, sb.String())
    h.mu.Unlock()
    return err
}
```

## Note on WithAttrs/WithGroup

If `WithAttrs()` or `WithGroup()` create new handler instances, ensure the mutex is shared appropriately or each instance has its own (current impl returns `h` without cloning, so shared mutex is correct).

Current code (`logger.go:92-98`):
```go
func (h *CustomTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return h  // Returns same instance, shared mutex is correct
}

func (h *CustomTextHandler) WithGroup(name string) slog.Handler {
    return h  // Returns same instance, shared mutex is correct
}
```
