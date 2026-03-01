# Go Concurrency and Data Structures Review

Scope: Go module `github.com/rsanheim/plur`, with emphasis on goroutine lifecycle, channel usage, and Go-side data structures/abstractions. This is a code-informed review with concrete cleanup recommendations.

## Executive Summary

The concurrency surface area in the Go code is relatively small (a handful of goroutine spawning sites), which is good.

**Remaining (high priority)**

* [ ] Make `logger.CustomTextHandler` concurrency-safe as a `slog.Handler` (`logger/logger.go`).

There are also a number of medium-priority cleanup opportunities (watch rule matching duplication, stringly-typed enums) that should improve maintainability and reduce future correctness risk.

## Concurrency & Goroutine Lifecycle Audit

### 1) Parallel spec execution (`Runner`)

**Where goroutines are created**

* `Runner.executeWorkers()` spawns:
  * 1 goroutine for `outputAggregator(...)` (`runner.go`)
  * 1 goroutine per worker command (`runner.go`)
* Each worker goroutine calls `runCommand(...)`, which calls `streamTestOutput(...)` that spawns:
  * 1 goroutine to read stdout (`stream_helper.go`)
  * 1 goroutine to read stderr (`stream_helper.go`)

**Good**

* Uses `WaitGroup`s and closes `results` / `outputChan` in the right order (`runner.go:183-188`).
* Serializes progress output through a single aggregator goroutine (reduces lock contention on stdout).

**Moderate issue: context is currently unused**

* `executeWorkers()` creates `ctx := context.Background()` (`runner.go:157`) and passes it to `runCommand`, but it is not used to cancel/kick workers.
* In Go 1.25, `exec.Cmd` supports cancellation patterns via `exec.CommandContext` + `WaitDelay`/`Cancel` (depending on your chosen approach). Either remove the context parameter or use it meaningfully (esp. for timeouts / watch-mode cancellation / "stop on first failure" behaviors).

### 2) Output aggregation and progress typing

`OutputMessage.Type` is still a `string` (i.e., "magic strings" remain). Converting it to a small typed enum would further reduce mismatch risk.

### 3) Watch mode (`Watcher`, `WatcherManager`, `Debouncer`)

#### Debouncer concurrency and output interleaving

`Debouncer` uses `time.AfterFunc`, which runs the callback in its own goroutine (`watch/debouncer.go:40-58`). This means:

