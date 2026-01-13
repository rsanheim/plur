# Go Concurrency and Data Structures Review

Scope: `plur/` (Go module `github.com/rsanheim/plur`), with emphasis on goroutine lifecycle, channel usage, and Go-side data structures/abstractions. This is a code-informed review with concrete cleanup recommendations.

## Executive Summary

The concurrency surface area in the Go code is relatively small (a handful of goroutine spawning sites), which is good. However, there are a few correctness and “hang risk” issues that should be treated as high priority:

- `watch/WatcherManager.aggregateEvents()` does not handle closed watcher channels and can spin / emit zero-value events (and in some error paths, can deadlock).
- `streamTestOutput()` uses `bufio.Scanner` for subprocess pipes; if a line exceeds the scanner limit, scanning stops and the pipes may no longer be drained, which can hang the worker process.
- Minitest progress `"E"` is mapped to `"error"` but the runner output aggregator treats `"error"` as “print msg.Content to stderr” (blank lines), not a progress glyph.
- `logger.CustomTextHandler` is not concurrency-safe as a `slog.Handler`, so logs from multiple goroutines can interleave.

Separately, there are a number of duplication/abstraction issues (mostly in watch-mode plumbing and notification types) that are good cleanup targets and should also improve maintainability and performance predictability.

## Concurrency & Goroutine Lifecycle Audit

### 1) Parallel spec execution (`Runner`)

**Where goroutines are created**

- `Runner.executeWorkers()` spawns:
  - 1 goroutine for `outputAggregator(...)` (`plur/runner.go:166-171`)
  - 1 goroutine per worker command (`plur/runner.go:173-181`)
- Each worker goroutine calls `runCommand(...)`, which calls `streamTestOutput(...)` that spawns:
  - 1 goroutine to scan stdout (`plur/stream_helper.go:51-107`)
  - 1 goroutine to scan stderr (`plur/stream_helper.go:110-130`)

**Good**

- Uses `WaitGroup`s and closes `results` / `outputChan` in the right order (`plur/runner.go:183-188`).
- Serializes progress output through a single aggregator goroutine (reduces lock contention on stdout).

**High-risk issue: `bufio.Scanner` can cause hangs**

- `bufio.Scanner` stops on token-too-long (`scanner.Err() == bufio.ErrTooLong`). When that happens, the goroutine exits and the corresponding pipe may no longer be drained.
- If the child process continues to write to that pipe (stdout or stderr), it can block on a full OS pipe buffer and never exit → `cmd.Wait()` blocks forever.
- This is a known failure mode for scanner-based subprocess capture.

Code locations:

- Stdout scan loop: `plur/stream_helper.go:54-106`
- Stderr scan loop: `plur/stream_helper.go:113-129`

Recommended fix:

- Replace `bufio.Scanner` with a `bufio.Reader` and a bounded `ReadString('\n')`/`ReadBytes('\n')` loop (or a custom split function that keeps draining even on long lines).
- If line-length bounds are important, enforce them without stopping the drain (e.g., truncate logged/collected content but continue reading and discarding remainder until newline).

**Moderate issue: context is currently unused**

- `executeWorkers()` creates `ctx := context.Background()` (`plur/runner.go:157`) and passes it to `runCommand`, but it is not used to cancel/kick workers.
- In Go 1.25, `exec.Cmd` supports cancellation patterns via `exec.CommandContext` + `WaitDelay`/`Cancel` (depending on your chosen approach). Either remove the context parameter or use it meaningfully (esp. for timeouts / watch-mode cancellation / “stop on first failure” behaviors).

### 2) Output aggregation and progress typing

**High-risk correctness issue: minitest `"E"` progress is misrouted**

- Minitest parser maps `'E'` to `"error"` (`plur/minitest/output_parser.go:47-58`).
- Runner output aggregator treats `"error"` as “print msg.Content to stderr” (`plur/runner.go:325-327`), which will print blank lines for progress events (because `Content` is empty for progress).

Recommended fix:

- Either:
  - Add a distinct progress type (e.g., `"error_progress"`) and handle it in `outputAggregator` by printing `E`, or
  - Map `'E'` to `"failure"` until a first-class error glyph is supported, or
  - Teach `outputAggregator` to interpret `"error"` with empty content as a progress glyph (but that overload is brittle).

Also consider replacing `OutputMessage.Type string` with a small typed enum to remove “magic strings” and avoid this class of mismatch.

### 3) Watch mode (`Watcher`, `WatcherManager`, `Debouncer`)

#### High-risk correctness issue: channel closure handling in `WatcherManager`

`Watcher.readEvents()` closes `w.eventChan` (`plur/watch/watcher.go:126-129`).

`WatcherManager.aggregateEvents()` receives from `w.Events()` / `w.Errors()` without checking the `ok` value (`plur/watch/watcher_manager.go:99-121`):

- Receiving from a closed channel returns immediately with the zero value.
- That makes the select-case “always ready”, creating a tight loop.
- The loop can emit zero-value `Event{}` into `wm.eventChan` and/or spin CPU.
- In some cases (especially when no consumer is draining `wm.eventChan`), this can deadlock while attempting to send.

This also interacts badly with the `Start()` error path:

