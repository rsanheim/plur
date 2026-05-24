# Agent Config Architecture Assessment

Reviewer: Laplace
Persona: senior Go/config architecture reviewer
Model: `gpt-5.5`, reasoning `high`
Mode: read-only assessment

## Scope

Laplace compared:

- `v0.56.0`
- `main`
- `v0.60.0-rc.1`
- current `prep-goal`

The focused lens was config schema, job selection, watch/session/planner
architecture, command execution/dry-run JSON, and API cleanliness.

## Executive Assessment

`v0.60.0-rc.1` is a substantial architectural improvement for config
correctness, watch planning parity, dry-run composability, and mode-specific API
cleanliness.

The best change is the move from duplicated watch command paths toward a shared
internal pipeline:

```text
CLI/config -> runtime config -> watch session -> event admission -> planner -> execution plan
```

That pipeline is visible in:

- `internal/watchsession/session.go`
- `watch/planner.go`
- `watch/execution_plan.go`
- `cmd_watch.go`
- `watch_find.go`

Remaining architecture concerns:

- TOML validation is stricter, but nested fields still come from reflecting over
  `job.Job` and `watch.WatchMapping`.
- Human help customization depends on Kong output string replacement and hidden
  flag mutation.
- Structured output has two public shapes: one-shot `workers[]` and watch
  `job_plans[]`. This is acceptable, but should not sprawl.
- Internal goal docs are useful process evidence but should not become public
  product docs.

## Scorecard

| Category | Score | Evidence | Main Issue | Recommendation |
| --- | ---: | --- | --- | --- |
| Obviousness | 4.5 | Workflow-first help; dry-run names selected job and reason; watch-find previews command intent. | `spec` is still both an explicit command and the implicit default run path. | Keep examples path-first and avoid new command nouns. |
| Brevity / surface area | 4 | Mode-irrelevant watch flags are hidden or rejected. | The CLI still has several daily and setup commands. | Continue pruning flags by mode. |
| Default quality | 4.5 | RSpec-first autodetect, `-C` config loading, and shared watch ignores are clearer. | Shared/helper file edits still need mappings. | Keep no-rule hints and document shared-file mappings. |
| Conceptual coherence | 4.5 | Run mode rejects `{{target}}`; watch plans use `watch.ExecutionPlan`. | Run and watch target passing remain different. | Preserve the rule: run appends targets, watch maps targets. |
| Feedback quality | 4.5 | Unknown TOML keys, invalid watch globs, and no-runnable watch changes are explicit. | JSON-mode errors are prose on stderr. | Keep documented unless scripts require structured errors. |
| Composability | 4.5 | Dry-run JSON and watch JSON expose canonical `argv`/`env`. | `shell` is human convenience. | Keep `argv` and `env` canonical. |
| Config/API cleanliness | 4 | Unknown and CLI-only config keys are rejected. | Nested schema still reflects over runtime structs. | Prefer an explicit config-schema manifest. |

## Architecture Findings

### Config schema is safer, but not fully owned

`v0.56.0` had an unknown-key detector that logged unknown keys instead of
failing. `v0.60.0-rc.1` fails unknown keys and gives specific CLI-only errors
for `dry-run` and `dry-run-format`.

Remaining risk: nested fields still come from reflection over runtime structs.
A future exported field with a missing `toml:"-"` tag could accidentally become
public config.

Recommendation: define explicit job and watch config field allowlists, even if
they repeat a few field names.

### Job selection is more explainable

Runtime selected-job information existed before, but the release candidate makes
it user-visible and machine-visible. Dry-run text and JSON now include job,
framework, and reason.

Recommendation: keep selection reasons small and stable. Add new reason strings
only when a user-facing ambiguity requires them.

### Watch/session/planner architecture is the strongest improvement

Before, live watch and `watch find` each handled parts of job selection, watch
directory setup, ignores, cwd normalization, target planning, and execution
shape.

After, the shared shape is:

- `internal/watchsession.Session`: selected job, cwd, watch dirs, ignore
  patterns, planner, preview admission, and handler construction.
- `watch.Planner`: side-effect-free plan object with matched rules, existing
  targets, missing targets, reload intent, errors, and job plans.
- `watch.ExecutionPlan`: shared command shape for preview and live execution.

Recommendation: keep lifecycle concerns outside this core. Startup, debounce,
prompt, timeout, reload, and output belong at the command edge.

### Dry-run JSON and watch JSON are at useful boundaries

One-shot dry-run JSON is produced after the runner has built the same worker
commands used by execution. Watch JSON is produced after the shared watch
planner and execution-plan builder.

Recommendation: avoid structured error JSON unless a concrete automation use
case needs it.

### Help customization is useful, but string-fragile

The custom help layer improved first-contact UX, especially by hiding run-only
flags from watch surfaces. It also mutates Kong flag hidden state temporarily
and post-processes generated help text.

Recommendation: keep help customization thin. If it grows, consider an explicit
renderer for the high-traffic help screens instead of more Kong post-processing.

## Process Lessons

What worked:

- The scorecard kept non-code qualities visible.
- The T3 baseline was honest.
- T47 exposed that CLI polish had outpaced config architecture.
- T51 was the best architecture pivot: it named the planner plus session facade
  as the right boundary.
- T54/T56/T68/T69/T72/T74 were appropriately incremental and test-backed.

What created churn:

- `docs/goal/**` became useful forensics but weak durable architecture
  documentation.
- `new_design.md` became an append-only phase ledger.
- Repeated scorecards risked score inflation.
- Docs-only cleanup after docs-only cleanup added release archaeology cost.

Recommendations:

- Accept the `v0.60.0-rc.1` architecture direction.
- Consider explicit config schema field lists before final release.
- Keep `watchsession` as the command-facing watch facade.
- Keep JSON contracts narrow and versioned.
- Archive or relocate `docs/goal/**` after this assessment.
