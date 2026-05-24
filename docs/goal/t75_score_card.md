# T75 Score Card - Watch Find / Live Watch Parity Closure

Status: verified
Commit: 8a35d02069a1da5dfcd4e49907a75793e6f3265e

## Context

This reflection answers the larger review question opened by the user: whether
`plur watch find FILE` and live `plur watch` reacting to the same file now use
the same code for job selection, watch configuration, ignores, planning, and
execution-plan construction.

Point in time reviewed: `97d88241` on branch `prep-goal`.

Implementation commits reviewed:

- T56: `7afbd8c447d1dcc062ba7847e774b47511ac67f5`
- T68: `cb15ed8f592602ecd9c6f1e1321b51f13473cc8d`
- T69: `9dcda42a53909f94a9e4da5e7e10ccb5bc518f6c`
- T70: `6e3e45c1533f7082481d23c5145afa0cb7f21a1a`
- T72: `2b2b4842d9d88a5e5793d52d78d12ce4e4e59473`
- T74: `02c2f0119f3a73cb0fcab712929affe031573464`

## Closure Assessment

The core parity issue is closed.

`watch find` and live watch now share:

- runtime job selection through `watchsession.New`
- watch mapping loading/merging through `RuntimeConfig`
- cwd normalization through `watchsession.Session`
- default and CLI ignore handling through session admission
- glob validation for watch config and watch-session ignores
- target planning through `watch.Planner`
- command argv/env/cwd/target construction through `watch.ExecutionPlan`

The remaining differences are outer-layer behavior, which should differ:

- `watch find` is one-shot and renders preview output.
- live watch installs/starts a watcher, debounces events, prompts, handles
  reload, and executes commands.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4.5 | `watch find` now previews the same admission/planning/execution inputs live watch uses. | Ignored text previews stay terse. | Keep detailed ignored-state information in JSON. | More text can clutter live-like output. |
| Brevity / surface area | 4.5 | No new user command was added; `--ignore` now works consistently for watch preview and live watch. | Watch still has advanced setup commands. | Leave advanced commands grouped in help. | Hiding setup too aggressively can hurt discoverability. |
| Default quality | 4.5 | Default ignores, custom ignores, missing targets, reload-only rules, multiple jobs, and planning errors are covered at the shared session boundary. | External watcher integration remains timing-sensitive. | Keep most parity checks below the watcher-process layer. | Session tests do not verify the external watcher binary. |
| Conceptual coherence | 4.5 | The shared path is now CLI config -> runtime config -> watch session -> admission -> planner -> execution plan. | Persistent watch lifecycle remains separate by design. | Treat output/persistence as edges, not part of core planning. | Over-sharing lifecycle code would add complexity. |
| Feedback quality | 4.5 | Invalid config globs, invalid `--ignore`, no-rule, missing-target, ignored, and planning-error paths all fail or report explicitly. | Planner-error JSON is still an uncommon internal path. | Do not expand that contract unless a real CLI case needs it. | More error schema increases maintenance cost. |
| Composability | 4.5 | `watch find --format=json` exposes command plans generated from the same `ExecutionPlan` type live watch executes. | Shell string remains human-oriented. | Keep `argv`/`env` canonical for scripts. | None significant. |
| Config/API cleanliness | 4.5 | Persistent and CLI watch ignore globs are both validated deliberately. | Internal goal docs still live under `docs/goal`. | Move internal planning docs later if public docs hygiene becomes priority. | Moving now would be churn. |

## Remaining Risks

- External watcher-process tests remain timing-sensitive; session tests are the
  safer parity guardrail.
- `watch find` no-watch-mapping behavior still has a small CLI-specific branch
  for human guidance, but that branch occurs before any watch planning can
  exist and does not affect configured projects.

## Things That Should Not Change

- Keep `watch find` side-effect-free.
- Keep live watch's watcher lifecycle, debounce, prompt, and reload handling at
  the outer command layer.
- Keep `watch.ExecutionPlan` as the shared command-shape boundary.
- Keep config and ignore validation strict.

## Done-Done Status

Done for the parity problem described by the user. Further work should be
treated as new polish or publication hygiene, not as required to stop the
watch-find/live-watch drift.

## T75 Validation

This phase was documentation/reflection only. Fresh verification:

- `script/check-links` passed, including MkDocs validation and `linkcheckmd`.
