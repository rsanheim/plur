# T67 Score Card - After Planning Error Surfacing

Status: verified
Commit: pending

## Context

This reflection covers the DEV loop after T63:

- T64: persistent TOML validation now uses a durable schema instead of walking
  the Kong CLI tree.
- T65: `watch find --format=json` now documents a successful command-plan
  shape with `job_plans`.
- T66: watch planning errors are no longer silently dropped by the shared
  planner, and invalid watch glob patterns fail config validation.

Point in time reviewed: `7a499ef5` on branch `prep-goal`.

Implementation commits reviewed:

- T64: `19e1fd5ce49abdcc80783e3acb5d1d1b4b3a6284`
- T65: `710ead53f93870d53bb7ab240cf9aebcfefebfe7`
- T66: `cdc241456f86b33dae823ed468fb49cabc2ad47c`

Inputs:

- Latest design notes in `docs/goal/new_design.md`.
- Public output contract in `docs/output-contracts.md`.
- Runtime/session code in `internal/watchsession`, `watch_find.go`, and
  `cmd_watch.go`.
- Advisor reviews from Jason and McClintock.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `watch find` now shows matched rules, target status, and runnable command plans; no-rule output includes a `[[watch]]` hint. | It can still preview a path that live watch would ignore via default/global ignores. | Route preview paths through the same event-admission rules as live watch. | Ignored previews need clear output so users do not think the mapping disappeared. |
| Brevity / surface area | 4 | T64-T66 added no new user commands and tightened existing `watch find`, config, and output-contract behavior. | `watch find` rejects `--ignore` even though ignore admission is part of live watch behavior. | Let `--ignore` mean the same thing for preview and live watch. | Accepting the flag expands the watch-find contract. |
| Default quality | 4 | Invalid watch glob config now fails before either preview or live watch starts. | Default ignore behavior can still diverge between preview and live mode. | Add default/global ignore admission to `watch find`. | Users may need an explanation when previewing ignored paths under `node_modules` or similar dirs. |
| Conceptual coherence | 4 | `watch find` and live watch share `watchsession.New`, `watch.Planner`, job selection, watch mappings, and job command construction. | They do not yet share the event-admission boundary that live watch uses before planning. | Add a session preview-admission helper and make `watch find` call it before `PlanPath`. | The no-watch and missing-watch-dir diagnostics need to stay understandable. |
| Feedback quality | 4 | Unknown config, CLI-only config, and invalid watch glob errors now name the source and failing key/pattern. | Planning-error JSON is documented but mostly unit-level protected, because invalid config is rejected before CLI JSON rendering. | Add handler/session tests for planning errors and adjust docs if an advertised JSON shape is intentionally rare. | More error contract detail creates more maintenance surface. |
| Composability | 4 | `watch find --format=json` exposes stable `job_plans[].argv`, `env`, `cwd`, and `shell`. | Parity specs still prove target agreement more than final command agreement. | Compare preview `job_plans` to the live executor call at the session boundary. | End-to-end live assertions can become timing-sensitive if placed too high. |
| Config/API cleanliness | 4 | Persistent config schema is now explicit: documented globals, `[job.*]`, and `[[watch]]`; transient CLI/session controls are rejected. | Tracking rows still record point-in-time short OIDs, while phase docs carry the implementation commit refs. | Keep phase docs as the source of commit refs, and include implementation refs in future reflection notes. | Retrofitting old tracking rows would add churn without improving current behavior. |

## Advisor Summary

Jason, watch/live architecture lens:

- The highest remaining parity risk is that `watch find` bypasses live event
  admission.
- Live watch calls `session.AdmitEvent` before planning; `watch find` calls
  `session.PlanPath` directly.
- Default/global ignore patterns and `--ignore` can therefore disagree between
  preview and live execution.
- Recommended next slice: synthesize the same "file modify" admission for
  `watch find`, stop rejecting `--ignore`, and render ignored previews as a
  non-runnable exit 2 plan.

McClintock, docs/test lens:

- T64-T66 phase docs are coherent and include full implementation commit refs.
- `tracking.md` rows are point-in-time events and can be ambiguous if read as
  implementation refs; future reflections should explicitly list reviewed
  implementation commits.
- Parity coverage is still target-heavy: it should also compare argv/env/cwd,
  missing-target behavior, reload-only behavior, multiple-job plans, and
  planning-error propagation.

## Evidence

T64 made the persistent config API deliberate:

```text
Persistent TOML keys are now owned by the runtime config schema rather than
inferred from every Kong command and flag.
```

T65 documented successful watch-find command plans:

```json
"job_plans": [
  {
    "job": "rspec",
    "targets": ["spec/calculator_spec.rb"],
    "argv": ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
    "env": [],
    "cwd": "/project",
    "shell": "bundle exec rspec spec/calculator_spec.rb"
  }
]
```

T66 made planner failures visible:

```text
watch "missing-source" has invalid source pattern "": must not be empty
```

Fresh verification from T66 before this reflection:

- `PLUR_BINARY=$PWD/tmp/plur-t66 bin/rspec spec/integration/watch/watch_config_spec.rb spec/integration/spec/configuration_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 51 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 384 examples, 0 failures, and 4 existing pending
  examples.

## Top Design Problems

1. `watch find` does not run the live watch event-admission path before
   planning.
2. `watch find --ignore` is rejected even though global ignores are part of
   live watch behavior.
3. Parity coverage compares preview targets to live output, but does not yet
   lock preview command plans to live executor calls.
4. Planning-error propagation is better, but live handler and CLI JSON coverage
   are still thin.
5. Session parity does not yet cover missing targets, reload-only mappings, or
   multiple jobs.

## Recommended Next Changes

1. T68-DEV: make `watch find` share live event admission, including default
   ignores and `--ignore`.
2. T69-DEV: strengthen session parity tests so preview `job_plans` match live
   executor calls for targets, argv/env/cwd, and job ordering.
3. T70-DEV: add parity cases for missing targets, reload-only mappings,
   multiple jobs, and planning errors.

## Things That Should Not Change

- `watch find` should remain a one-shot preview command, not a persistent
  watcher.
- `watch find --format=json` should keep clean JSON on stdout with exit 2 for
  non-runnable previews.
- Config validation should stay strict; invalid watch definitions should fail
  before preview or live watch startup.
- Old tracking rows do not need an audit or rewrite; current and future phase
  docs should carry the commit refs.

## Done-Done Status

Not done. T64-T66 raised config cleanliness to the same baseline as the other
categories, but the user's larger concern is still live: `watch find` and live
watch must share the admission and execution-planning path all the way from CLI
boundary to handler. The next slice should close the ignore/admission gap.

## T67 Validation

This phase was documentation/reflection only. Fresh verification:

- `script/check-links` passed, including MkDocs validation and `linkcheckmd`.
