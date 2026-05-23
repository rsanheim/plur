# T38 Score Card - After JSON Guidance, Watch Help, And Dry-Run Summary

## Context

This reflection covers:
- T35: added direct guidance for the removed `--json` flag.
- T36: aligned `plur watch run --help` dry-run wording with `plur watch --help`.
- T37: added a skimmable human dry-run plan summary before worker commands.

Inputs:
- Original baseline: `docs/goal/current_design.md`.
- Previous reflection: `docs/goal/t34_score_card.md`.
- Latest design notes: `docs/goal/new_design.md`.
- Current executable checks.
- Reviewer feedback from McClintock, Turing, and Heisenberg.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `plur --help` leads with commandless usage and common workflows. Text dry-run now shows selected job, run count, plan summary, and command details. | `spec` remains a visible command while commandless `plur` is the everyday path. | Keep examples focused on commandless use and explicit target paths. | Hiding `spec` too much may make command help harder to find. |
| Brevity / surface area | 4 | The unused `--json` flag is gone, and removed-flag guidance points to real APIs. | Top-level help still shows several advanced/global flags before users ask for them. | Split common flags from advanced/debug flags in help. | More custom help code can drift from Kong output. |
| Default quality | 4 | Commandless RSpec discovery, RSpec-first mixed-project behavior, `-C`, dry-run, and watch preview remain strong defaults. | Human dry-run is better, but the summary says `no tests will run` even for custom jobs. | Change wording to `no commands will run`. | Slightly less test-specific language for the common RSpec path. |
| Conceptual coherence | 4 | `--json` now has a direct “not a Plur flag” message; `watch run --help` now says watch rejects one-shot dry-run. | One-shot JSON uses `--dry-run-format=json`, while watch preview uses `--format=json`. | Keep documenting the two preview contexts, or later converge only if the model gets simpler. | Renaming structured-output flags would be a breaking change. |
| Feedback quality | 4 | `./plur --json=tmp/results.json --dry-run` now points to `plur --dry-run --dry-run-format=json` and `plur watch find --format=json <file>`. | Parser/config failures remain prose-only. | Add contract examples for common error modes, or add structured error JSON. | Structured errors expand the stable API surface. |
| Composability | 3 | JSON dry-run and watch-find JSON success/no-op paths are clean, but worker startup failures can stream stderr and later replay captured stderr on stdout via errored-file output. | Stdout/stderr separation is not reliable for errored worker startup paths. | Fix errored-worker output routing and add CI-style stdout/stderr contract tests. | Changing failure output may require golden snapshot updates and careful RSpec/Minitest handling. |
| Config/API cleanliness | 4 | T33/T35 removed the false `--json` API and left two real structured preview APIs. | Watch/run help still exposes some run-only flags, and docs need to catch up to the new dry-run text. | Refresh `docs/output-contracts.md`; review inherited watch-run flags. | Hiding inherited flags can surprise users who rely on global behavior. |

## Reviewer Summary

McClintock:
- Scores: all 4s.
- Direction: trending right.
- Main concerns: first-contact help remains flag-heavy, `spec` remains a
  command and concept, structured failures are still prose-only, and one-shot
  versus watch JSON have different flag names.

Turing:
- Scores: all 4s.
- Direction: T35-T37 landed cleanly.
- Main concerns: `docs/output-contracts.md` is stale after T37, `watch run`
  still exposes inherited run flags, and the dry-run summary should say
  `no commands will run` for custom jobs.

Heisenberg:
- Scores: all 4s except Composability 3.
- Direction: human and JSON preview paths improved, but CI/agent safety has a
  real stdout/stderr risk.
- Main concern: worker startup failures can capture subprocess stderr into
  `WorkerResult.Output`, then result rendering prints errored output with
  `fmt.Print`, replaying stderr text on stdout.

## Evidence

Removed `--json` now has direct guidance:

```text
Error: --json is not a Plur flag.
Use `plur --dry-run --dry-run-format=json [patterns...]` for a structured one-shot plan.
Use `plur watch find --format=json <file>` for a structured watch preview.
```

`plur watch run --help` now matches watch behavior:

```text
--dry-run                  One-shot run preview only; watch mode rejects it
--dry-run-format="text"    One-shot dry-run output format: text or json
```

Human dry-run now gives a skimmable plan before command details:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)
[dry-run] Running 2 specs [rspec] in parallel using 2 workers
[dry-run] Plan: 2 targets across 2 workers; no tests will run
[dry-run] Commands:
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=2 TEST_ENV_NUMBER=1 bin/rspec ...
```

Dry-run JSON remains clean:

```text
exit=0
stderr:
plur version=v0.56.1-0.20260523142447-6adebdd1c145+dirty
stdout:
{
  "version": 1,
  "mode": "spec",
  "job": {
    "name": "rspec",
    "framework": "rspec",
    "reason": "explicit_patterns"
  }
}
```

Watch-find JSON no-op remains structured:

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

T37 verification before commit:

```text
bin/rake
```

`bin/rake` passed with 370 examples, 0 failures, and 4 existing pending
examples.

## Are We Moving In The Right Direction?

Yes for human use: the common path is clearer, dry-run is more skimmable, and
bad `--json` usage no longer sends users toward `--job`. The latest reviewer
feedback also uncovered an automation-specific quality gap, so the next loop
should shift from help polish to output-stream correctness.

## Top Design Problems

1. Worker startup failure output can break stdout/stderr separation by replaying
   captured stderr on stdout for errored workers.
2. `docs/output-contracts.md` dry-run text is stale after T37 and omits the new
   `Plan` and `Commands` lines.
3. The dry-run summary says `no tests will run`, which is accurate for built-in
   test jobs but not for custom/lint jobs.
4. `watch run --help` still exposes inherited flags such as `--workers`,
   `--rspec-split`, and `--first-is-1`.
5. JSON-mode planning/config errors are still plain stderr with empty stdout.

## Recommended Next Changes

1. T39-DEV: fix errored-worker stdout/stderr routing and add contract coverage
   for worker startup failure.
2. T40-DEV: update `docs/output-contracts.md` to include the new dry-run text
   summary and add exit-code/stdout/stderr examples for agents.
3. T41-DEV: change human dry-run summary wording from `no tests will run` to
   `no commands will run`.
4. Later: review inherited `watch run` flags and decide whether to hide or
   reject more of them.

## Things That Should Not Change

- Commandless `plur` should remain the primary everyday entry point.
- Text dry-run should remain human-oriented and explicitly non-contractual for
  scripts.
- Dry-run JSON should keep stdout clean and keep `argv`/`env` as canonical
  machine fields.
- `watch find --format=json` should keep returning structured no-op previews
  with exit code 2.
- The unused `--json` file-output flag should stay removed.

## Done-Done Status

Not done. Human CLI quality is steadily better, but Composability is not yet at
4 across reviewers because worker failure stream routing can pollute stdout.
Continue the DEV loop with output-stream correctness first.
