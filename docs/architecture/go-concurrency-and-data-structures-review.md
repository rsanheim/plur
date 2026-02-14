# Go Concurrency and Data Structures Review

Scope: `plur/` (Go module `github.com/rsanheim/plur`), with emphasis on goroutine lifecycle, channel usage, and Go-side data structures/abstractions. This is a code-informed review with concrete cleanup recommendations.

## Executive Summary

The concurrency surface area in the Go code is relatively small (a handful of goroutine spawning sites), which is good.
The original review identified a few hang/correctness risks; the highest priority items are now addressed on this branch.

**Completed (high priority)**

- [x] Watch mode: `WatcherManager` channel closure handling + `Start()` failure cleanup + idempotent shutdown (`plur/watch/watcher_manager.go`, `plur/watch/watcher.go`).
- [x] Subprocess output draining: replace `bufio.Scanner` with `bufio.Reader` to remove token-limit hang risk + add a long-line hang regression test (`plur/stream_helper.go`, `plur/stream_helper_test.go`).
- [x] Minitest progress: map `E` to `error_progress` and render as a progress glyph (`plur/minitest/output_parser.go`, `plur/runner.go`).

**Remaining (high priority)**

- [ ] Make `logger.CustomTextHandler` concurrency-safe as a `slog.Handler` (`plur/logger/logger.go`).

There are also a number of medium-priority cleanup opportunities (watch rule matching duplication, stringly-typed enums, config/doc drift) that should improve maintainability and reduce future correctness risk.

## Concurrency & Goroutine Lifecycle Audit

### 1) Parallel spec execution (`Runner`)

**Where goroutines are created**

- `Runner.executeWorkers()` spawns:
  - 1 goroutine for `outputAggregator(...)` (`plur/runner.go`)
  - 1 goroutine per worker command (`plur/runner.go`)
- Each worker goroutine calls `runCommand(...)`, which calls `streamTestOutput(...)` that spawns:
  - 1 goroutine to read stdout (`plur/stream_helper.go`)
  - 1 goroutine to read stderr (`plur/stream_helper.go`)

**Good**

- Uses `WaitGroup`s and closes `results` / `outputChan` in the right order (`plur/runner.go:183-188`).
- Serializes progress output through a single aggregator goroutine (reduces lock contention on stdout).

**Resolved (high priority): subprocess pipe draining hang risk**

Previously, `streamTestOutput()` used `bufio.Scanner`, which can stop draining on `bufio.ErrTooLong` (token limit).
If a child process keeps writing after that, it can block on a full pipe and hang `cmd.Wait()`.

Fix applied:

- `streamTestOutput()` now uses `bufio.Reader.ReadString('\n')` for both stdout and stderr and continues draining until EOF (`plur/stream_helper.go`).
- Added a regression test that spawns a helper process emitting a >256KB single line and asserts no hang (`plur/stream_helper_test.go`).

**Moderate issue: context is currently unused**

- `executeWorkers()` creates `ctx := context.Background()` (`plur/runner.go:157`) and passes it to `runCommand`, but it is not used to cancel/kick workers.
- In Go 1.25, `exec.Cmd` supports cancellation patterns via `exec.CommandContext` + `WaitDelay`/`Cancel` (depending on your chosen approach). Either remove the context parameter or use it meaningfully (esp. for timeouts / watch-mode cancellation / “stop on first failure” behaviors).

### 2) Output aggregation and progress typing

**Resolved (high priority): minitest `"E"` progress rendering**

Minitest progress now maps `E` to `error_progress`, and `outputAggregator` renders it as a progress glyph (instead of routing it through the `"error"` stderr path).

Note: `OutputMessage.Type` is still a `string` (i.e., “magic strings” remain). Converting it to a small typed enum would further reduce mismatch risk.

### 3) Watch mode (`Watcher`, `WatcherManager`, `Debouncer`)

#### Resolved (high priority): channel closure + lifecycle safety in watch mode

Fixes applied:

- `WatcherManager.aggregateEvents()` now checks the `ok` value when receiving and nils closed channels to prevent spinning / zero-value events (`plur/watch/watcher_manager.go`).
- `WatcherManager.Start()` calls `Stop()` on failure, ensuring partial startup can’t leak goroutines (`plur/watch/watcher_manager.go`).
- `Watcher.Stop()` is idempotent and safe pre-Start, and `Watcher.readEvents()` owns channel closure (`plur/watch/watcher.go`).
- `Watcher.readEvents()`/`readErrors()` now use `bufio.Reader` to avoid Scanner token-limit hangs on large watcher output (`plur/watch/watcher.go`).

#### Watcher lifecycle edge cases

Status: resolved (covered by the changes above).

#### Debouncer concurrency and output interleaving

`Debouncer` uses `time.AfterFunc`, which runs the callback in its own goroutine (`plur/watch/debouncer.go:40-58`). This means:

- Multiple runs can overlap if prior runs take longer than the debounce delay.
- Output from concurrent jobs can interleave (tracked in [#207](https://github.com/rsanheim/plur/issues/207)).
- `Timer.Stop()` is called but the code does not handle the “already fired / callback running” case; overlapping callback execution is possible.

Recommended fix:

- If the goal is “at most one job execution at a time”, move to a single goroutine + queue model:
  - Debouncer just batches file paths.
  - A runner goroutine serializes `HandleBatch` executions.
- If the goal is “cancel in-flight job when new changes arrive”, use `context.Context` with `exec.CommandContext` and/or process-group termination.

### 4) Logging concurrency (`slog.Handler`)

`CustomTextHandler` writes to its `io.Writer` without synchronization (`plur/logger/logger.go:58-86`).

The `slog.Handler` contract requires handlers to be safe for concurrent use. Without a lock, log lines from different goroutines can interleave at the byte level.

Recommended fix:

- Add a `sync.Mutex` in `CustomTextHandler` and lock around the final write, or
- Use `slog.NewTextHandler` and customize via `HandlerOptions.ReplaceAttr` (keeps the built-in concurrency guarantees).

## Data Structures & Abstraction Review

### 1) Watch rule matching is duplicated in multiple places

- `EventProcessor.ProcessPath` does matching + ignore handling (`plur/watch/processor.go:31-76`).
- `FindTargetsForFile` re-matches rules again (`plur/watch/find.go:61-66`) via `matchesWatch`, with a comment acknowledging duplication (`plur/watch/find.go:87-99`).

Recommended cleanup:

- Make the processor return both:
  - the `jobName -> targets` mapping, and
  - the list of matched `WatchMapping`s
  so `FindTargetsForFile` can be a thin wrapper without re-implementing matching semantics.

### 2) Notification type surface area looks larger than needed

The `types.TestNotification` interface + many concrete structs (`plur/types/notifications.go`) is workable, but there are a few obvious simplifications:

- `GroupStartedNotification` appears unused outside `rspec/parser.go` and is ignored by the collector due to type mismatch (`plur/types/notifications.go:110-117` + `plur/rspec/parser.go:107-115` + `plur/test_collector.go:49-61`).
  - If the only purpose is updating `CurrentFile`, then emitting a notification is unnecessary.
- “Formatted*Notification” types all report `RawOutput` and are distinguished only by Go type (`plur/types/notifications.go:86-109`).
  - Consider one `FormattedOutputNotification{Kind, Content}` instead of three separate structs.

This reduction makes downstream collection simpler and reduces type-switching.

### 3) `FileGroup.TotalSize` is used as both bytes and milliseconds

`FileGroup.TotalSize` is a byte count in size-based grouping, but runtime grouping stores runtime milliseconds into the same field (`plur/grouper.go:141-144`).

Recommended cleanup:

- Rename `TotalSize` to `TotalWeight` (or similar) to match its real semantics, or
- Split into explicit fields (`TotalBytes`, `TotalRuntimeMs`) if both are used elsewhere.

### 4) Stringly-typed enums and “magic strings”

Examples:

- `OutputMessage.Type string` (`plur/result.go:37-42`)
- Watcher event `PathType` / `EffectType` strings (`plur/watch/watcher.go:21-27`)

Recommended cleanup:

- Define small typed constants (e.g., `type OutputMessageType uint8`) with `const` values.
- This prevents accidental mismatch (like the `"error"` progress bug) and reduces allocations/comparisons.

### 5) “Set” maps should use `map[string]struct{}`

Current patterns use `map[string]bool` in several places:

- `watch.Deduplicate` (`plur/watch/processor.go:115-127`)
- `Debouncer.pending` (`plur/watch/debouncer.go:13-21`)

Using `struct{}` avoids storing an extra boolean per entry and communicates intent.

### 6) RSpec command building is potentially O(args * files) per insertion

`insertBeforeFiles` scans `args` and for each arg scans all `files` to find the file-arg boundary (`plur/runner.go:353-377`), and is called multiple times by `buildRSpecCommand` (`plur/runner.go:380-397`).

For large file lists this becomes disproportionately expensive.

Recommended cleanup:

- Have `BuildJobCmd` also return the file-start index (or ensure file args are always appended at the end for frameworks where we control the command shape).
- Or detect file-start by known structure (e.g., `{{target}}` placement) rather than nested scanning.

### 7) Config template output appears out of sync with current config schema (**resolved**)

`plur config init` templates now generate `job` + `watch` mappings that match the current config schema.

### 8) Docs/code drift in concurrency documentation

The former `docs/architecture/concurrency-model.md` described a worker pool with "job channel" and "unbuffered result channel" that no longer matched the implementation. That doc has been removed. The current implementation pre-groups files and spawns one goroutine per group; there is no job channel in the hot path. This review document is the canonical concurrency reference.

## Go 1.25-Specific Notes / Opportunities

- Go 1.25 makes it reasonable to standardize on `slices`/`maps` helpers (`maps.Clone`, `maps.Copy`, `slices.Compact`, etc.) to reduce handwritten map-copy loops (e.g., `RuntimeTracker.SaveToFile` in `plur/runtime_tracker.go:65-73`).
- Loop variable capture footguns are substantially reduced in recent Go versions; a few “defensive” patterns (e.g., `workerIndex := i` in `plur/database.go:37-65`) can be simplified if you’re comfortable assuming 1.25+ semantics throughout.
- If you want robust subprocess cancellation, review use of `exec.CommandContext` plus modern `exec.Cmd` knobs (e.g., `WaitDelay`) so hangs are bounded even if a child ignores termination.

## Suggested Cleanup Sequence (No Back-Compat Assumed)

**Completed**

- [x] Watch mode: channel closure + Start cleanup + idempotent shutdown (`plur/watch/watcher_manager.go`, `plur/watch/watcher.go`).
- [x] Subprocess output draining hang prevention (`plur/stream_helper.go`).
- [x] Minitest progress mapping / output typing mismatch (`plur/minitest/output_parser.go`, `plur/runner.go`).

**Next**

- [ ] Make logging handler concurrency-safe (`plur/logger/logger.go`).
- [ ] Collapse duplicated watch matching logic (`plur/watch/find.go`, `plur/watch/processor.go`) and simplify notification types (`plur/types/notifications.go`).
- [ ] Address structural cleanups (`FileGroup.TotalSize`, config templates, `insertBeforeFiles`) and update docs where they drifted.

## Validation Recommendations

- [x] Run Go tests under the race detector: `PLUR_RACE=1 mise x ruby -- bin/rake test:go`.
- [x] Add a focused unit test for `WatcherManager.aggregateEvents` to ensure it exits cleanly when a watcher channel closes (and does not emit zero-value events).
- [x] Add a regression test for long-line subprocess output to ensure `streamTestOutput()` always drains pipes (`plur/stream_helper_test.go`).
- [ ] Add a small integration-ish test for minitest progress output to ensure `E/F/.` map correctly to on-screen glyphs.

Local note: if your system Ruby/Bundler is older than the version in `Gemfile.lock`, prefer `mise x ruby -- bin/rake ...` to run the rake tasks with the correct Ruby/Bundler.
