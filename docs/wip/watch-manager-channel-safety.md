# Watch Manager Channel Safety

Fix channel closure handling in WatcherManager to prevent spinning, goroutine leaks, and deadlocks.

## Context

The watch subsystem has several channel safety issues that can cause problems during shutdown or error conditions:

**Problem 1: `errorChan` closure ownership is wrong/unclear**

In `plur/watch/watcher.go`, `readEvents()` is the only goroutine that writes to `w.errorChan` (scanner/IO errors while reading JSON events). `readErrors()` only logs stderr lines to `os.Stderr`.

That means:
* `readErrors()` should not close `w.errorChan` (it is not a sender)
* `w.errorChan` should be closed by the sender (`readEvents()`), alongside `eventChan`, to signal the watcher is fully done producing
* Without closure (and without handling `ok` on receives), consumers can spin on closed channels or block indefinitely waiting for errors that will never arrive

**Problem 2: `Start()` error path doesn't fully clean up**

`plur/watch/watcher_manager.go:53-56` - If a later watcher fails to start:
* `cleanup()` only stops watchers that successfully started
* But aggregator goroutines for successful watchers may still be running
* Those goroutines can spin or block when watcher channels close

**Problem 3: `Watcher.Stop()` lacks idempotency**

`plur/watch/watcher.go:110-113` - Unlike `WatcherManager.Stop()` which uses `sync.Once`:
* `Watcher.Stop()` has no protection against double-close
* Calling `Stop()` twice panics: "close of closed channel"
* Also blocks forever if `Start()` was never called (nothing closes `done`)

**Related Fix (already done):**

Commit `cb0c32a` fixed a race condition where `reload()` was called from debouncer goroutine while main select loop was still reading. This is now handled via `reloadChan` signaling.

## Success Criteria

* [ ] `readEvents()` closes `errorChan` on exit (sender owns closure; matches `eventChan` handling)
* [ ] `Start()` error path ensures no goroutine leaks
* [ ] `Watcher.Stop()` is idempotent and safe to call without prior `Start()`
* [ ] `PLUR_RACE=1 bin/rake test:go` passes with no race conditions
* [ ] New unit tests verify channel closure behavior
* [ ] Watch mode still functions correctly in normal operation

## Task List

### Phase 1: Fix watcher channel closure

* [ ] Ensure `w.eventChan` and `w.errorChan` are both closed by `readEvents()` (the sender)
* [ ] Avoid closing `w.errorChan` from `readErrors()` unless it actually becomes a sender
* [ ] Add a test that `WatcherManager.aggregateEvents()` does not spin on closed channels

### Phase 2: Make Watcher.Stop() idempotent

* [ ] Add `stopOnce sync.Once` field to `Watcher` struct
* [ ] Wrap `Stop()` body in `w.stopOnce.Do()`
* [ ] Handle case where `Stop()` called before `Start()`:
  * Option A: Close `done` channel in constructor, re-signal via different mechanism
  * Option B: Add `started` flag and guard the `<-w.done` wait
* [ ] Add test for double-stop (should not panic)
* [ ] Add test for stop-without-start (should not block)

### Phase 3: Fix Start() error path cleanup

* [ ] When a watcher fails to start, call `wm.Stop()` instead of just `wm.cleanup()`
* [ ] Alternatively: close `wm.stopChan` and wait for `wm.wg` in the error path
* [ ] Ensure aggregator goroutines from successful watchers exit cleanly
* [ ] Add test simulating partial start failure

### Phase 4: Validation

* [ ] Run `PLUR_RACE=1 bin/rake test:go`
* [ ] Run `bin/rake test` (full Ruby integration suite)
* [ ] Manual testing of watch mode with file changes
* [ ] Manual testing of watch mode with rapid start/stop cycles

## Files to Modify

* `plur/watch/watcher.go` - Add channel closure, idempotent stop
* `plur/watch/watcher_manager.go` - Fix Start() error path
* `plur/watch/watcher_test.go` (new or existing) - Add channel safety tests
