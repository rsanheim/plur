# Plur CLI-UX Outcome Assessment

## Summary

The CLI-UX goal produced a real, user-visible improvement over `v0.56.0`.
Plur's daily path is now easier to discover, dry-run output explains what will
happen and why, watch previews are useful in both text and JSON, and the config
API is much harder to misuse.

The most important product shift is from implicit behavior to inspectable plans:

- `plur --help` now leads with commandless daily usage.
- `plur --dry-run` explains selected job, framework, reason, plan size, and the
  worker commands that would run.
- `plur --dry-run --dry-run-format=json` provides a versioned one-shot command
  plan.
- `plur watch find --format=json` provides a versioned watch-preview plan with
  `job_plans[].argv`, `env`, `cwd`, and `shell`.
- Live watch and `watch find` now share runtime config, event admission,
  planning, and execution-plan construction.
- TOML config now rejects unknown keys and CLI-only preview controls.

The process also overran its natural stopping point. By T50 the broad CLI UX was
already strong enough to stop or ship. T51-T75 produced good watch architecture,
but it was effectively a new focused parity goal. The final T75 reflection
closed that parity issue, not a broad release-readiness audit. This assessment
fills that missing broad audit.

## Evidence Base

Raw evidence is stored in [artifacts/](artifacts/README.md).

Key generated artifacts:

- [v0.56.0 top-level help](artifacts/v056-help.stdout.txt)
- [main top-level help](artifacts/main-help.stdout.txt)
- [v0.60.0-rc.1 top-level help](artifacts/v060-help.stdout.txt)
- [v0.56.0 dry-run output](artifacts/v056-dry-run.stdout.txt)
- [v0.60.0-rc.1 dry-run output](artifacts/v060-dry-run.stdout.txt)
- [v0.60.0-rc.1 dry-run JSON](artifacts/v060-dry-run-json.stdout.txt)
- [v0.60.0-rc.1 watch-find JSON](artifacts/v060-watch-find-json.stdout.txt)
- [git shortstat](artifacts/git-diff-shortstat-v0.56-to-v0.60rc1.txt)
- [cloc diff](artifacts/cloc-diff-v0.56.0-to-v0.60.0-rc.1.txt)
- [static check exits](artifacts/static-v060-go-test.exit.txt)
- [dry-run benchmark](artifacts/hyperfine-default-ruby-v056-v060.md)
- [one-spec benchmark](artifacts/hyperfine-default-ruby-run-v056-v060.md)

Sub-agent notes:

- [subagent-review-index.md](subagent-review-index.md)
- [agent-cli-ergonomics.md](agent-cli-ergonomics.md)
- [agent-config-architecture.md](agent-config-architecture.md)
- [agent-release-docs.md](agent-release-docs.md)
- [agent-process-adversarial.md](agent-process-adversarial.md)
- [agent-quality-metrics.md](agent-quality-metrics.md)

## Version Baselines

| Ref | Object / Commit | Role |
| --- | --- | --- |
| `v0.56.0` | tag object `6659c398`, commit `9662fd23` | Original version requested in the assessment README. |
| `main` | `3feb1146` | Cleaner baseline for isolating CLI-UX work from prior runtime-cache work. |
| `v0.60.0-rc.1` | tag object `60523f62`, commit `d39385cd` | CLI-UX release-candidate version. |
| `prep-goal` at assessment start | `f867bc4c` | One docs-prep commit beyond the RC tag. |

## Main Compared To The RC

`main` is already ahead of `v0.56.0` in some areas, including experimental
runtime-cache/RSpec-split work. It is therefore the cleaner baseline for the
focused CLI-UX goal.

The measured `main..v0.60.0-rc.1` diff is:

```text
113 files changed, 11606 insertions(+), 957 deletions(-)
```

Freshly built `main` still has the older CLI-UX shape:

- top-level help starts with `Usage: plur <command> [flags]`;
- `plur --dry-run --dry-run-format=json` exits with
  `unknown flag --dry-run-format`;
- `plur watch find --format=json ...` exits with `unknown flag --format`.

The RC adds the workflow-first help, dry-run JSON, watch-find JSON, stricter
config behavior, and shared watch planning surfaces described below.

## Before / After CLI Shape

### Top-Level Help

`v0.56.0` opened with command-first help:

```text
Usage: plur <command> [flags]
```

`v0.60.0-rc.1` opens with both commandless and command modes, then immediately
lists daily workflows:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur                                Run the detected test suite
  plur spec/calculator_spec.rb        Run one target
  plur test/calculator_test.rb        Run one Minitest target
  plur --dry-run                      Preview the one-shot test plan
  plur watch                          Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  Preview a watch file change
```

Assessment: the common path is no longer hidden behind a command-first model.

### Watch Help

`v0.56.0` watch help exposed run-mode flags like `--dry-run`, `--json`, and
`--workers`.

`v0.60.0-rc.1` watch help is mode-focused:

```text
Usage: plur watch [flags]
       plur watch find <changed-file> [flags]
       plur watch <command> [flags]

Common workflows:
  plur watch                          Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  Preview which tests a change runs
  plur --dry-run [patterns...]        Preview a one-shot test run
