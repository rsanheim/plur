# Watch Plan Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `plur watch find FILE` and live `plur watch` use one shared core watch planning path for the same changed file.

**Architecture:** Extract pure planning before changing behavior. `watch.Planner` owns changed-path planning and returns side-effect-free plans. A later shared watch session facade owns selected job lookup, cwd normalization, ignore defaults, watch directory planning, and planner construction for both commands.

**Tech Stack:** Go packages in this repo, RSpec integration specs, `bin/rake`, `go test -mod=mod`.

---

## File Map

- Modify: `watch/file_event_handler_test.go`
  Characterization tests around current `FileEventHandler.HandleBatch` behavior.
- Modify: `watch/find_test.go` or create it if absent
  Characterization tests around `FindTargetsForFile` and future `Planner`.
- Modify: `watch/file_event_handler.go`
  Eventually remove planning responsibility and execute a plan.
- Modify: `watch/find.go`
  Eventually becomes planner-compatible or is replaced by `Planner.PlanPath`.
- Modify: `watch/processor.go`
  Eventually return matched rules and rendered targets from one pass.
- Modify: `cmd_watch.go`
  Replace inline event admission and later consume shared session/planner.
- Modify: `watch_find.go`
  Render from shared planner/session output.
- Create: `watch/event_admission.go`
  Pure watcher event admission function for path type, effect type, cwd-relative path, and global ignore.
- Create: `watch/planner.go`
  Pure planner types and methods.
- Create: `internal/watchsession/session.go`
  Shared command/session setup after planner extraction, if it avoids import cycles cleanly.

## Task 1: Characterize Current Watch Planning Surfaces

**Files:**
- Modify: `watch/file_event_handler_test.go`
- Create or modify: `watch/find_test.go`

- [ ] **Step 1: Add `FindTargetsForFile` characterization tests**

Add tests for these cases:

```go
func TestFindTargetsForFile_PlanCases(t *testing.T) {
    // Create a temp project with lib/user.rb and spec/user_spec.rb.
    // Assert lib/user.rb matches lib-to-spec, existing target is spec/user_spec.rb.
    // Assert spec/spec_helper.rb has no matched rules and no targets.
    // Assert lib/missing.rb matches lib-to-spec and reports missing spec/missing_spec.rb.
    // Assert a watch mapping with Ignore: []string{"lib/ignored.rb"} does not match that path.
}
```

- [ ] **Step 2: Add `FileEventHandler` characterization tests**

Add tests for these cases:

```go
func TestFileEventHandler_HandleBatch_ReloadOnlyMappingDoesNotReportMissingTargets(t *testing.T) {
    // A reload watch with no existing target should set ShouldReload=true
    // and leave NoRunnableChanges empty.
}

func TestFileEventHandler_HandleBatch_MixedRunnableAndNoRule(t *testing.T) {
    // A batch containing lib/user.rb and spec/spec_helper.rb should execute rspec
    // for spec/user_spec.rb and report a no_matching_rule change for spec/spec_helper.rb.
}
```

- [ ] **Step 3: Run red/green characterization check**

Run:

```bash
go test -mod=mod ./watch -run 'Test(FindTargetsForFile|FileEventHandler)' -count=1
```

Expected: pass after adding characterization tests without behavior changes.

- [ ] **Step 4: Commit**

```bash
git add watch/file_event_handler_test.go watch/find_test.go docs/goal/new_design.md tracking.md
git commit -m "watch: characterize planning parity"
```

## Task 2: Extract Watcher Event Admission

**Files:**
- Create: `watch/event_admission.go`
- Create: `watch/event_admission_test.go`
- Modify: `cmd_watch.go`

- [ ] **Step 1: Write failing event admission tests**

Add tests for:

```go
func TestAdmitEvent(t *testing.T) {
    // watcher path type is ignored
    // destroy effect is ignored
    // modify and create are admitted
    // global ignore pattern skips admitted-looking path
    // absolute event path is converted relative to cwd
}
```

- [ ] **Step 2: Run the focused test and verify failure**

Run:

```bash
go test -mod=mod ./watch -run TestAdmitEvent -count=1
```

Expected before implementation: build failure for missing `AdmitEvent`.

- [ ] **Step 3: Implement `watch.AdmitEvent`**

