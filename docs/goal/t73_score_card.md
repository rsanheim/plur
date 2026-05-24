# T73 Score Card - After Watch Parity Review

Status: verified
Commit: c14aa0ac300261c1879ac2d11509154f8f7fb723

## Context

This reflection covers the DEV loop after T67:

- T68: `watch find` now shares live watch event admission, including
  default/custom ignores and `--ignore`.
- T69: live watch and `watch find` now share `watch.ExecutionPlan` for
  argv/env/cwd/targets.
- T70: session parity tests cover missing targets, reload-only mappings,
  multiple jobs, and planning errors.
- T72: review follow-up fixed `ExecutionPlan` to preserve the planner-resolved
  job key instead of trusting `job.Job.Name`.

Point in time reviewed: `3ccd8efc` on branch `prep-goal`.

Implementation commits reviewed:

- T68: `cb15ed8f592602ecd9c6f1e1321b51f13473cc8d`
- T69: `9dcda42a53909f94a9e4da5e7e10ccb5bc518f6c`
- T70: `6e3e45c1533f7082481d23c5145afa0cb7f21a1a`
- T72: `2b2b4842d9d88a5e5793d52d78d12ce4e4e59473`

Inputs:

- `docs/goal/new_design.md`
- `internal/watchsession/session.go` and tests
- `watch/planner.go`, `watch/file_event_handler.go`, `watch/execution_plan.go`
- `watch_find.go`
- Code-review feedback from Meitner

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4.5 | `watch find` now reports ignored previews as exit 2 before planning and shows runnable command plans from the same execution-plan type live watch uses. | Text output for ignored previews is intentionally terse. | Keep ignored-preview JSON documented; only add text detail if users ask. | More text can make editor-save output noisy. |
| Brevity / surface area | 4 | No new command nouns were added; `--ignore` now means the same thing for `watch find` and live watch. | The watch surface still has run/find/install plus inherited global flags. | Continue pruning only flags that truly do not affect the selected mode. | More help filtering can drift from Kong. |
| Default quality | 4.5 | Default ignore patterns are now applied before both live planning and preview planning. | Invalid CLI ignore globs are still silently non-matching. | Consider validating `--ignore` patterns at the watch command boundary. | Tightening CLI ignore validation is a breaking behavior change for typoed commands. |
| Conceptual coherence | 4.5 | CLI to session to planner to handler now shares job selection, watch mappings, cwd normalization, admission, planning, and execution-plan construction. | Live watch still has persistent process concerns outside `watch find`, as expected. | Leave persistence/output differences at the CLI edge; keep core planning shared. | Over-abstracting the persistent watcher could make the code harder to follow. |
| Feedback quality | 4.5 | Config errors, ignored previews, missing targets, no-rule changes, and planning errors now have explicit paths through shared result types. | Planning-error JSON remains mostly unit/session protected because invalid config fails earlier. | Treat structured planner-error JSON as an internal escape hatch unless a real CLI path needs it. | Advertising rare shapes can overpromise. |
| Composability | 4.5 | `watch find --format=json` command fields now come from `watch.ExecutionPlan`; live handler returns `ExecutedPlans` for parity tests. | End-to-end live parity still relies on session-level tests for command details to avoid watcher timing. | Keep command-shape parity at session level; use integration specs only for CLI contract smoke tests. | Session tests do not exercise the external watcher process. |
| Config/API cleanliness | 4.5 | Persistent TOML schema is durable, and watch config validation rejects invalid watch globs before either mode starts. | Planning docs remain under `docs/goal`, but that is intentionally internal for this loop. | Move/partition internal goal docs only when publication hygiene becomes the highest-value issue. | Moving docs now could distract from CLI behavior work. |

## Review Summary

Meitner reviewed T68-T70 and found one medium issue:

- `watch.ExecutionPlan` originally used `job.Job.Name`, which could be blank or
  stale for direct planner/handler callers.
- T72 fixed it by preserving `JobPlan.JobName` from the planner and adding a
  guardrail for a blank `job.Job.Name`.

No other blocking issues were reported.

Residual risks worth carrying forward:

- CLI `--ignore` invalid glob patterns silently never match.
- `watch find` and live watch now share core behavior, but watcher-process
  lifecycle behavior remains intentionally separate and timing-sensitive.

## Evidence

Shared admission:

```text
plur watch --ignore=lib/** find --format=json lib/calculator.rb
exit_code: 2
admission.reason: ignored
```

Shared execution plan:

```text
Session.PlanPath(...) -> watch.BuildExecutionPlans(...)
Session.Handler().HandleBatch(...) -> HandleResult.ExecutedPlans
```

Edge cases covered at the session boundary:

- missing targets
- reload-only mappings
- multiple jobs
- planning errors

Fresh verification from this loop:

- T68 `bin/rake` passed with 385 examples, 0 failures, 4 pending.
- T69 `bin/rake` passed with 385 examples, 0 failures, 4 pending.
- T70 `bin/rake` passed with 385 examples, 0 failures, 4 pending.
- T72 `bin/rake` passed with 385 examples, 0 failures, 4 pending.

## Top Design Problems

1. Invalid `--ignore` CLI globs are still accepted and silently non-matching.
2. The external watcher process remains harder to test than the session core.
3. Internal goal docs still live under `docs/goal`.

## Recommended Next Changes

1. T74-DEV: validate watch CLI `--ignore` patterns with the same glob validator
   used for persistent watch config ignores.
2. T75-REFLECT: decide whether the watch-find/live-watch parity objective is
   sufficiently closed, or whether to invest in external watcher-process
   integration coverage.

## Things That Should Not Change

- `watch find` should remain one-shot and side-effect-free.
- Live watch should remain persistent and own watcher lifecycle, prompts, and
  reload behavior at the outer layer.
- Session-level tests should remain the main place for command-shape parity;
  they are faster and less timing-sensitive than full watcher integration.
- Config validation should remain strict and fail before command execution.

## Done-Done Status

Nearly done for the user's stated parity concern. The core path from CLI-loaded
runtime config through session admission, planning, and execution-plan
construction is shared. The remaining useful cleanup is validating CLI
`--ignore` globs so the now-shared ignore path is not silently undermined by
typos.

## T73 Validation

This phase was documentation/reflection only after verified T72 code review
follow-up. Fresh verification:

- `script/check-links` passed, including MkDocs validation and `linkcheckmd`.
