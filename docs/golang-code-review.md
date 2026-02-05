# Plur Go Code Review

## Overview

* **Date**: 2026-02-05
* **Skill Used**: golang-best-practices-skill v2.0.0
* **Go Version**: 1.25.2
* **Files Reviewed**: 8 Go files in `plur/` and `plur/watch/`
* **Total Lines Reviewed**: ~1,900 lines

## Summary by Priority

| Priority | Count | Description |
|----------|-------|-------------|
| CRITICAL | 2 | Prevents bugs, crashes, and production failures |
| HIGH | 2 | Reliability and architecture improvements |
| MEDIUM | 5 | Code quality and idiomatic Go improvements |

---

## Critical Issues

### 1. Error Wrapping Uses `%v` Instead of `%w` - `runner.go:223-230`

**Rule**: `critical-error-wrapping`

**Impact**: Without `%w`, error chains are lost, breaking `errors.Is/As` and stack traces for debugging.

**Current Code**:
```go
stdout, err := cmd.StdoutPipe()
if err != nil {
    return errorResult(fmt.Errorf("failed to create stdout pipe: %v", err), start)
}
stderr, err := cmd.StderrPipe()
if err != nil {
    return errorResult(fmt.Errorf("failed to create stderr pipe: %v", err), start)
}
if err := cmd.Start(); err != nil {
    return errorResult(fmt.Errorf("failed to start command: %v", err), start)
}
```

**Fix**: Use `%w` for proper error wrapping:
```go
stdout, err := cmd.StdoutPipe()
if err != nil {
    return errorResult(fmt.Errorf("failed to create stdout pipe: %w", err), start)
}
stderr, err := cmd.StderrPipe()
if err != nil {
    return errorResult(fmt.Errorf("failed to create stderr pipe: %w", err), start)
}
if err := cmd.Start(); err != nil {
    return errorResult(fmt.Errorf("failed to start command: %w", err), start)
}
```

---

### 2. Error Wrapping Uses `%v` Instead of `%w` - `main.go:350`

**Rule**: `critical-error-wrapping`

**Impact**: Caller cannot use `errors.Is()` to check specific error types.

**Current Code**:
```go
if err := os.Chdir(dir); err != nil {
    return fmt.Errorf("failed to change directory to %s: %v", dir, err)
}
```

**Fix**:
```go
if err := os.Chdir(dir); err != nil {
    return fmt.Errorf("failed to change directory to %s: %w", dir, err)
}
```

---

## High-Priority Issues

### 1. Channels Closed in Multiple Locations - `watcher.go:142-143`

**Rule**: `high-channel-not-closed` (related)

**Impact**: Both `eventChan` and `errorChan` are closed in `readEvents()`, but `readErrors()` also runs concurrently. If `readErrors()` tries to send to `errorChan` after it's closed, it could panic.

**Current Code** (in `readEvents`):
```go
func (w *Watcher) readEvents(stdout io.Reader) {
    reader := bufio.NewReaderSize(stdout, WatcherBufferSize)
    defer close(w.eventChan)
    defer close(w.errorChan)  // Closes errorChan
    // ...
}
```

**Current Code** (in `readErrors`):
```go
func (w *Watcher) readErrors(stderr io.Reader) {
    // This function may still be running when errorChan is closed
    // ...
}
```

**Assessment**: The current code appears safe because `readErrors()` doesn't send to `errorChan` (it writes directly to stderr), but the design is fragile. If `readErrors()` is later modified to send errors, it could panic.

**Recommendation**: Consider a clearer ownership model where only the channel creator closes it, or use a separate done channel to coordinate shutdown.

---

### 2. Missing Context Propagation - `runner.go:171`

**Rule**: `high-context-propagation`

**Impact**: `context.Background()` is created but not propagated through to commands, preventing graceful cancellation of running tests.

**Current Code**:
```go
func (r *Runner) executeWorkers(commands []*exec.Cmd) ([]WorkerResult, time.Duration) {
    start := time.Now()
    ctx := context.Background()  // Created but not fully utilized
    // ...
}
```

**Assessment**: The context is passed to `runCommand` but `exec.Cmd` isn't using `CommandContext`. This means if the parent process is signaled, child processes won't be killed gracefully.

**Recommendation**: Use `exec.CommandContext` when building commands:
```go
cmd := exec.CommandContext(ctx, args[0], args[1:]...)
```

---

## Medium-Priority Issues

### 1. Long Function - `cmd_watch.go:117-368` (251 lines)

**Rule**: `high-god-object` / `high-extract-method`

**Impact**: `runWatchWithConfig` at 251 lines is approaching the "yellow flag" threshold (200 lines) for function length.

**Recommendation**: Consider extracting logical sections:
* Configuration loading and validation (lines 117-165)
* Watcher manager setup (lines 184-200)
* Event loop setup with channels (lines 202-240)
* Main event loop (lines 257-367)

---

### 2. Magic Numbers - `watcher.go:59-62`

**Rule**: `medium-magic-constants`

**Impact**: Buffer sizes 100 and 10 for channels are unexplained.

**Current Code**:
```go
return &Watcher{
    config:     config,
    binaryPath: binaryPath,
    eventChan:  make(chan Event, 100),
    errorChan:  make(chan error, 10),
    stopChan:   make(chan struct{}),
    done:       make(chan struct{}),
}
```

