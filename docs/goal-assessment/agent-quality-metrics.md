# Agent Quality Metrics Assessment

Reviewer: Sagan
Persona: QA/performance/composability analyst
Model: `gpt-5.5`, reasoning `xhigh`
Mode: read-only assessment

## Scope And Caveats

Sagan focused on measurable outcome evidence: changed files, lines of code,
docs/tests, static-analysis signals, machine-readable output contracts,
performance benchmark possibilities, and residual risk.

The agent's own probes used the local `./plur` binary and treated them as smoke
checks. The synthesized assessment in `metrics.md` also uses fresh local builds
from `v0.56.0` and `v0.60.0-rc.1`.

## Ref Baseline

| Ref | Note |
| --- | --- |
| `v0.56.0` | Original version requested. |
| `main` | Cleaner CLI-UX-specific baseline than `v0.56.0` for some metrics. |
| `v0.60.0-rc.1` | RC tag for the goal outcome. |
| `prep-goal` | One commit ahead of the RC tag at assessment start. |

Important attribution note: `v0.56.0..v0.60.0-rc.1` includes runtime cache /
RSpec split work in addition to CLI-UX work. `main..v0.60.0-rc.1` is the
cleaner CLI-UX delta.

## Change Volume

Sagan reported:

| Comparison | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `v0.56.0..v0.60.0-rc.1` | 148 | 15,152 | 1,222 |
| `main..v0.60.0-rc.1` | 113 | 11,606 | 957 |

Bucketed `main..v0.60.0-rc.1` diff:

| Bucket | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| docs | 39 | 7,030 | 510 |
| tests | 38 | 2,310 | 102 |
| code | 27 | 1,897 | 332 |
| fixtures | 6 | 17 | 13 |
| tooling | 3 | 352 | 0 |

The main synthesis recalculated these buckets slightly differently because it
counts `README.md`, `CHANGELOG.md`, and `mkdocs.yml` as docs, but both views
show the same shape: documentation and tests account for most of the change
volume.

## Outcome Evidence

The strongest measurable outcomes are composability and misuse-resistance:

- `docs/output-contracts.md` added stable JSON/stdout/stderr reference material.
- `spec/integration/spec/dry_run_plan_spec.rb` covers one-shot JSON plan shape,
  warnings, configured env, env dedupe, and parser misuse.
- `spec/integration/watch/watch_find_json_spec.rb` covers runnable, no-rule,
  ignored, no-watch-mapping, and invalid-job JSON paths.
- `internal/watchsession/session_test.go` verifies shared session setup,
  admission, planner, execution-plan construction, env dedupe, and edge cases.
- `internal/kongtoml/kongtoml_test.go` rejects unknown TOML keys and CLI-only
  persistent config keys.

## Machine-Readable Contracts

Smoke probes showed these intended contracts:

- `plur --dry-run --dry-run-format=json` returns `version: 1`, `mode: spec`,
  selected `job`, explicit `targets`, and worker `argv`/`env`.
- `plur watch find --format=json <file>` returns `mode: watch_find`,
  `existing_targets`, `job_plans`, and `exit_code`.
- No-rule watch-find JSON returns exit code 2 with no job plans.

Residual contract risk: JSON success paths are strong, but parser/config/runtime
error modes intentionally remain prose on stderr with empty stdout.

## Static And Tooling Signals

Available gates include:

- `lint:go`
- `lint:ruby`
- `lint:shell`
- `toolchain:check`
- `test:go`
- `test:default_ruby`
- `test`
- `test:all`
- `vuln:check`

Tooling changes:

- `.mise.toml` adds `shellcheck = "0.11.0"`.
- `script/cli-inventory` adds repeatable CLI inventory coverage.
- `script/track-goal` adds logfmt-style phase tracking.

Negative static signal:

- `git diff --check main..v0.60.0-rc.1^{}` fails on trailing whitespace in
  internal goal docs.

## Performance Evidence

Sagan recommended `script/bench-git` for a stronger performance bundle. The
main synthesis ran a smaller benchmark and stores those outputs under
`docs/goal-assessment/artifacts/`.

Existing performance docs for runtime cache/RSpec split do not prove CLI-UX
performance broadly. The assessment should avoid broad speed claims.

## Process Evidence

Strengths:

- `tracking.md` contains start/done entries through T75.
- `docs/goal/**` plus `tracking.md` provide unusually strong traceability.
- The process recorded real blockers, including agent thread-limit failures.
- A concrete review issue was caught and fixed: execution plan job key
  preservation.
- T74 recorded `bin/rake` passing with 386 examples, 0 failures, and 4 pending;
  T75 recorded `script/check-links` passing.

Weak points:

- Sub-agent claims were summarized in docs, but raw transcripts and model
  identities were not machine-verifiable from repo artifacts.
- Full test/build pass evidence was recorded in markdown, not terminal logs.
- Performance evidence was insufficient for broad "same or better" claims.

## Residual Risk Summary

| Risk | Severity | Evidence |
| --- | --- | --- |
| Limited fresh performance benchmark | High | Fixture-only benchmark evidence. |
| `git diff --check` fails on historical docs | Medium | Trailing whitespace in goal docs. |
| Error JSON remains unstructured | Medium | Documented prose-stderr behavior. |
| Sub-agent evidence not preserved historically | Medium | Summaries existed, raw notes did not. |
| Internal goal docs are large | Low/Medium | `docs/goal/**` is process-heavy. |
