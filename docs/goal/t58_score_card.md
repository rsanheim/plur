# T58 Score Card - After Watch Planning Parity

Status: verified
Commit: d1ad78ac594b25b6a7946964d198c9754faa90a6

## Context

This reflection covers the DEV/ARCH loop after T47:

- T48: watch jobs append resolved targets consistently with one-shot run jobs.
- T49: text `watch find` no-watch-mapping cases exit 2 like JSON.
- T50: dry-run JSON env entries are deduped by key with final value winning.
- T51: architecture review for `watch find` and live `watch` parity.
- T52: characterization tests for existing watch planning behavior.
- T53: extracted live watch event admission.
- T54: extracted side-effect-free `watch.Planner`.
- T55: `watch find` renders from `watch.Planner`.
- T56: live watch and `watch find` share `internal/watchsession` setup.
- T57: added live-vs-find parity coverage at session and CLI boundaries.

Inputs:

- Baseline: `docs/goal/current_design.md`.
- Previous reflection: `docs/goal/t47_score_card.md`.
- Latest design notes: `docs/goal/new_design.md`.
- Current executable artifacts under `tmp/t58_reflect/`.
- Reviewer feedback from Dewey, James, and Raman.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `tmp/t58_reflect/plur_help.out` leads with `Usage: plur [patterns...] [flags]` and common workflows for `plur`, one-file runs, `--dry-run`, `watch`, and `watch find`. | Common helper/support file changes still feel like dead ends: `watch find spec/spec_helper.rb` only says `No matching rule`. | Add clearer no-rule guidance for common helper files, or add a conservative built-in helper-file behavior. | Running broad suites for helper edits can make watch mode expensive or noisy. |
| Brevity / surface area | 4 | `tmp/t58_reflect/watch_help.out` keeps watch help focused on `watch`, `watch find`, and live-watch flags; one-shot worker flags are gone from watch help. | The top-level surface still includes implicit commandless run, visible `spec`, `watch run`, `watch find`, config, rails/rake, and advanced setup commands. | Keep pruning mode-specific flags and avoid adding new commands unless they collapse real complexity. | More custom help filtering can drift from Kong output. |
| Default quality | 4 | `tmp/t58_reflect/dry_run.err` selects `rspec` by autodetect, finds 13 specs, and says no commands will run; T48 fixed watch target passing for generated-style jobs. | Helper/support files are not covered by default watch mappings, so important edits can still no-op. | Add helper-file guidance or a default mapping that makes the intended behavior explicit. | Better helper defaults may run more tests than users expect. |
| Conceptual coherence | 4 | T56/T57 route live watch and `watch find` through shared `watchsession` and `watch.Planner`, and `spec/integration/watch/watch_find_live_parity_spec.rb` proves the preview target matches live execution. | One-shot dry-run and watch-find plans remain separate public shapes, and `watch find` still has a small no-watch branch before session setup. | Use the shared planner to show an exact watch execution plan, and consider a session-shaped no-watch helper. | Expanding JSON contracts requires versioned care. |
| Feedback quality | 4 | Dry-run text says selected job/reason and plan size; `watch find` text/JSON exit 2 for no runnable target and machine JSON is clean on stdout. | No-rule feedback is accurate but thin; it does not tell users what to do next. | Make no-rule watch output actionable without printing noisy explanations for every editor save burst. | Extra hints can clutter watch output during rapid changes. |
| Composability | 4 | `tmp/t58_reflect/dry_run_json.out` and `tmp/t58_reflect/watch_find_no_rule_json.out` are parseable stdout JSON with stable mode/version fields and exit codes; dry-run JSON env dedupe now matches final-value behavior. | JSON-mode errors are still plain stderr, and watch-find JSON exposes targets rather than final argv/env/cwd command plans. | Either document stderr-only JSON errors as final, or add a small structured error shape; expose watch execution plans in JSON. | More machine fields are commitments that must be maintained. |
| Config/API cleanliness | 3 | Target semantics improved, and config docs now describe watch target appending. Unknown TOML keys are still only debug-logged in `internal/kongtoml/kongtoml.go`, and operational flags remain config-addressable. | Config is still permissive and can silently accept typos or persist session controls. | Reject unknown config keys and split or reject ephemeral CLI/session controls in project config. | Stricter validation is a breaking change for sloppy existing configs. |

## Reviewer Summary

Dewey, everyday CLI reviewer:
- Scores: Obviousness 4, Brevity 4, Default quality 4, Conceptual coherence
  4, Feedback quality 4, Composability 4, Config/API cleanliness 3.
- Direction: T51-T57 removed a trust hazard. `watch find` is now a believable
  diagnostic because its preview is covered against live execution.
- Main concerns: helper/support file edits are still dead ends, and `watch
  find` previews targets rather than the actual final command.

