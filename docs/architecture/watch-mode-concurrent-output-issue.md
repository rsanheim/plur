# Watch Mode Concurrent Output Issue

## Problem Description

The current watch mode implementation allows concurrent test runs which can lead to interleaved output and multiple prompt displays. While functional, this creates a "janky" user experience.

## Current Behavior

When multiple file changes occur in quick succession:
1. Each change triggers a debounced test run (100ms delay)
2. The debouncer uses `time.AfterFunc` which spawns a new goroutine for each batch
3. Multiple test runs can execute concurrently
4. Output from different test runs gets interleaved
5. Multiple "rux> " prompts appear as each test run completes

### Example Timeline
```
Time 0ms:    spec/foo_spec.rb changes → Timer starts
Time 100ms:  Timer fires → Goroutine 1 starts running foo_spec.rb
Time 200ms:  spec/bar_spec.rb changes (while foo is still running)
Time 300ms:  Timer fires → Goroutine 2 starts running bar_spec.rb
Time 350ms:  Goroutine 1 completes → prints "rux> "
Time 400ms:  Goroutine 2 completes → prints "rux> "
```

## Root Cause

The debouncer implementation (`watch/debouncer.go`) uses `time.AfterFunc` which executes the callback in a new goroutine. There's no mechanism to prevent concurrent test executions or manage output synchronization.

## Potential Solutions

### 1. Queue-Based Approach
- Implement a test run queue that processes runs sequentially
- Only allow one active test run at a time
- Queue subsequent changes while tests are running

### 2. Mutex Protection
- Add a mutex to `runSpecsOrDirectory` to ensure serial execution
- Simple but might miss some file changes during long-running tests

### 3. Cancel Previous Runs
- Use `context.Context` to make test runs cancellable
- Cancel in-progress tests when new changes are detected
- More complex but provides better responsiveness

### 4. Terminal Management Library
- Use a library like `termbox-go`, `tview`, or `bubbletea`
- Proper cursor management and screen regions
- Clean separation of output areas
- Most robust solution but adds dependencies

## Current Workaround

The prompt is reprinted after each test run with a small delay (50ms) to allow output to flush. This works but can result in multiple prompts appearing when concurrent runs complete.

## Decision

For now, we're accepting this behavior as "good enough" for the watch mode MVP. The functionality works correctly even if the output is sometimes confusing. This can be revisited when watch mode stability is proven and user feedback indicates this is a priority issue.

## Related Files
- `/rux/watch.go` - Main watch mode implementation
- `/rux/watch/debouncer.go` - Debouncer that allows concurrent execution
- Lines 201-204 in watch.go - Prompt reprint logic