# Remove Target Command Placeholder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Remove the legacy target command placeholder from job command definitions completely, with no compatibility path and a final repository audit proving the literal token is gone.

**Architecture:** Job commands become plain executable-plus-fixed-arguments arrays. Run mode and watch mode both append resolved target paths after command construction; watch target templates remain only in `[[watch]].targets`. Runtime config validation rejects any template-looking token in `job.cmd`, so command templating cannot re-enter through user config.

**Tech Stack:** Go, TOML embedded defaults, RSpec integration specs, testify Go tests, `bin/rake`.

---

### Task 1: Add Breaking-Change Tests

**Files:**
- Modify: `internal/runtime/config_test.go`
- Modify: `spec/integration/spec/configuration_spec.rb`
- Modify: `framework/run_args_test.go`
- Modify: `watch/watcher_test.go`
- Modify: `watch/processor_test.go`
- Modify: `watch/file_event_handler_test.go`

- [x] **Step 1: Add runtime validation tests**

Add tests in `internal/runtime/config_test.go` that reject template-looking command arguments in any job command. Cover a standalone placeholder-shaped arg and an embedded flag value.

- [x] **Step 2: Update integration validation spec**

Replace the run-mode-specific rejection spec in `spec/integration/spec/configuration_spec.rb` with a config validation spec that runs `plur doctor`, defines a job command with a watch-style template token, and expects an error containing `job "custom" command must not contain template tokens`.

- [x] **Step 3: Update framework command tests**

Remove the legacy placeholder arg from every `job.Job{Cmd: ...}` fixture in `framework/run_args_test.go`. Expected run args should stay the same because framework code appends targets.

- [x] **Step 4: Update watch command tests**

Remove command-placeholder fixtures from `watch/watcher_test.go`, `watch/processor_test.go`, and `watch/file_event_handler_test.go`. Keep watch target-template coverage in `[[watch]].targets` tests; those tokens are still valid.

- [x] **Step 5: Verify red**

Run:

```bash
go test ./internal/runtime ./framework ./watch
PLUR_BINARY=/Users/rsanheim/src/rsanheim/plur/plur bin/rspec spec/integration/spec/configuration_spec.rb
```

Expected before implementation: runtime validation and/or command behavior tests fail because command tokens are still accepted or stripped.

### Task 2: Remove Command Placeholder Code

**Files:**
- Modify: `internal/runtime/defaults.toml`
- Modify: `internal/runtime/config.go`
- Modify: `cmd_spec.go`
- Modify: `runner.go`
- Modify: `framework/run_args.go`
- Modify: `job/job.go`
- Modify: `internal/runtime/defaults.go`

- [x] **Step 1: Remove placeholders from embedded defaults**

In `internal/runtime/defaults.toml`, remove the legacy command placeholder arg from `rspec`, `minitest`, and `go-test` defaults.

- [x] **Step 2: Validate job commands at config load**

In `internal/runtime/config.go`, add validation during `validateRuntimeConfig`:

```go
for _, arg := range j.Cmd {
	if strings.Contains(arg, "{{") || strings.Contains(arg, "}}") {
		return fmt.Errorf("configuration error in %v: job %q command must not contain template tokens", rc.Sources, name)
	}
}
```

Use the existing `strings` import already present in that file.

- [x] **Step 3: Delete selected-job rejection**

Remove `rejectRunModeTargetTemplate` and its call from `cmd_spec.go`.

- [x] **Step 4: Delete debug strip note**

Remove the run-mode debug log in `runner.go` that mentions ignoring command target tokens.

- [x] **Step 5: Simplify run args**

In `framework/run_args.go`, remove `stripTargetTokens`, remove the `strings` import if it becomes unused, and start from a copy of `j.Cmd`:

```go
args := append([]string{}, j.Cmd...)
```

- [x] **Step 6: Simplify job command helpers**

In `job/job.go`, make `BuildJobCmd` append targets unconditionally. Delete `BuildJobAllCmd`, delete `UsesTargets`, and remove the `strings` import.

- [x] **Step 7: Reconsider inherited command metadata**

Keep `InheritedFields.Cmd` only if debug output still needs it. Do not keep any behavior that depends on inherited commands containing command placeholders.

- [x] **Step 8: Verify green for core code**

Run:

```bash
go test ./internal/runtime ./framework ./job ./watch
```

Expected: all listed packages pass.

### Task 3: Remove Stale Config and Docs

**Files:**
- Modify: `docs/configuration.md`
- Modify: `docs/architecture/runner-jobs-framework.md`
- Modify: `docs/architecture/runner-jobs-rfc.md`
- Modify: `docs/architecture/go-concurrency-and-data-structures-review.md`
- Modify: `docs/configuration-test-cases.md`
- Modify: `fixtures/projects/config-test/*.toml`
- Modify: `spec/integration/**/*.rb`

- [x] **Step 1: Update user configuration docs**

In `docs/configuration.md`, state that job commands are executable plus fixed args only. Remove any discussion of command placeholders. Keep watch placeholder docs scoped to `[[watch]].targets`.

- [x] **Step 2: Update architecture docs**

In `docs/architecture/runner-jobs-framework.md`, remove strip/reject/inherited-placeholder language and describe direct target appending. In RFC or review notes, replace stale command examples with plain commands or remove the stale note.

- [x] **Step 3: Update old config notes**

In `docs/configuration-test-cases.md`, avoid command-placeholder examples in TOML snippets.

- [x] **Step 4: Update fixtures and specs**

Run:

```bash
rg -n '\\{\\{target\\}\\}' fixtures spec docs
```

For every match, either remove the stale command placeholder or rewrite the text to discuss watch target templates without naming the removed command token.

### Task 4: Full Verification and Audit

**Files:**
- Verify all changed files.

- [x] **Step 1: Run targeted verification**

Run:

```bash
bin/rake build
PLUR_BINARY=/Users/rsanheim/src/rsanheim/plur/plur bin/rspec spec/integration/spec/configuration_spec.rb spec/integration/spec/framework_output_spec.rb spec/integration/spec/change_dir_config_spec.rb spec/integration/doctor/doctor_spec.rb
go test ./internal/runtime ./framework ./job ./watch
```

Expected: all targeted checks pass.

- [x] **Step 2: Run full project verification**

Run:

```bash
bin/rake
```

Expected: full project gate exits 0.

- [x] **Step 3: Run final literal placeholder audit**

Run:

```bash
rg -n '\\{\\{target\\}\\}' .
```

Expected: no matches.

- [x] **Step 4: Commit and push**

Run:

```bash
git status --short
git add docs/superpowers/plans/2026-06-06-remove-target-command-placeholder.md .
git commit -m "Remove target command placeholder"
git push
```

Expected: branch is pushed and working tree is clean.

### Self-Review

- Spec coverage: The plan covers code removal, validation, docs, fixtures, tests, and final literal audit.
- Placeholder scan: The plan avoids spelling the removed literal command token directly; the final audit pattern is escaped.
- Type consistency: Existing types remain `job.Job`, `RuntimeConfig`, `InheritedFields`, and framework args use `[]string`.