- `Start()` launches aggregate goroutines as it starts watchers (`plur/watch/watcher_manager.go:63-66`).
- If a later watcher fails to start, `wm.cleanup()` is called but `wm.stopChan` is not closed and `wm.eventChan` is not closed (because `Stop()` wasn’t called).
- Any aggregator goroutine attached to a watcher whose `eventChan` is now closed can spin and then block forever.

Recommended fix:

- In `aggregateEvents`, use `event, ok := <-w.Events()` / `err, ok := <-w.Errors()` and exit (or nil the channel) when `ok == false`.
- Ensure `Watcher` closes both `eventChan` and `errorChan`, or make the manager robust to either being left open.
- In `Start()` failure path, call `wm.Stop()` (or explicitly close `wm.stopChan` + wait for `wm.wg`) so goroutines cannot leak.

#### Watcher lifecycle edge cases

- `Watcher.Stop()` closes `w.stopChan` without `sync.Once` (`plur/watch/watcher.go:110-113`), so double-stop panics.
- `Watcher.Stop()` blocks on `<-w.done` even if `Start()` was never called (no goroutine will ever close `done`).

Recommended fix:

- Make `Stop()` idempotent (`sync.Once`) and safe pre-Start (either close `done` in constructor and re-open on Start, or guard Stop with a started flag and close `done` on Start failure).

#### Debouncer concurrency and output interleaving

`Debouncer` uses `time.AfterFunc`, which runs the callback in its own goroutine (`plur/watch/debouncer.go:40-58`). This means:

- Multiple runs can overlap if prior runs take longer than the debounce delay.
- Output from concurrent jobs can interleave (documented in `docs/architecture/watch-concurrent-output.md`).
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

### 7) Config template output appears out of sync with current config schema

`plur config init` templates in `plur/config_init.go` use keys like `[spec] command = ...` and `[watch.run] command = ...`, which don’t match the “job + watch mappings” schema described elsewhere in the repo.

Recommended cleanup:

- Update `config_init.go` templates to generate `.plur.toml` that matches the current `job` and `watch` structures.

### 8) Docs/code drift in concurrency documentation

`docs/architecture/concurrency-model.md` describes a worker pool with “job channel” and “unbuffered result channel”, but current implementation pre-groups files and spawns one goroutine per group; there is no job channel in the hot path.

Recommended cleanup:

- Update the doc to reflect the current implementation to prevent future refactors from being guided by stale assumptions.

## Go 1.25-Specific Notes / Opportunities

- Go 1.25 makes it reasonable to standardize on `slices`/`maps` helpers (`maps.Clone`, `maps.Copy`, `slices.Compact`, etc.) to reduce handwritten map-copy loops (e.g., `RuntimeTracker.SaveToFile` in `plur/runtime_tracker.go:65-73`).
- Loop variable capture footguns are substantially reduced in recent Go versions; a few “defensive” patterns (e.g., `workerIndex := i` in `plur/database.go:37-65`) can be simplified if you’re comfortable assuming 1.25+ semantics throughout.
- If you want robust subprocess cancellation, review use of `exec.CommandContext` plus modern `exec.Cmd` knobs (e.g., `WaitDelay`) so hangs are bounded even if a child ignores termination.

## Suggested Cleanup Sequence (No Back-Compat Assumed)

- [x] Fix watch manager channel closure + Start error-path leaks (`plur/watch/watcher_manager.go`, `plur/watch/watcher.go`).
- [ ] Replace scanner-based pipe reading to remove subprocess hang risk (`plur/stream_helper.go`).
- [x] Fix minitest progress mapping vs output aggregator typing (`plur/minitest/output_parser.go`, `plur/runner.go`).
- [ ] Make logging handler concurrency-safe (`plur/logger/logger.go`).
- [ ] Collapse duplicated watch matching logic (`plur/watch/find.go`, `plur/watch/processor.go`) and simplify notification types (`plur/types/notifications.go`).
- [ ] Address structural cleanups (`FileGroup.TotalSize`, config templates, `insertBeforeFiles`) and update docs where they drifted.

## Validation Recommendations

- [ ] Run Go tests under the race detector: `PLUR_RACE=1 bin/rake test:go`.
- [x] Add a focused unit test for `WatcherManager.aggregateEvents` to ensure it exits cleanly when a watcher channel closes (and does not emit zero-value events).
- [ ] Add a small integration-ish test for minitest progress output to ensure `E/F/.` map correctly to on-screen glyphs.

## Follow-up Notes (Current Changes)

- Minitest: progress `"E"` now maps to `"error_progress"` and is rendered as a progress glyph (instead of printing a blank stderr line).
- Watcher/WatcherManager: channel closure + shutdown are now safe and idempotent:
  - `Watcher.Stop()` is safe to call multiple times and safe before `Start()`.
  - `readEvents()` owns closing both `eventChan` and `errorChan` (the sender closes), and `aggregateEvents()` exits cleanly when watcher channels close (no spinning).
  - `WatcherManager.Start()` failure now calls `wm.Stop()` (instead of `cleanup()`), preventing goroutine leaks on partial startup.
- Local validation note:
  - `bin/rake test:go` requires Bundler 4+ (per `Gemfile.lock`) and may fail if your Ruby/Bundler is older.
  - `go test ./...` can be run by setting `GOCACHE`, `GOMODCACHE`, and `GOPATH` to paths under `./tmp/` if the default Go cache directory is not writable in your environment.