James, architecture/config reviewer:
- Scores: all 4 except Config/API cleanliness at 3.
- Direction: `internal/watchsession` is the right command boundary and now owns
  job selection, cwd normalization, watch dirs, ignores, planner, event
  admission, and handler setup.
- Main concerns: `watch.Planner` currently swallows `FindTargetsForFile`
  errors, `watch find` keeps a no-watch special case, unknown config keys are
  debug-logged, and runtime config still mixes resolved API, built-ins, and
  diagnostics.

Raman, docs/composability reviewer:
- Scores: all 4 except Config/API cleanliness at 3.
- Direction: public output contracts and help are now much clearer, and JSON
  contracts are useful for agents.
- Main concerns: `docs/goal/*` is internal planning material living under the
  public docs tree, `configuration.md` mixes Diataxis roles, unknown config
  behavior is risky, and JSON error behavior remains less composable.

## Evidence

Top-level help remains aligned with daily workflows:

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

Watch help keeps one-shot preview separate from persistent watch:

```text
Usage: plur watch [flags]
       plur watch find <changed-file> [flags]
       plur watch <command> [flags]

Common workflows:
  plur watch                          Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  Preview which tests a change runs
  plur --dry-run [patterns...]        Preview a one-shot test run
```

Dry-run text is explicit about selection and execution:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 13 specs [rspec] in parallel using 4 workers
[dry-run] Plan: 13 targets across 4 workers; no commands will run
[dry-run] Commands:
```

Dry-run JSON stays script-friendly:

```json
{
  "version": 1,
  "mode": "spec",
  "job": {
    "name": "rspec",
    "framework": "rspec",
    "reason": "explicit_patterns"
  },
  "targets": [
    "spec/models/user_spec.rb"
  ],
  "warnings": []
}
```

Watch no-rule preview is structured and exits 2:

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

The main watch parity invariant is now executable:

```text
spec/integration/watch/watch_find_live_parity_spec.rb
internal/watchsession/session_test.go
```

## Are We Moving In The Right Direction?

Yes. The T51 concern was real: sharing `FindTargetsForFile` was too low-level
to guarantee parity between `plur watch find FILE` and live `plur watch`. The
current code now shares a command-facing session and a pure planner, and T57
proves a watch-find preview target matches live watch execution for the same
changed path.

The course correction is to stop doing invisible parity work for a moment and
cash out the new planner in user-facing output. `watch find` should show the
exact command plan it is previewing, not just matched rules and target lists.
After that, the next major weakness is config/API strictness.

## Top Design Problems

1. `watch find` still previews targets, not final argv/env/cwd command plans.
2. Common RSpec helper/support files produce terse no-rule output and no next
   action.
3. Unknown config keys are accepted with only debug logging.
4. Ephemeral CLI/session controls can still be persisted in config.
5. JSON-mode error behavior is plain stderr rather than a structured contract.
6. Internal goal/planning docs remain under `docs/goal`, which is awkward for
   public documentation hygiene.

## Recommended Next Changes

1. T59-DEV: make `watch find` show exact command plans from the shared
   `watch.Plan` job plans, in text and JSON.
2. T60-DEV: improve no-rule watch guidance for common helper/support files.
3. T61-DEV: reject unknown config keys instead of debug-logging them.
4. T62-DEV: reject or split ephemeral CLI/session controls from project config.
5. T63-DEV: decide the JSON-mode error contract.
6. T64-DEV: move or exclude internal goal docs from public docs publishing.

## Things That Should Not Change

- Commandless `plur` should remain the primary daily entry point.
- `plur --dry-run` should remain the one-shot preview surface.
- `plur watch find --format=json` should keep structured no-op previews with
  exit code 2.
- `watch find` and live watch should continue to share the session/planner
  path.
- Output contracts should remain the canonical machine-output reference.

## Done-Done Status

Not done. Most user-facing categories are now 4s, but Config/API cleanliness is
still 3, and the latest reviewers agree that hidden config permissiveness and
thin watch no-rule feedback remain. Continue with DEV work.

## T58 Validation

Fresh executable evidence was captured under `tmp/t58_reflect/`:

```text
bin/rake build
./plur --help
./plur watch --help
./plur -C fixtures/projects/default-ruby --dry-run
./plur -C fixtures/projects/default-ruby --dry-run --dry-run-format=json spec/models/user_spec.rb
./plur -C fixtures/projects/default-ruby watch find spec/spec_helper.rb
./plur -C fixtures/projects/default-ruby watch find --format=json spec/spec_helper.rb
```

Exit statuses:

```text
dry_run=0
dry_run_json=0
plur_help=0
watch_help=0
watch_find_no_rule=2
watch_find_no_rule_json=2
```

Final verification:

```text
script/check-links
bin/rake
```

`script/check-links` passed. `bin/rake` passed with 379 examples, 0 failures,
and 4 existing pending examples.
