# T63 Score Card - After Config Cleanup

Status: verified
Commit: 22015d45b1c01655b36abdb2f32be8a863905cb9

## Context

This reflection covers the focused DEV loop after T58:

- T59: `watch find` text and JSON now show command plans.
- T60: no-rule shared helper changes now print a concise `[[watch]]` hint.
- T61: unknown TOML keys now fail fast with config path and key path.
- T62: `dry-run` and `dry-run-format` are rejected in config as CLI-only
  preview controls.

Inputs:

- Current executable output from `./plur`.
- Latest design notes in `docs/goal/new_design.md`.
- Public contracts in `docs/output-contracts.md` and `docs/configuration.md`.
- Reviewer feedback from Feynman, Copernicus, and Boole.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | Top-level help leads with `plur`, one-target runs, `--dry-run`, `watch`, and `watch find`; helper files now print a no-rule hint. | The command list still has several advanced/setup entries visible. | Keep daily workflows first and avoid adding more command nouns. | Hiding advanced commands too much can make setup harder to discover. |
| Brevity / surface area | 4 | T59-T62 added no new user commands and tightened existing `watch find` and config behavior. | The surface still includes commandless run, explicit `spec`, watch subcommands, Rails/Rake, doctor, config, and version. | Prefer pruning or scoping flags over adding commands. | More custom help can drift from Kong defaults. |
| Default quality | 4 | `plur -C fixtures/projects/default-ruby --dry-run` autodetects RSpec, finds 13 specs, and shows the plan before execution. | Shared helper edits still do not run tests by default. | Keep the helper hint, or later add an explicit starter watch mapping example. | Running broad suites on helper edits can be expensive. |
| Conceptual coherence | 4 | `watch find --format=json lib/calculator.rb` includes final `job_plans` with `argv`, `env`, `cwd`, and `shell`, matching live watch execution shape. | One-shot JSON still uses `--dry-run-format=json`, while watch preview uses `watch find --format=json`. | Leave both documented unless a later clean break is worth it. | Renaming JSON flags would be disruptive. |
| Feedback quality | 4 | `spec/spec_helper.rb` no-rule output now says to add `[[watch]]`; unknown config and CLI-only config errors name the file and key. | JSON-mode errors are still prose on stderr. | Add structured error envelopes only if scripts need them. | More machine contracts raise maintenance cost. |
| Composability | 4 | Dry-run JSON and watch-find JSON expose stable `argv`/`env`; watch find no-op exits 2 with clean stdout JSON. | Successful JSON is strong, but parser/config errors are not structured JSON. | Add a successful `watch find` JSON example with `job_plans` to output contracts. | Docs-only clarity may be enough; schema expansion may be unnecessary. |
| Config/API cleanliness | 3 | T61/T62 fixed the worst hazards: unknown keys and persisted dry-run controls. | The allowed TOML key set is still derived from the full Kong CLI model, then patched with a small CLI-only denylist. | Define a durable config schema allowlist independent from transient CLI flags. | A stricter schema is a clean break for configs using undocumented flags. |

## Reviewer Summary

Feynman, everyday CLI lens:
- Scores every category at 4.
- Main concern: helper-file feedback still requires users to know how to write a
  `[[watch]]` mapping.

Copernicus, config/API architecture lens:
- Scores every category at 4 and says the goal could stop on the threshold.
- Main risks: config schema validation is still coupled to Kong reflection,
  watch docs and accepted edge cases differ, planner errors can be quiet, and
  JSON errors are prose.

Boole, agent/composability/docs lens:
- Scores every category at 4 except Config/API cleanliness at 3.
- Main concern: strict config catches typos, but the persistent config API still
  follows the CLI model more than a documented durable schema.

## Evidence

Top-level help remains workflow-first:

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

Shared helper changes now get actionable no-rule feedback:

```text
[watch] Checking spec/spec_helper.rb
[watch] No matching rule for spec/spec_helper.rb
[watch] Hint: add a [[watch]] mapping for shared files if this change should run tests.
```

Watch find JSON now includes command plans:

```json
"job_plans": [
  {
    "job": "rspec",
    "targets": ["spec/calculator_spec.rb"],
    "argv": ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
    "env": [],
    "cwd": "/Users/rsanheim/src/rsanheim/plur/fixtures/projects/default-ruby",
    "shell": "bundle exec rspec spec/calculator_spec.rb"
  }
]
```

Config feedback is much stricter:

```text
plur: error: Configuration error: /Users/rsanheim/src/rsanheim/plur/tmp/t63_bad_config.toml contains unknown config key: wokers
```

```text
plur: error: Configuration error: /Users/rsanheim/src/rsanheim/plur/tmp/t63_cli_only_config.toml contains CLI-only config keys: dry-run, dry-run-format; pass these as command-line flags instead
```

Verification already run on the current code before this reflection:

- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 382 examples, 0 failures, and 4 existing pending
  examples.

## Next Recommended Changes

1. T64-DEV: define a durable TOML config schema allowlist independent from the
   full CLI model.
2. T65-DEV: add a successful `watch find --format=json` output-contract example
   showing `job_plans`.
3. T66-DEV: decide whether `watch.Planner` should surface target resolution
   errors instead of quietly skipping them.

## Conclusion

The CLI and watch UX are now consistently at 4. The remaining blocker for the
overall goal is Config/API cleanliness: it is stricter than before, but still
not clean enough to call the persistent config API fully deliberate.
