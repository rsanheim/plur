# T42 Score Card - After Stream And Output Contract Cleanup

## Context

This reflection covers:
- T39: stopped worker subprocess stderr from replaying on stdout for errored
  workers.
- T40: refreshed `docs/output-contracts.md` with current stdout/stderr and JSON
  examples.
- T41: changed human dry-run wording from `no tests will run` to
  `no commands will run`.

Inputs:
- Baseline: `docs/goal/current_design.md`.
- Previous reflection: `docs/goal/t38_score_card.md`.
- Latest design notes: `docs/goal/new_design.md`.
- Current executable checks.
- Reviewer feedback from Carson, Popper, and Meitner.

## Scorecard

| Category | Score | Evidence | Main Issue | Suggested Improvement | Risk / Tradeoff |
| --- | ---: | --- | --- | --- | --- |
| Obviousness | 4 | `plur --help` leads with commandless usage and common workflows. Dry-run now says selected job, run count, plan shape, and that no commands will run. | `spec` remains both an explicit command and the implicit default path. | Keep examples path-first and avoid adding more command names. | Hiding `spec` too much could make command-specific help harder to discover. |
| Brevity / surface area | 4 | The false `--json` flag is gone, `watch find` help is focused, and dry-run text is compact before worker details. | Top-level and `watch run` help still expose advanced/inherited flags early. | Hide or regroup advanced run-only flags in watch help. | More custom help code can drift from Kong's generated output. |
| Default quality | 4 | RSpec-first detection, `-C`, dry-run, and watch preview remain strong defaults. T41 wording is job-neutral for custom jobs. | Missing executable startup failures still render a Plur runtime error on stdout. | Route Plur worker startup/runtime errors to stderr. | Moving error text can affect existing snapshots or users who captured stdout only. |
| Conceptual coherence | 3 | Successful dry-run JSON and watch-find JSON have clear documented shapes. | Some JSON-mode edges break the model: no-watch-mapping `watch find --format=json` prints human text, and dry-run JSON omits configured job env. | Make JSON modes consistently JSON on stdout when requested, and make dry-run JSON describe the actual command environment. | Expanding the JSON contract requires careful docs and tests. |
| Feedback quality | 3 | Removed `--json` guidance is direct; output contracts now show JSON-mode parser errors and watch no-op exit 2. | Startup errors go to stdout, parser/config errors are still plain stderr with mixed exit codes, and some JSON-mode edge errors are not contract-tested. | Add focused stderr/stdout and JSON-mode error contract tests. | Structured error JSON would add stable API surface. |
| Composability | 3 | T39 fixed the main subprocess stderr replay problem; successful dry-run JSON keeps clean stdout and `watch find` no-op JSON exits 2. | `cmd.Start()` errors still pollute stdout; dry-run JSON `workers[].env` is incomplete for configured job env; one watch JSON edge emits prose. | Fix the three contract gaps before more UI polish. | Tightening contracts may reveal more legacy output assumptions. |
| Config/API cleanliness | 3 | `--json` stayed removed, `argv`/`env` remain the intended machine fields, and worker stderr streaming is now documented. | `workers[].env` does not currently include `job.Env`, and global run flags still bleed into watch surfaces. | Include configured job env in dry-run JSON and then clean up inherited watch-run flags. | Showing env can expose values users consider sensitive; docs should frame it as the executable plan. |

## Reviewer Summary

Carson, first-contact CLI review:
- Scores: all 4s.
- Direction: T39-T41 landed cleanly.
- Main concern: advanced/global flags still appear early, especially in watch
  help.

Meitner, maintainer/API review:
- Scores: all 4s except Config/API cleanliness 3.
- Direction: no blocking API regression.
- Main concern: global run flags bleed into `watch run`; only `watch find` has
  focused hidden/no-op flag handling.

Popper, automation/CI review:
- Scores: Obviousness 4, Brevity 4, Default quality 4, Conceptual coherence 3,
  Feedback quality 3, Composability 3, Config/API cleanliness 3.
- Direction: T39 fixed the reviewed subprocess stderr bug, but deeper contract
  gaps remain.
- Main concerns: startup errors on stdout, incomplete dry-run JSON env, and
  human text from `watch find --format=json` when no watch mappings exist.

## Evidence

Human dry-run is now job-neutral:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)
[dry-run] Running 2 specs [rspec] in parallel using 2 workers
[dry-run] Plan: 2 targets across 2 workers; no commands will run
[dry-run] Commands:
```

The T39 stream fix works for subprocess stderr:

```text
status=1
stdout_has_marker=false
stderr_has_marker=true
stdout_summary=0 examples, 0 failures
```

But missing executables still put Plur runtime errors on stdout:

```text
status=1
stdout_has_start_error=true
stderr_has_start_error=false
stdout:
Finished in 0.00021 seconds (files took 0 seconds to load)
0 examples, 0 failures
Error: failed to start command: exec: "definitely-not-a-real-plur-command": executable file not found in $PATH
```

Dry-run JSON omits configured job env:

```text
status=0
env=["PARALLEL_TEST_GROUPS=1", "TEST_ENV_NUMBER=1"]
has_custom_token=false
```

`watch find --format=json` with no watch mappings can emit human text:

```text
status=0
stdout_first=No watch mappings configured.
stdout_is_json=false
stderr=""
```

T39-T41 verification before reflection:

```text
bin/rake
```

`bin/rake` passed with 371 examples, 0 failures, and 4 existing pending
examples after T41.

T42 validation before commit:

```text
script/check-links
bin/rake
```

`script/check-links` passed. `bin/rake` passed with 371 examples, 0 failures,
and 4 existing pending examples.

## Are We Moving In The Right Direction?

Yes, but this reflection changes the next priority. The human-facing CLI is
steadily clearer, and the three T38 recommendations were addressed. However,
the latest automation review found contract gaps that matter for scripts and
agents. The next loop should stay on output/API correctness before returning to
help-surface polish.

## Top Design Problems

1. Worker startup errors from `cmd.Start()` still render on stdout through
   errored-worker result printing.
2. Dry-run JSON `workers[].env` does not include configured `job.Env`, so the
   documented `argv`/`env` execution plan is incomplete.
3. `watch find --format=json` can print human prose when no watch mappings are
   configured.
4. `watch run --help` still exposes inherited run-only flags and advanced
   globals.
5. JSON-mode planning/config failures remain plain stderr with empty stdout and
   mixed exit codes.

## Recommended Next Changes

1. T43-DEV: route Plur worker startup/runtime errors to stderr and add a
   missing-executable stdout/stderr contract spec.
2. T44-DEV: make dry-run JSON `workers[].env` include configured job env, then
   document any intentional inherited-env exclusions.
3. T45-DEV: make `watch find --format=json` return JSON for the no-watch-mapping
   path, likely with exit code 2.
4. T46-DEV: clean up `watch run` inherited flag help/rejection after the
   agent-facing contracts are tighter.

## Things That Should Not Change

- Commandless `plur` should remain the primary everyday entry point.
- Text dry-run should remain human-oriented and explicitly non-contractual for
  scripts.
- Dry-run JSON should keep `argv` and `env` as canonical machine fields.
- `watch find --format=json` should keep structured no-op previews with exit
  code 2.
- The removed `--json` file-output flag should stay removed.

## Done-Done Status

Not done. Most human-facing categories are stable at 4, but automation and API
contracts are still at 3. Continue the DEV loop with stream routing and JSON
contract correctness before more visible help polish.