```

Assessment: watch is now presented as persistent watch plus side-effect-free
preview, not as a place where one-shot dry-run flags might apply.

### Dry-Run Text

`v0.56.0` dry-run output:

```text
plur version=v0.56.0
[dry-run] Running 13 specs [rspec] in parallel using 4 workers
[dry-run] Worker 0: ...
```

`v0.60.0-rc.1` dry-run output:

```text
plur version=v0.60.0-rc.1
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 13 specs [rspec] in parallel using 4 workers
[dry-run] Plan: 13 targets across 4 workers; no commands will run
[dry-run] Commands:
```

Assessment: the after state explains why the plan exists and that it is
side-effect-free.

### Dry-Run JSON

`v0.56.0` rejects the JSON dry-run flag:

```text
plur: error: unknown flag --dry-run-format
```

`v0.60.0-rc.1` emits a versioned command plan:

```json
{
  "version": 1,
  "mode": "spec",
  "job": {
    "name": "rspec",
    "framework": "rspec",
    "reason": "explicit_patterns"
  },
  "targets": ["spec/models/user_spec.rb"],
  "warnings": [],
  "workers": [
    {
      "index": 0,
      "targets": ["spec/models/user_spec.rb"],
      "argv": ["bundle", "exec", "rspec", "-r", "/home/yolo/.plur/formatter/json_rows_formatter.rb"],
      "env": ["PARALLEL_TEST_GROUPS=1", "TEST_ENV_NUMBER=1"]
    }
  ]
}
```

Assessment: one-shot runs became scriptable without parsing human text.

### Watch Find

`v0.56.0` rejects JSON watch preview:

```text
plur: error: unknown flag --format
```

`v0.60.0-rc.1` emits a versioned watch preview with final command plans:

```json
{
  "version": 1,
  "mode": "watch_find",
  "file": "lib/calculator.rb",
  "existing_targets": {
    "rspec": ["spec/calculator_spec.rb"]
  },
  "job_plans": [
    {
      "job": "rspec",
      "targets": ["spec/calculator_spec.rb"],
      "argv": ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
      "env": [],
      "cwd": "/Users/rsanheim/src/rsanheim/plur/fixtures/projects/default-ruby",
      "shell": "bundle exec rspec spec/calculator_spec.rb"
    }
  ],
  "exit_code": 0
}
```

Assessment: `watch find` now supports humans, scripts, and agents.

## Product Scorecard

Scores use the original criteria from `docs/goal/tx_score_card.md`.

| Category | T3 Baseline | Assessment Score | Evidence | Main Remaining Issue |
| --- | ---: | ---: | --- | --- |
| Obviousness | 3 | 4.5 | Workflow-first help and explicit dry-run selected-job output. | Commandless `plur` and explicit `spec` both remain visible. |
| Brevity / surface area | 3 | 4 | Help grouping and mode-specific flag pruning reduce confusion. | The command set is still broad. |
| Default quality | 3 | 4.5 | RSpec-first defaults, explicit warnings, and watch-find diagnostics. | Shared/helper watch mappings still require explicit config. |
| Conceptual coherence | 2 | 4.5 | Run mode appends targets; watch mode maps changes; both expose command plans. | One-shot JSON and watch JSON use different command surfaces. |
| Feedback quality | 2 | 4.5 | Plain command errors, no-op hints, strict config errors, warning paths. | Error JSON remains intentionally unstructured. |
| Composability | 3 | 4.5 | Versioned JSON plans with canonical `argv` and `env`. | Scripts must avoid parsing human `shell` strings. |
| Config/API cleanliness | 2 | 4 | Unknown keys and CLI-only TOML controls now fail. | Nested config schema still partly reflects runtime structs. |

## Top Improvements

1. Help now matches everyday use instead of implying a required subcommand.
2. Dry-run is a real plan view, not just worker command text.
3. Stable JSON output exists for one-shot run plans and watch previews.
4. Watch preview and live watch share the important code path for job selection,
   watch config, ignores, planning, and execution-plan construction.
5. Config mistakes fail early instead of silently degrading to defaults.
6. Output contracts are documented in one place.

## Remaining Design Problems

1. The CLI surface is clearer but still wide.
2. Internal goal docs are large and live in the public repo, though MkDocs
   excludes them.
3. Release docs and README have not fully caught up with the RC behavior.
4. Error-mode JSON remains intentionally absent; this is documented but less
   composable than success paths.
5. A tiny fixture benchmark shows no dry-run regression but a small one-spec
   run slowdown; broader performance evidence is still thin.

## What Should Not Change

- Keep commandless `plur` as the primary daily entry point.
- Keep `plur --dry-run` as the one-shot preview.
- Keep `plur watch find` side-effect-free.
- Keep `-C` as a first-class workflow.
- Keep RSpec-first autodetection when both `spec/` and `test/` exist.
- Keep strict config validation.
- Keep `argv` and `env` as the machine contract; keep `shell` as display text.

## Bottom Line

The CLI-UX work achieved the core product goal: everyday usage is more obvious,
more inspectable, harder to misuse, and much more scriptable. The process should
still be treated as a success with a process caveat: future goals need stronger
stop rules and a broader final audit before declaring the whole goal closed.