Create a pure function with this shape:

```go
type AdmissionResult struct {
    Path     string
    Admitted bool
    Reason   string
}

func AdmitEvent(event Event, cwd string, ignorePatterns []string) AdmissionResult {
    if event.PathType == "watcher" {
        return AdmissionResult{Admitted: false, Reason: "watcher"}
    }
    if event.EffectType != "modify" && event.EffectType != "create" {
        return AdmissionResult{Admitted: false, Reason: "effect"}
    }
    path, err := filepath.Rel(cwd, event.PathName)
    if err != nil {
        return AdmissionResult{Admitted: false, Reason: "relative_path"}
    }
    if IsIgnored(path, ignorePatterns) {
        return AdmissionResult{Path: path, Admitted: false, Reason: "ignored"}
    }
    return AdmissionResult{Path: path, Admitted: true}
}
```

- [ ] **Step 4: Replace inline live-watch event filtering**

In `cmd_watch.go`, replace the inline `PathType`, `filepath.Rel`, `IsIgnored`,
and `EffectType` checks with `watch.AdmitEvent`. Preserve the existing debug and
warning output where practical.

- [ ] **Step 5: Verify**

Run:

```bash
go test -mod=mod ./watch -run TestAdmitEvent -count=1
go test -mod=mod ./watch
bin/rake build
```

- [ ] **Step 6: Commit**

```bash
git add watch/event_admission.go watch/event_admission_test.go cmd_watch.go docs/goal/new_design.md tracking.md
git commit -m "watch: extract event admission"
```

## Task 3: Introduce Pure Watch Planner

**Files:**
- Create: `watch/planner.go`
- Create: `watch/planner_test.go`
- Modify: `watch/file_event_handler.go`
- Modify: `watch/find.go`
- Modify: `watch/processor.go`

- [ ] **Step 1: Write planner tests from characterization cases**

Add tests for:

```go
func TestPlanner_PlanPath(t *testing.T) {
    // runnable target: matched rule, existing target, ordered job plan
    // no matching rule: no-runnable reason
    // missing target: missing target and no job plan
    // reload-only: ShouldReload true even with no runnable target
}

func TestPlanner_PlanBatch(t *testing.T) {
    // dedupes targets per job
    // preserves first matched rule job order
    // includes no-runnable changes for paths that did not produce work
}
```

- [ ] **Step 2: Run the focused planner test and verify failure**

Run:

```bash
go test -mod=mod ./watch -run TestPlanner -count=1
```

Expected before implementation: build failure for missing `Planner`.

- [ ] **Step 3: Implement planner types**

Create types equivalent to:

```go
type Plan struct {
    Paths             []string
    MatchedRules      []WatchMapping
    ExistingTargets   map[string][]string
    MissingTargets    map[string][]string
    JobPlans          []JobPlan
    ShouldReload      bool
    NoRunnableChanges []NoRunnableChange
}

type JobPlan struct {
    JobName string
    Job     job.Job
    Targets []string
}

type Planner struct {
    Jobs    map[string]job.Job
    Watches []WatchMapping
    CWD     string
}
```

- [ ] **Step 4: Move planning logic behind `Planner`**

Move aggregation, matched rule collection, existing/missing split, reload
detection, target dedupe, and ordered job-plan creation out of
`FileEventHandler.HandleBatch` into `Planner.PlanBatch`.

- [ ] **Step 5: Keep `FileEventHandler` as executor**

Change `FileEventHandler.HandleBatch` to:

```go
plan := h.planner().PlanBatch(paths)
for _, jobPlan := range plan.JobPlans {
    _ = h.executor()(jobPlan.Job, jobPlan.Targets, h.CWD)
}
return HandleResult{...from plan...}
```

- [ ] **Step 6: Verify**

Run:

```bash
go test -mod=mod ./watch -run 'TestPlanner|TestFileEventHandler' -count=1
go test -mod=mod ./watch
```

- [ ] **Step 7: Commit**

```bash
git add watch/planner.go watch/planner_test.go watch/file_event_handler.go watch/find.go watch/processor.go docs/goal/new_design.md tracking.md
git commit -m "watch: add shared planner"
```

## Task 4: Render `watch find` From Planner Output

