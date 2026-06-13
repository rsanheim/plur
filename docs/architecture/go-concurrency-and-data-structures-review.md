# Go Concurrency and Data Structures Review

Scope: Go module `github.com/rsanheim/plur`, with emphasis on goroutine lifecycle, channel usage, and Go-side data structures/abstractions. This is a code-informed review with concrete cleanup recommendations.

## Executive Summary

The concurrency surface area in the Go code is relatively small: worker
execution, stream draining, watch-process readers, and the debounce callback.
The remaining items are mostly medium-priority cleanup opportunities around
cancellation, watch-mode serialization, and stringly-typed message surfaces.

## Concurrency & Goroutine Lifecycle Audit

### 1) Parallel spec execution (`Runner`)

**Where goroutines are created**

* `Runner.executeWorkers()` spawns:
  * 1 goroutine for `outputAggregator(...)` (`runner.go:285-288`)
  * 1 goroutine per worker command (`runner.go:290-298`)
* Each worker goroutine calls `runCommand(...)`, which calls `streamTestOutput(...)` that spawns:
  * 1 goroutine to read stdout (`stream_helper.go:51-115`)
  * 1 goroutine to read stderr (`stream_helper.go:117-153`)

**Good**

* Uses `WaitGroup`s and closes `results` / `outputChan` in the right order (`runner.go:300-304`).
* Serializes progress output through a single aggregator goroutine (reduces lock contention on stdout).

**Moderate issue: spec worker cancellation is not wired**

* `executeWorkers()` creates `ctx := context.Background()` (`runner.go:278-281`) and passes it to `runCommand`, but `runCommand` does not use it (`runner.go:321-342`).
* `buildArgsPerWorkerCommands()` does use `exec.CommandContext` for Rails/Rake per-worker commands (`runner.go:222-239`), so cancellation behavior is inconsistent across command paths.

Recommended fix:

* If spec-worker cancellation is not a near-term behavior, remove the unused context parameter from `runCommand`.
* If watch-mode cancellation, timeout, or "stop on first failure" is desired, build spec worker commands with `exec.CommandContext` and decide how to terminate child processes that ignore cancellation.

### 2) Output aggregation and progress typing

`OutputMessage.Type` is still a `string` (`result.go:41-46`) and `outputAggregator` switches on string literals (`runner.go:397-436`). Parsers also return progress types as strings through `types.TestOutputParser.NotificationToProgress` (`types/parser.go:8-10`).

Recommended cleanup:

* Define a small typed enum for output message types, for example `type OutputMessageType uint8`.
* Return that type from `NotificationToProgress` instead of a raw string.
* Keep the string literals at framework boundaries only, not across the internal output pipeline.

### 3) Watch mode (`Watcher`, `WatcherManager`, `Debouncer`)

#### Debouncer concurrency and output interleaving

`Debouncer` uses `time.AfterFunc`, which runs the callback in its own goroutine (`watch/debouncer.go:23-50`). This means:

* Multiple runs can overlap if prior runs take longer than the debounce delay.
* Output from concurrent jobs can interleave (tracked in [#207](https://github.com/rsanheim/plur/issues/207)).
* `Timer.Stop()` is called (`watch/debouncer.go:31-33`) but the code does not handle the "already fired / callback running" case; overlapping callback execution is still possible.

Recommended fix:

* If the goal is "at most one job execution at a time", move to a single goroutine + queue model:
  * Debouncer just batches file paths.
  * A runner goroutine serializes `Planner.Plan` + `ExecuteJob` executions.
* If the goal is "cancel in-flight job when new changes arrive", use `context.Context` with `exec.CommandContext` and process-group termination where needed.

## Data Structures & Abstraction Review

### 1) Notification type surface area looks larger than needed

The `types.TestNotification` interface + many concrete structs (`types/notifications.go`) is workable, but there are a few obvious simplifications:

* RSpec tracks `CurrentFile` directly inside `internal/framework/rspec/parser.go:13-26` and consumes `group_started` without emitting a notification (`internal/framework/rspec/parser.go:119-124`).
* `FormattedFailuresNotification`, `FormattedPendingNotification`, and `FormattedSummaryNotification` all report `RawOutput` and are distinguished only by Go type (`types/notifications.go:92-114`).
  * Consider one `FormattedOutputNotification{Kind, Content}` instead of three separate structs.

This reduction makes downstream collection simpler and reduces type-switching.

### 2) Stringly-typed enums and "magic strings"

Examples:

* `OutputMessage.Type string` (`result.go:41-46`)
* Watcher event `PathType` / `EffectType` strings (`watch/watcher.go:23-29`)

Recommended cleanup:

* Define small typed constants (e.g., `type OutputMessageType uint8`) with `const` values.
* This prevents accidental mismatch (like the `"error"` progress bug) and reduces allocations/comparisons.

### 3) "Set" maps should use `map[string]struct{}`

Current patterns use `map[string]bool` in several places:

* `Planner.buildRuns` tracks seen jobs with `map[string]bool` (`watch/plan.go:132-142`).
* `deduplicate` uses `map[string]bool` (`watch/plan.go:243-255`).
* `Debouncer.pending` uses `map[string]bool` (`watch/debouncer.go:8-18`, `watch/debouncer.go:27-43`).

Using `struct{}` avoids storing an extra boolean per entry and communicates intent.

### 4) Keep framework command building append-based

Run-mode command building now keeps file targets appended at the end of the
command shape controlled by `internal/framework.Job.BuildRunArgs`. That avoids the old
file-argument boundary scan that was previously used when inserting framework
args.

Recommended cleanup:

* Keep future framework-specific args before targets so command construction
  remains linear and easy to inspect.

## Go 1.25-Specific Notes / Opportunities

* Go 1.25 makes it reasonable to keep standardizing on `slices`/`maps` helpers (`maps.Clone`, `maps.Copy`, `slices.Compact`, etc.). This is already happening in places like `RuntimeTracker.PendingFileRuntimes()` (`internal/testruntime/tracker.go:142-145`); apply the same style opportunistically when touching older loops.
* If spec-worker cancellation becomes a product requirement, review use of `exec.CommandContext` plus modern `exec.Cmd` knobs (e.g., `WaitDelay`) so hangs are bounded even if a child ignores termination.

## Suggested Cleanup Sequence (No Back-Compat Assumed)

* [ ] Simplify notification types (`types/notifications.go`).
* [ ] Replace output/progress magic strings with typed constants (`result.go`, `runner.go`, `types/parser.go`).
* [ ] Decide whether spec-worker contexts should be removed or wired into cancellation (`runner.go`).
* [ ] Serialize or cancel overlapping watch job executions (`watch/debouncer.go`).
* [ ] Convert string set maps from `map[string]bool` to `map[string]struct{}` (`watch/plan.go`, `watch/debouncer.go`).

Local note: if your system Ruby/Bundler is older than the version in `Gemfile.lock`, prefer `mise x ruby -- bin/rake ...` to run the rake tasks with the correct Ruby/Bundler.
