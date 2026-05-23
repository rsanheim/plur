# T47 Score Card - After Output And Watch Flag Cleanup

Status: verified
Commit: pending

## Context

This reflection covers the DEV loop after T42:

- T43: routed worker startup/runtime errors to stderr.
- T44: included configured job env in dry-run JSON.
- T45: kept `watch find --format=json` structured when no watch mappings exist.
- T46: focused `watch run` help and rejected explicit no-op one-shot runner
  flags.

Inputs:

- Baseline: `docs/goal/current_design.md`.
- Previous reflection: `docs/goal/t42_score_card.md`.
- Latest design notes: `docs/goal/new_design.md`.
- Current executable checks from `./plur`.
- Reviewer feedback from Carver, Descartes, and Helmholtz.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `./plur --help` starts with `Usage: plur [patterns...] [flags]`, common workflows, commandless run, one-file run, `--dry-run`, `watch`, and `watch find`. | Shared helper files still look like accidental no-ops: `watch find spec/spec_helper.rb` only says `No matching rule`. | Add built-in RSpec helper watch behavior or clearer no-rule guidance for common helper files. | Running too much on helper edits can make watch mode expensive. |
| Brevity / surface area | 4 | `./plur watch --help` and `./plur watch run --help` no longer show `--workers`, `--first-is-1`, `--rspec-split`, `--dry-run`, or `--dry-run-format` as watch flags. | Top-level help still necessarily exposes both run and watch concepts, and `spec` remains a visible command plus implicit default. | Keep pruning mode-specific flags and avoid adding command names. | More custom help filtering can drift from Kong's generated help. |
| Default quality | 4 | `./plur -C fixtures/projects/default-ruby --dry-run` selects `rspec`, explains `reason: autodetect`, finds 13 specs, and says no commands will run. | Generated config templates can create watch jobs whose actual execution drops changed targets if `cmd` lacks `{{target}}`. | Make generated configs and watch execution semantics agree. | Changing watch command semantics is a breaking config/API change. |
| Conceptual coherence | 3 | One-shot preview is consistently `plur --dry-run`; watch preview is `plur watch find`; `watch run --workers=99` now errors before starting watch mode. | `job.cmd` still means "append targets" in one-shot run mode, but watch mode only passes targets when `cmd` contains `{{target}}`. | Unify run and watch target passing semantics, or make `watch find` report the exact command shape it previews. | A clean break may surprise existing users relying on whole-suite watch jobs. |
| Feedback quality | 3 | Dry-run says selected job/reason, watch flag misuse errors directly, and JSON no-op previews include `exit_code: 2`. | `watch find` text mode exits 0 for no configured mappings while JSON exits 2, and startup failures still pair stderr errors with a stdout `0 examples, 0 failures` summary. | Normalize no-mapping text exit code to 2 and make never-started worker summaries less successful-looking. | Tightening exit codes may affect scripts that treated empty watch config as success. |
| Composability | 4 | Dry-run JSON and watch-find JSON parse cleanly from stdout; configured job env now appears in `workers[].env`; no-mapping watch JSON returns structured exit 2. | `workers[].env` can contain duplicate keys when configured env overrides Plur-managed env, requiring implicit last-wins behavior. | Dedupe dry-run JSON env by key and document override order. | Dedupe order must match actual `exec.Cmd` environment behavior. |
| Config/API cleanliness | 2 | Watch help is focused, but `config_init.go` templates define `cmd` without `{{target}}`, `watch/watcher.go` drops targets for such jobs, and `internal/kongtoml/kongtoml.go` only debug-logs unknown config keys. | Config is still too easy to misuse: unknown keys are ignored, operational flags can be persisted, and watch target passing depends on hidden token semantics. | Reject unknown config keys and split persistable project config from ephemeral CLI/session controls. | Stricter config validation is a breaking change for sloppy existing configs. |

## Reviewer Summary

Carver, first-contact Ruby developer:
- Scores: mostly 4s, composability 5.
- Direction: the first path is now coherent: `plur`, `plur FILE`,
  `plur --dry-run`, `plur watch`, `plur watch find FILE`.
- Main concerns: shared helper files like `spec/spec_helper.rb` feel like a
  no-op, and human dry-run still exposes formatter internals by default.

Descartes, automation / agent workflow reviewer:
- Scores: all 4s.
- Direction: T43-T46 fixed the T42 automation-contract concerns.
- Main concerns: text no-mapping watch find exits 0 while JSON exits 2,
  dry-run JSON env can contain duplicate keys, JSON-mode errors remain plain
  stderr with empty stdout, and worker startup errors still have a
  successful-looking stdout summary.

