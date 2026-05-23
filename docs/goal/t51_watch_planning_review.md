# T51 Watch Planning Architecture Review

Status: verified
Commit: 13ebbf1e1113c62396603eba27bb274c8d8d3b96

## User Problem

`plur watch find FILE` and live `plur watch` currently share some mapping
logic, but they do not share the same full watch path. That leaves the project
patching discrepancies one by one: job selection, watch configuration, global
ignores, watcher event admission, reload rules, target planning, and execution
planning can drift.

The desired shape is strict:

- `plur watch find foo_spec.rb`
- `plur watch` after `foo_spec.rb` is modified

Both paths must use the same core code from CLI/config/session through file
event planning. The only differences should be output, persistence, and
whether the plan is executed.

## Advisor Review

Sub-agents were consulted before selecting the next phase shape.

- Herschel traced live watch from CLI/bootstrap through runtime config, watch
  directory filtering, event admission, `FileEventHandler`, target mapping, and
  execution.
- Hubble traced `watch find`, compared it with live watch, and classified
  output-only differences versus semantic drift.
- Locke reviewed architecture options and recommended a pure planner followed
  by a shared watch session facade.

All three agreed that sharing `watch.FindTargetsForFile` is useful but too
low-level. The risky drift sits around that function: selected job lookup,
watch directory planning, global ignore handling, event admission, reload
planning, and final command planning.

## Current Flow Map

Shared today:

- `main.go` builds `globals.runtimeConfig`.
- `internal/runtime/config.go` resolves jobs, selects built-in watches, merges
  user watches, and validates watch references/templates.
- `watch.FindTargetsForFile` maps a changed path to matched rules and
  existing/missing targets.

Live watch only:

- `cmd_watch.go` selects a job for logging and run-all.
- `cmd_watch.go` derives watch directories and filters them with
  `watch.FilterDirectories`.
- `cmd_watch.go` applies global ignore patterns and event metadata filters.
- `cmd_watch.go` resolves symlinked cwd before turning watcher events into
  relative paths.
- `watch.FileEventHandler.HandleBatch` aggregates path results, decides reload,
  and executes jobs.

Watch find only:

- `watch_find.go` normalizes the requested file path against raw `os.Getwd`.
- `watch_find.go` has its own empty-watch behavior and job-selection ordering.
- `watch_find.go` directly calls `watch.FindTargetsForFile` and formats text or
  JSON.
- `watch_find.go` does not represent global ignores, watcher event admission,
  watch-directory filtering, reload behavior, or command planning.

## Decision

Use a two-layer refactor over multiple DEV phases.

1. Extract a pure `watch.Planner`.
   - Input: normalized changed paths, jobs, watches, cwd.
   - Output: a side-effect-free plan with matched rules, existing targets,
     missing targets, reload intent, ordered job plans, and no-runnable
     changes.
   - No stdout, no process execution, no Kong/runtime imports.

2. Add a shared watch session facade.
   - Location: `internal/watchsession` or main package, not package `watch`, to
     avoid an import cycle because `internal/runtime` already imports `watch`.
   - Responsibilities: selected job lookup, cwd normalization, default/custom
     ignore resolution, watch directory planning, and planner construction.
   - Both `watch find` and live `watch` call this facade.

3. Keep edges separate.
   - `watch find` presents text/JSON and exits.
   - Live watch starts the watcher process, debounces events, executes planned
     jobs, handles reload, prompts, and persistence.
   - Pressing Enter to run all tests remains a no-target command path.

Rejected approach: a full event pipeline with sinks is conceptually clean but
too broad for the next loop. It would combine watcher startup, debouncing,
planning, execution, and output changes into one large refactor.

## Phase Sequence

The next implementation loops should be small and committed independently:

1. T52-DEV: add characterization tests for existing path planning parity and
   no-runnable/reload behavior.
2. T53-DEV: extract watcher event admission into a pure function and use it in
   live watch.
3. T54-DEV: introduce `watch.Planner` and make `FileEventHandler` execute a
   plan instead of planning and executing together.
4. T55-DEV: make `watch find` render from the same planner output.
5. T56-DEV: add the shared watch session facade and move selected job lookup,
   cwd normalization, ignore defaults, and watch directory planning behind it.
6. T57-DEV: add integration parity coverage comparing `watch find` output with
   live watch behavior for a modified file.

## Success Criteria

- A changed file has one canonical watch plan regardless of whether it comes
  from `watch find` or a live watcher event.
- `watch find` can explain exactly what live watch would do for the same
  admitted file event.
- Live watch uses the same plan to execute jobs and decide reload/no-runnable
  feedback.
- Global ignore, cwd normalization, watch directory planning, and selected job
  lookup have one owner.
- Existing output contracts remain stable unless a later phase explicitly
  records a breaking change.

## Evidence

- Local code map reviewed `cmd_watch.go`, `watch_find.go`,
  `watch/file_event_handler.go`, `watch/find.go`, `watch/processor.go`, and
  `watch/watcher.go`.
- Three sub-agent reviewers independently identified the same divergence:
  `watch.FindTargetsForFile` is shared but is below the correct abstraction
  boundary.
- Implementation plan written at
  `docs/superpowers/plans/2026-05-23-watch-plan-parity.md`.
