# Watch Mode Config & Output Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix three watch-mode bugs: config merge (user watches replace defaults), duplicate failure warnings, and prompt/command rendering.

**Architecture:** All fixes are minimal changes to existing files. No new abstractions or files. Fix config merge in `loadWatchConfiguration`, remove duplicate WARN from `ExecuteJob`, and change command echo format to continue the prompt line.

**Tech Stack:** Go, testify (assert/require), RSpec integration tests

---

## File Structure

| File | Change | Purpose |
|------|--------|---------|
| `cmd_watch.go` | Modify lines 52-56 | Merge user + default watches instead of replacing |
| `cmd_watch.go` | Modify lines 313-314 | Clear prompt line before debug logging |
| `cmd_watch_test.go` | Already exists (line 11) | Existing failing test for merge behavior |
| `watch/watcher.go` | Modify lines 237, 246, 258, 267 | Remove `[plur]` prefix from command echo; remove duplicate WARN |
| `watch/watcher_test.go` | Add test | Test that `ExecuteJob` does not log WARN on failure |
| `watch/file_event_handler.go` | No change | Keeps its WARN on line 109 (single place for failure logging) |
| `spec/integration/plur_watch/watch_integration_spec.rb` | Already exists (line 15) | Existing failing test for config merge |
| `autodetect/defaults.go` | Modify lines 292-295 | Update `ValidateConfig` to validate merged watch set |

---

### Task 1: Fix watch config merge — make user watches additive

The core bug. `loadWatchConfiguration` uses user watches OR defaults, never both.

**Files:**
* Modify: `cmd_watch.go:52-56`
* Test: `cmd_watch_test.go` (existing test `TestLoadWatchConfigurationMergesUserAndDefaultWatches`)

- [ ] **Step 1: Run the existing failing Go test to confirm it fails**

Run: `mise exec -- go test . -run TestLoadWatchConfigurationMergesUserAndDefaultWatches -v`
Expected: FAIL — watches contains only `custom-config-watch`, missing `lib-to-spec` and `spec-files`

- [ ] **Step 2: Run the existing failing Ruby integration test to confirm it fails**

Run: `mise exec -- bin/rspec spec/integration/plur_watch/watch_integration_spec.rb -e "keeps default watches when user config adds a custom watch mapping"`
Expected: FAIL — `plur watch find lib/calculator.rb` exits with code 2, `found rules count=0`

- [ ] **Step 3: Fix `loadWatchConfiguration` to merge watches**

In `cmd_watch.go`, replace lines 52-56:

```go
	// Use user's watches if provided, else from resolved result
	watches := cli.WatchMappings
	if len(watches) == 0 {
		watches = result.Watches
	}
```

With:

```go
	// Merge built-in watches for the resolved job with any user-defined watches
	watches := result.Watches
	watches = append(watches, cli.WatchMappings...)
```

- [ ] **Step 4: Run the Go test to verify it passes**

Run: `mise exec -- go test . -run TestLoadWatchConfigurationMergesUserAndDefaultWatches -v`
Expected: PASS

- [ ] **Step 5: Run the Ruby integration test to verify it passes**

Run: `mise exec -- bin/rake install && mise exec -- bin/rspec spec/integration/plur_watch/watch_integration_spec.rb -e "keeps default watches when user config adds a custom watch mapping"`
Expected: PASS — `watch find lib/calculator.rb` finds `lib-to-spec` rule and resolves to `spec/calculator_spec.rb`

- [ ] **Step 6: Commit**

```bash
git add cmd_watch.go
git commit -m "$(cat <<'EOF'
fix: merge user watch mappings with built-in defaults

User-defined [[watch]] entries in .plur.toml now augment the built-in
watch mappings for the resolved job instead of replacing them.
EOF
)"
```

---

### Task 2: Update ValidateConfig for consistency

`ValidateConfig` in `autodetect/defaults.go` has the same replace-not-merge pattern. Built-in watches are guaranteed valid, but for correctness the merged set should be validated.

**Files:**
* Modify: `autodetect/defaults.go:292-295`
* Test: `autodetect/defaults_test.go` (add new test)

- [ ] **Step 1: Write a failing test for validation of merged watches**

Add to `autodetect/defaults_test.go`:

```go
func TestValidateConfigAcceptsUserWatchesAlongsideDefaults(t *testing.T) {
	userWatches := []watch.WatchMapping{
		{
			Name:   "custom-watch",
			Source: "config/**/*.yml",
			Jobs:   []string{"rspec"},
		},
	}

	err := ValidateConfig(nil, userWatches)
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run test to verify it passes (it already should — this is a safety check)**

Run: `mise exec -- go test ./autodetect/ -run TestValidateConfigAcceptsUserWatchesAlongsideDefaults -v`
Expected: PASS (validation already works for valid user watches)

- [ ] **Step 3: Update ValidateConfig to merge watches**

In `autodetect/defaults.go`, replace lines 292-295:

```go
	watches := userWatches
	if len(watches) == 0 {
		watches = builtinDefaults.Defaults.Watches
	}