Helmholtz, maintainer / config API reviewer:
- Scores: Obviousness 4, Brevity 4, Default quality 3, Conceptual coherence 3,
  Feedback quality 3, Composability 4, Config/API cleanliness 2.
- Direction: CLI surfaces improved, but the config/API model is not yet hard
  to misuse.
- Main concerns: `watch find` can say a target would run while actual watch
  execution drops targets if the job command lacks `{{target}}`; unknown config
  keys are ignored; action/debug flags can be persisted in config.

## Evidence

Top-level help is clearer:

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

Watch help is focused on live-watch controls:

```text
Usage: plur watch run [flags]

Flags:
  -u, --use=""               Job to use (overrides autodetection)
      --ignore=IGNORE,...    Patterns to ignore from watch events
      --timeout=INT          Exit after specified seconds
      --debounce=30          Debounce delay in milliseconds
```

Dry-run text now gives a compact plan:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 13 specs [rspec] in parallel using 4 workers
[dry-run] Plan: 13 targets across 4 workers; no commands will run
[dry-run] Commands:
```

Dry-run JSON is structured and includes env:

```json
{
  "version": 1,
  "mode": "spec",
  "job": {
    "name": "rspec",
    "framework": "rspec",
    "reason": "explicit_patterns"
  },
  "workers": [
    {
      "env": [
        "PARALLEL_TEST_GROUPS=1",
        "TEST_ENV_NUMBER=1"
      ]
    }
  ]
}
```

Watch no-op JSON is structured:

```text
status=2
stdout:
{
  "version": 1,
  "mode": "watch_find",
  "file": "spec/spec_helper.rb",
  "matched_rules": [],
  "existing_targets": {},
  "missing_targets": {},
  "exit_code": 2
}
```

`watch run` no-op runner flags now fail before watch starts:

```text
plur: error: --workers does not apply to plur watch run; watch run executes configured watch jobs directly and does not use one-shot parallel runner flags
```

The biggest newly exposed config/API issue is watch target semantics:

```go
// watch/watcher.go
// Jobs without {{target}} placeholder run once without targets
if !j.UsesTargets() {
    cmd := j.Cmd
    ...
}
```

But generated config templates define watch jobs without `{{target}}`:

```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec"]

[[watch]]
name = "spec-files"
source = "spec/**/*_spec.rb"
jobs = ["rspec"]
```

## Are We Moving In The Right Direction?

Yes. The T42 contract gaps were addressed: startup errors moved to stderr,
dry-run JSON now includes configured env, no-mapping watch JSON is structured,
and `watch run` no longer advertises or accepts one-shot runner flags.

The course correction is that the next loop should shift from output surfaces
to config/API correctness. The human-facing CLI is now mostly a 4, while the
config model still has hidden semantics and permissive validation that can make
reasonable configs do the wrong thing.

## Top Design Problems

1. Watch and one-shot run target semantics still diverge for `job.cmd`.
2. `watch find` previews targets, but actual watch execution can drop those
   targets for commands without `{{target}}`.
3. Unknown config keys are debug-logged, not rejected.
4. Operational/session flags such as `dry-run`, `dry-run-format`, `debug`, and
   `verbose` can be persisted in config.
5. `watch find` text no-mapping behavior exits 0 while JSON exits 2.
6. Dry-run JSON env can contain duplicate keys when configured env overrides
   managed env.

## Recommended Next Changes

1. T48-DEV: unify watch and one-shot run target semantics, likely by appending
   targets in watch mode unless `{{target}}` customizes placement.
2. T49-DEV: normalize `watch find` text no-mapping exit code to 2 and document
   it.
3. T50-DEV: dedupe dry-run JSON env by key so scripts do not need implicit
   last-wins semantics.
4. T51-DEV: reject unknown config keys, including typos in nested job/watch
   sections.
5. T52-DEV: separate persistable project config from ephemeral CLI/session
   controls.

## Things That Should Not Change

- Commandless `plur` should remain the primary daily entry point.
- `plur --dry-run` should remain the one-shot preview surface.
- `plur watch find --format=json` should keep structured no-op previews with
  exit code 2.
- Watch help should stay focused on live-watch controls.
- The output contract doc should remain the canonical machine-output reference.

## Done-Done Status

Not done. The human CLI and automation surfaces are substantially better than
T42, but the latest reflection still has Config/API cleanliness at 2 and
Conceptual coherence / Feedback quality at 3. Continue with DEV work focused
on watch/run target semantics and stricter config behavior.

## T47 Validation

```text
script/check-links
bin/rake
```

`script/check-links` passed. `bin/rake` passed with 377 examples, 0 failures,
and 4 existing pending examples.