**Recommendation**: Define named constants:
```go
const (
    eventChannelBuffer = 100  // Buffer for file system events
    errorChannelBuffer = 10   // Buffer for error messages
)
```

---

### 3. Repeated Error Handling Pattern - `runtime_tracker.go:114-131`

**Rule**: `medium-data-clumps` (related)

**Impact**: Error handling silently returns empty map in multiple places, making debugging difficult.

**Current Code**:
```go
func loadExistingData(runtimeFile string) map[string]float64 {
    if _, err := os.Stat(runtimeFile); os.IsNotExist(err) {
        return make(map[string]float64)
    }

    file, err := os.Open(runtimeFile)
    if err != nil {
        return make(map[string]float64)  // Silent failure
    }
    defer file.Close()

    var runtimes map[string]float64
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&runtimes); err != nil {
        return make(map[string]float64)  // Silent failure
    }

    return runtimes
}
```

**Assessment**: This is intentional - the function is meant to be resilient and return empty data if the file doesn't exist or is corrupt. Consider adding debug logging for the error cases.

---

### 4. Interface Usage - `result.go:147-153`

**Rule**: `medium-accept-interface-return-struct`

**Impact**: `PrintResults` function accepts concrete types instead of interfaces, reducing testability.

**Current Code**:
```go
func PrintResults(summary TestSummary, colorOutput bool, currentJob job.Job) {
    spec, err := framework.Get(currentJob.Framework)
    // ...
}
```

**Assessment**: For this CLI application, this pattern is acceptable. The function is at the boundary of the application and doesn't need to be mocked in tests. No change recommended.

---

### 5. Long Parameter List - `stream_helper.go:40-47`

**Rule**: `medium-long-parameter-list`

**Impact**: Function has 7 parameters, which can be confusing.

**Current Code**:
```go
func streamTestOutput(
    stdout, stderr io.Reader,
    parser types.TestOutputParser,
    collector *TestCollector,
    outputChan chan<- OutputMessage,
    workerIndex int,
    streamStdout bool,
) (stderrOutput string) {
```

**Recommendation**: Consider a parameter object if more parameters are added:
```go
type StreamConfig struct {
    Stdout       io.Reader
    Stderr       io.Reader
    Parser       types.TestOutputParser
    Collector    *TestCollector
    OutputChan   chan<- OutputMessage
    WorkerIndex  int
    StreamStdout bool
}
```

---

## Files Reviewed

| File | Lines | Issues Found |
|------|-------|--------------|
| `plur/runner.go` | 369 | 2 (1 CRITICAL, 1 HIGH) |
| `plur/stream_helper.go` | 169 | 1 (MEDIUM) |
| `plur/watch/watcher.go` | 372 | 2 (1 HIGH, 1 MEDIUM) |
| `plur/watch/debouncer.go` | 60 | 0 |
| `plur/main.go` | 421 | 1 (CRITICAL) |
| `plur/cmd_watch.go` | 369 | 1 (MEDIUM) |
| `plur/result.go` | 215 | 0 |
| `plur/runtime_tracker.go` | 133 | 1 (MEDIUM) |

---

## Recommendations

### Immediate Actions (CRITICAL)

1. **Fix `%v` to `%w` in error wrapping** - Simple search and replace in `runner.go` and `main.go`

### Short-term Improvements (HIGH)

1. **Consider using `exec.CommandContext`** for better process cancellation support
2. **Review channel ownership** in watcher.go for clearer responsibility

### Long-term Considerations (MEDIUM)

1. **Extract `runWatchWithConfig`** into smaller functions as it grows
2. **Add named constants** for channel buffer sizes
3. **Add debug logging** for silent error returns in `runtime_tracker.go`

---

## Positive Observations

The codebase demonstrates several good practices and excellent use of modern Go 1.24/1.25 features:

### Modern Go Features (Go 1.24/1.25)

* **`sync.WaitGroup.Go()`** (Go 1.25) - Used correctly in `runner.go:181-193` and `stream_helper.go:55,127`. This [new Go 1.25 feature](https://appliedgo.net/spotlight/go-1.25-waitgroup-go/) eliminates the common `Add(1)`/`Done()` mismatch bugs by combining goroutine launch with WaitGroup tracking.

* **`os.OpenRoot`** (Go 1.24) - Used correctly in `watcher.go:296` for secure directory operations. This [Go 1.24 feature](https://go.dev/doc/go1.24) prevents path traversal attacks by confining file operations to a specific directory tree.

### General Best Practices

* **Proper channel closing** - `runner.go:196-198` correctly closes channels after `wg.Wait()`
* **Context usage in watcher** - `watcher.go` uses done channels and `stopOnce` for safe shutdown
* **Clean error messages** - Most error messages provide good context about what failed
* **Proper defer usage** - `debouncer.go:27` and other files use `defer` correctly for mutex unlocking
* **Bounded goroutines** - Worker pools have bounded concurrency based on worker count
* **Good separation of concerns** - Clear package boundaries between `watch/`, `config/`, `framework/`, etc.
* **Idiomatic signal handling** - `cmd_watch.go:208-209` uses the recommended buffer size 1 pattern for signal channels

---

*Generated using golang-best-practices-skill v2.0.0*

## References

* [Go 1.25 WaitGroup.Go() Feature](https://appliedgo.net/spotlight/go-1.25-waitgroup-go/)
* [Go 1.25 Release Notes](https://go.dev/doc/go1.25)
* [Go 1.24 Release Notes](https://go.dev/doc/go1.24)