```

With:

```go
	watches := builtinDefaults.Defaults.Watches
	watches = append(watches, userWatches...)
```

- [ ] **Step 4: Run existing validation tests to verify nothing breaks**

Run: `mise exec -- go test ./autodetect/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add autodetect/defaults.go autodetect/defaults_test.go
git commit -m "$(cat <<'EOF'
fix: validate merged watch set in ValidateConfig

Consistent with loadWatchConfiguration — validate user watches
alongside built-in defaults, not instead of them.
EOF
)"
```

---

### Task 3: Remove duplicate failure warning (S3)

When a job fails, both `ExecuteJob` (watcher.go:246,267) and `HandleBatch` (file_event_handler.go:109) log a WARN. Remove the one in `ExecuteJob` since it returns the error to the caller.

**Files:**
* Modify: `watch/watcher.go:246,267`
* Test: `watch/watcher_test.go` (add test)

- [ ] **Step 1: Write a test that ExecuteJob returns the error without logging WARN**

Add to `watch/watcher_test.go`:

```go
func TestExecuteJobReturnsErrorWithoutLogging(t *testing.T) {
	j := job.Job{
		Name: "failing-job",
		Cmd:  []string{"false"},
	}

	err := ExecuteJob(j, nil, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit status 1")
}
```

- [ ] **Step 2: Run the test to verify it passes (error is already returned)**

Run: `mise exec -- go test ./watch/ -run TestExecuteJobReturnsErrorWithoutLogging -v`
Expected: PASS

- [ ] **Step 3: Remove the duplicate WARN from ExecuteJob**

In `watch/watcher.go`, remove the WARN log at line 246 (non-target path):

```go
		if err := execCmd.Run(); err != nil {
			logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
			return err
		}
```

Change to:

```go
		if err := execCmd.Run(); err != nil {
			return err
		}
```

And remove the WARN log at line 267 (target path):

```go
	if err := execCmd.Run(); err != nil {
		logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
		return err
	}
```

Change to:

```go
	if err := execCmd.Run(); err != nil {
		return err
	}
```

- [ ] **Step 4: Run watch tests to verify nothing breaks**

Run: `mise exec -- go test ./watch/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add watch/watcher.go watch/watcher_test.go
git commit -m "$(cat <<'EOF'
fix: remove duplicate WARN on job failure in watch mode

ExecuteJob now returns the error without logging. HandleBatch in
file_event_handler.go remains the single place that logs job failures.
EOF
)"
```

---

### Task 4: Fix prompt/command rendering (S1 + S2)

Two related issues: the command should appear on the same line as the prompt (S1), and debug output should not interleave with the prompt (S2).

**Files:**
* Modify: `watch/watcher.go:237,258` — remove `[plur]` prefix and leading `\n` from command echo
* Modify: `cmd_watch.go:313` — clear prompt line before debug logging

- [ ] **Step 1: Change ExecuteJob command echo to continue the prompt line**

In `watch/watcher.go`, change line 237 (non-target path):

```go
		fmt.Printf("\n[plur] %s\n", strings.Join(cmd, " "))
```

To:

```go
		fmt.Printf("%s\n", strings.Join(cmd, " "))
```

And change line 258 (target path):

```go
	fmt.Printf("\n[plur] %s\n", strings.Join(cmd, " "))
```

To:

```go
	fmt.Printf("%s\n", strings.Join(cmd, " "))
```

This makes the command continue on the `[plur] > ` prompt line in normal mode, producing:
```
[plur] > rspec spec/calculator_spec.rb
```

- [ ] **Step 2: Clear prompt line before debug logging in event handler**

In `cmd_watch.go`, add a prompt-clearing newline before the debug log at line 313. Change:

```go
		logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)
```

To:

```go
		if logger.IsDebugEnabled() {
			fmt.Println()
		}
		logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)
```

In debug mode this produces:
```
[plur] >
14:06:41 - DEBUG - watch path="spec/..." event="modify" type="file"
14:06:41 - INFO  - Executing job job="rspec" targets="[spec/...]"
rspec spec/calculator_spec.rb
<rspec output>

[plur] >
```

- [ ] **Step 3: Run all Go tests**

Run: `mise exec -- go test ./... -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add watch/watcher.go cmd_watch.go
git commit -m "$(cat <<'EOF'
fix: render watch command on same line as prompt

ExecuteJob now prints just the command without [plur] prefix, so it
continues the prompt line. In debug mode, the prompt is cleared before
log output to prevent interleaving.
EOF
)"
```

---

### Task 5: Full integration verification

- [ ] **Step 1: Run full Go test suite**

Run: `mise exec -- go test ./...`
Expected: All tests PASS

- [ ] **Step 2: Install and run full Ruby integration suite**

Run: `bin/rake install && bin/rake test`
Expected: All tests PASS

- [ ] **Step 3: Run full build (lint + install + tests)**

Run: `bin/rake`
Expected: All steps PASS
