# T34 Score Card - After Test Target, Output Contract, And JSON Cleanup

## Context

This reflection covers:
- T31: improved bare `plur test --dry-run` feedback.
- T32: clarified output contracts around exit code 1, JSON-mode command errors,
  and `argv`/`env` versus `shell`.
- T33: removed the unused global `--json` file-output flag.

Inputs:
- Original baseline: `docs/goal/current_design.md`.
- Latest design notes: `docs/goal/new_design.md`.
- Current executable help and output checks.
- Reviewer feedback from Archimedes, Beauvoir, and Volta.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `plur --help` leads with `Usage: plur [patterns...] [flags]` and common workflows. `./plur test --dry-run` now says `test` is a target path, not a command. | `spec` is still both a visible command and the implicit generic run path. | Keep improving examples and command-specific help around commandless usage. | Over-customizing help can drift from Kong behavior. |
| Brevity / surface area | 4 | T33 removed `--json` from top-level, `spec`, `watch`, and `watch run` help. Commands are grouped into daily versus advanced/setup. | Global flags still dominate first-contact help. | Slim or group global flags by common versus advanced usage. | Hiding too much can make discoverability worse for experienced users. |
| Default quality | 4 | Commandless `plur`, RSpec-first detection, `-C`, dry-run, and `watch find` all work with useful defaults. | Human dry-run still becomes a wall of worker commands for larger suites. | Add a compact dry-run summary before detailed worker commands. | Less immediate copy/paste detail unless the detailed section remains visible. |
| Conceptual coherence | 4 | T31 made `test` consistently a target path; T33 leaves two real JSON preview APIs: `--dry-run-format=json` and `watch find --format=json`. | One-shot JSON and watch JSON use different flag shapes. | Either document the difference with examples or later converge names if a clean model appears. | Renaming structured-output flags would be a breaking change. |
| Feedback quality | 4 | Missing bare `test` errors are direct. Command errors use plain `Error:` output. Watch no-op and exclude warnings explain what happened. | Parser errors remain generic; removed `--json` currently suggests `--job`. | Add a targeted pre-parse error for removed `--json` pointing to the real JSON preview APIs. | More pre-parse special cases add maintenance weight. |
| Composability | 4 | Dry-run JSON emits clean stdout with `workers[].argv` and `workers[].env`; watch-find JSON includes `exit_code`. T32 documents stderr-only command errors. | JSON-mode failures are plain stderr with empty stdout rather than structured errors. | Either keep the stderr-only contract explicit with examples, or add structured error JSON. | Structured error envelopes would expand the stable contract. |
| Config/API cleanliness | 4 | Removing unused `--json` deleted a false config/API surface. Run-mode `{{target}}` rejection is documented and tested. | Config docs remain broad, and target substitution semantics still require learning run versus watch modes. | Keep config reference compact and prefer examples that show current supported behavior only. | Oversimplifying docs could hide legitimate custom-job use cases. |

## Reviewer Summary

Archimedes:
- Scores: all 4s.
- Direction: yes, trending right.
- Main concerns: human dry-run is worker-command-heavy, `--json` parser hint is
  poor, and advanced flags still appear early.

Beauvoir:
- Scores: all 4s.
- Direction: T31-T33 improved automation/docs.
- Main concerns: `current_design.md` is now stale as a present-tense document,
  JSON-mode errors remain stderr-only, and removed `--json` has a poor parser
  suggestion.

Volta:
- Scores: all 4s.
- Direction: clear improvement for agents and CI.
- Main concerns: mixed non-zero exit-code meanings, JSON-mode error handling,
  and `watch run --help` still describes `--dry-run` generically even though
  watch rejects it.

## Evidence

Top-level help now has no unused `--json` flag:

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

Bare `test` target feedback is now explicit:

```text
plur version=v0.56.1-0.20260523140632-fa24dea847e7+dirty
Error: file not found: test; `test` is a target path, not a Plur command. Create a test/ directory or pass a Minitest target like test/calculator_test.rb
```

Removed `--json` no longer silently does nothing:

```text
plur: error: unknown flag --json, did you mean "--job"?
```

Dry-run JSON remains clean stdout with status on stderr:

```text
exit=0
stdout:
{
  "version": 1,
  "mode": "spec",
  "job": {
    "name": "rspec",
    "framework": "rspec",
    "reason": "explicit_patterns"
  },
  "targets": [
    "spec/integration/spec/help_spec.rb"
  ],
  "warnings": [],
  "workers": [
```

Watch-find JSON no-op is structured and exits 2:

```text
exit=2
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

`watch run --help` still has generic one-shot dry-run wording:

```text
--dry-run                  Print what would be executed without running
--dry-run-format="text"    Dry-run output format: text or json
```

T33 verification before commit:

```text
bin/rake
```

`bin/rake` passed with 367 examples, 0 failures, and 4 existing pending
examples.

## Are We Moving In The Right Direction?

Yes. The recent work removed false affordances and made the command model more
honest: `test` is not a command, output contracts say what scripts can depend
on, and the unused `--json` surface is gone. The interface is smaller and less
misleading than it was at T30.

## Top Design Problems

1. Removed `--json` now fails correctly, but the parser hint points to `--job`
   instead of the real JSON preview APIs.
2. Human dry-run remains too command-heavy for first-pass comprehension.
3. `watch run --help` still displays generic one-shot dry-run wording, while
   `watch --help` has the better watch-specific wording.
4. JSON success paths are well structured, but JSON-mode planning/config errors
   are still plain stderr with empty stdout.
5. The original `current_design.md` is now useful as a baseline, not as a live
   present-tense description.

## Recommended Next Changes

1. T35-DEV: add a targeted removed-`--json` error that points users to
   `plur --dry-run --dry-run-format=json` and
   `plur watch find --format=json`.
2. T36-DEV: make `watch run --help` annotate `--dry-run` and
   `--dry-run-format` the same way `watch --help` does.
3. T37-DEV: make human dry-run more skim-friendly by adding a compact summary
   before worker commands.
4. Later: decide whether JSON-mode command errors should stay stderr-only or
   gain a structured error envelope.

## Things That Should Not Change

- Commandless `plur` should remain the primary everyday entry point.
- RSpec-first detection for mixed `spec/` and `test/` projects should stay.
- Dry-run JSON should keep `argv` and `env` as the canonical script fields.
- `watch find --format=json` should keep returning structured no-op previews
  with exit code 2.
- The unused `--json` file-output flag should stay removed until a real feature
  exists.

## Done-Done Status

Not done. The design is consistently better and reviewers now score every
category at 4, but the remaining issues are still visible in everyday help,
parser errors, and agent-facing failure paths. Continue the DEV loop.