* Multiple runs can overlap if prior runs take longer than the debounce delay.
* Output from concurrent jobs can interleave (tracked in [#207](https://github.com/rsanheim/plur/issues/207)).
* `Timer.Stop()` is called but the code does not handle the "already fired / callback running" case; overlapping callback execution is possible.

Recommended fix:

* If the goal is "at most one job execution at a time", move to a single goroutine + queue model:
  * Debouncer just batches file paths.
  * A runner goroutine serializes `HandleBatch` executions.
* If the goal is "cancel in-flight job when new changes arrive", use `context.Context` with `exec.CommandContext` and/or process-group termination.

### 4) Logging concurrency (`slog.Handler`)

`CustomTextHandler` writes to its `io.Writer` without synchronization (`logger/logger.go:58-86`).

The `slog.Handler` contract requires handlers to be safe for concurrent use. Without a lock, log lines from different goroutines can interleave at the byte level.

Recommended fix:

* Add a `sync.Mutex` in `CustomTextHandler` and lock around the final write, or
* Use `slog.NewTextHandler` and customize via `HandlerOptions.ReplaceAttr` (keeps the built-in concurrency guarantees).

## Data Structures & Abstraction Review

### 1) Watch rule matching is duplicated in multiple places

* `EventProcessor.ProcessPath` does matching + ignore handling (`watch/processor.go:31-76`).
* `FindTargetsForFile` re-matches rules again (`watch/find.go:61-66`) via `matchesWatch`, with a comment acknowledging duplication (`watch/find.go:87-99`).

Recommended cleanup:

* Make the processor return both:
  * the `jobName -> targets` mapping, and
  * the list of matched `WatchMapping`s
  so `FindTargetsForFile` can be a thin wrapper without re-implementing matching semantics.

### 2) Notification type surface area looks larger than needed

The `types.TestNotification` interface + many concrete structs (`types/notifications.go`) is workable, but there are a few obvious simplifications:

* `GroupStartedNotification` appears unused outside `rspec/parser.go` and is ignored by the collector due to type mismatch (`types/notifications.go:110-117` + `rspec/parser.go:107-115` + `test_collector.go:49-61`).
  * If the only purpose is updating `CurrentFile`, then emitting a notification is unnecessary.
* "Formatted*Notification" types all report `RawOutput` and are distinguished only by Go type (`types/notifications.go:86-109`).
  * Consider one `FormattedOutputNotification{Kind, Content}` instead of three separate structs.

This reduction makes downstream collection simpler and reduces type-switching.

### 3) `FileGroup.TotalSize` is used as both bytes and milliseconds

`FileGroup.TotalSize` is a byte count in size-based grouping, but runtime grouping stores runtime milliseconds into the same field (`grouper.go:141-144`).

Recommended cleanup:

* Rename `TotalSize` to `TotalWeight` (or similar) to match its real semantics, or
* Split into explicit fields (`TotalBytes`, `TotalRuntimeMs`) if both are used elsewhere.

### 4) Stringly-typed enums and "magic strings"

Examples:

* `OutputMessage.Type string` (`result.go:37-42`)
* Watcher event `PathType` / `EffectType` strings (`watch/watcher.go:21-27`)

Recommended cleanup:

* Define small typed constants (e.g., `type OutputMessageType uint8`) with `const` values.
* This prevents accidental mismatch (like the `"error"` progress bug) and reduces allocations/comparisons.

### 5) "Set" maps should use `map[string]struct{}`

Current patterns use `map[string]bool` in several places:

* `watch.Deduplicate` (`watch/processor.go:115-127`)
* `Debouncer.pending` (`watch/debouncer.go:13-21`)

Using `struct{}` avoids storing an extra boolean per entry and communicates intent.

### 6) RSpec command building is potentially O(args * files) per insertion

`insertBeforeFiles` scans `args` and for each arg scans all `files` to find the file-arg boundary (`runner.go:353-377`), and is called multiple times by `buildRSpecCommand` (`runner.go:380-397`).

For large file lists this becomes disproportionately expensive.

Recommended cleanup:

* Have `BuildJobCmd` also return the file-start index (or ensure file args are always appended at the end for frameworks where we control the command shape).
* Or detect file-start by known structure (e.g., `{{target}}` placement) rather than nested scanning.

## Go 1.25-Specific Notes / Opportunities

* Go 1.25 makes it reasonable to standardize on `slices`/`maps` helpers (`maps.Clone`, `maps.Copy`, `slices.Compact`, etc.) to reduce handwritten map-copy loops (e.g., `RuntimeTracker.SaveToFile` in `runtime_tracker.go:65-73`).
* Loop variable capture footguns are substantially reduced in recent Go versions; a few "defensive" patterns (e.g., `workerIndex := i` in `database.go:37-65`) can be simplified if you're comfortable assuming 1.25+ semantics throughout.
* If you want robust subprocess cancellation, review use of `exec.CommandContext` plus modern `exec.Cmd` knobs (e.g., `WaitDelay`) so hangs are bounded even if a child ignores termination.

## Suggested Cleanup Sequence (No Back-Compat Assumed)

* [ ] Make logging handler concurrency-safe (`logger/logger.go`).
* [ ] Collapse duplicated watch matching logic (`watch/find.go`, `watch/processor.go`) and simplify notification types (`types/notifications.go`).
* [ ] Address structural cleanups (`FileGroup.TotalSize`, `insertBeforeFiles`).

## Validation Recommendations

* [ ] Add a small integration-ish test for minitest progress output to ensure `E/F/.` map correctly to on-screen glyphs.

Local note: if your system Ruby/Bundler is older than the version in `Gemfile.lock`, prefer `mise x ruby -- bin/rake ...` to run the rake tasks with the correct Ruby/Bundler.