**Files:**
- Modify: `watch_find.go`
- Modify: `spec/integration/watch/watch_find_spec.rb`
- Modify: `spec/integration/watch/watch_find_json_spec.rb`

- [ ] **Step 1: Write a parity-focused failing test**

Add or adjust an integration spec so `watch find --format=json lib/calculator.rb`
asserts fields that come from the plan: matched rules, existing targets, missing
targets, and exit code.

- [ ] **Step 2: Update `watch_find.go` to use `watch.Planner`**

Replace direct `watch.FindTargetsForFile` usage with a single-path planner call.
Keep current public text and JSON shapes unless the planner adds an explicitly
documented field in a later phase.

- [ ] **Step 3: Verify**

Run:

```bash
bin/rake build
PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb
go test -mod=mod ./watch
```

- [ ] **Step 4: Commit**

```bash
git add watch_find.go spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb docs/goal/new_design.md tracking.md
git commit -m "watch: render find from planner"
```

## Task 5: Add Shared Watch Session Facade

**Files:**
- Create: `internal/watchsession/session.go`
- Create: `internal/watchsession/session_test.go`
- Modify: `cmd_watch.go`
- Modify: `watch_find.go`

- [ ] **Step 1: Write session tests**

Add tests for:

```go
func TestSessionNew_SelectsJobAndComputesWatchDirs(t *testing.T) {
    // Given runtime config with rspec job and watches,
    // session.Selected.Name is rspec and WatchDirs are filtered source dirs.
}

func TestSessionNew_UsesDefaultIgnoresWhenNoneProvided(t *testing.T) {
    // Ignore equals watch.DefaultIgnorePatterns.
}

func TestSessionNew_UsesProvidedIgnores(t *testing.T) {
    // Ignore equals provided patterns.
}
```

- [ ] **Step 2: Implement `internal/watchsession.Session`**

The facade should contain selected job, jobs, watches, cwd, ignore patterns,
watch dirs, and a planner. Keep command output outside the package.

- [ ] **Step 3: Wire live watch**

In `cmd_watch.go`, replace selected-job lookup, cwd normalization, ignore
defaulting, watch-dir derivation, and handler construction with session fields.

- [ ] **Step 4: Wire `watch find`**

In `watch_find.go`, use the same session constructor. For no-watch cases, keep
the existing text/JSON output and exit code unless a later phase changes the
contract.

- [ ] **Step 5: Verify**

Run:

```bash
go test -mod=mod ./internal/watchsession ./watch
bin/rake build
PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_run_flags_spec.rb
```

- [ ] **Step 6: Commit**

```bash
git add internal/watchsession cmd_watch.go watch_find.go docs/goal/new_design.md tracking.md
git commit -m "watch: share session setup"
```

## Task 6: Add Live-vs-Find Integration Parity Coverage

**Files:**
- Modify or create: `spec/integration/watch/watch_plan_parity_spec.rb`
- Modify: `docs/output-contracts.md` only if a stable output field changes

- [ ] **Step 1: Add integration parity spec**

Write an integration spec that:

- runs `plur watch find --format=json lib/calculator.rb`
- starts live watch in a fixture with a short timeout or interactive helper
- modifies `lib/calculator.rb`
- asserts live output runs the same job and target shown by `watch find`

- [ ] **Step 2: Add ignore parity spec if session exposes global ignores to find**

Only add this if the implementation makes `watch find` use the same ignore
admission settings. If `watch find` intentionally remains a raw mapping preview
for one more phase, record that as the next follow-up instead.

- [ ] **Step 3: Verify full gate**

Run:

```bash
go test -mod=mod ./...
script/check-links
bin/rake
```

- [ ] **Step 4: Commit**

```bash
git add spec/integration/watch/watch_plan_parity_spec.rb docs/output-contracts.md docs/goal/new_design.md tracking.md
git commit -m "watch: cover find live parity"
```

## Plan Self-Review

- Spec coverage: The plan covers characterization, event admission, pure
  planning, `watch find` rendering, shared session construction, and integration
  parity.
- Placeholder scan: No placeholder markers or intentionally vague task remains.
- Type consistency: `Plan`, `JobPlan`, `Planner`, `Session`, and
  `AdmissionResult` names are consistent across tasks.
- Scope check: This is multiple DEV phases, not one commit. Each task is
  independently testable and committable.
