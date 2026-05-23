# T30 Score Card

Source review: T27-T29 after `docs/goal/t26_score_card.md`.

Recent changes:
- T27 stopped presenting bare `plur test` as a command and made
  `plur test --help` explain that `test` is a target path.
- T28 changed command/runtime errors from timestamped log lines to plain
  `Error: ...` stderr.
- T29 quoted dry-run shell strings and clarified that dry-run JSON `argv` and
  `env` are canonical script fields.

## Local Score

| Criteria | Score | Notes |
| --- | ---: | --- |
| Obviousness | 4 | Top-level help now shows real target examples, including `plur test/calculator_test.rb`. Bare `plur test --dry-run` can still fall through to `Error: file not found: test` in repos without a `test/` directory. |
| Brevity / Surface Area | 4 | The first screen is clearer than before, but global flags still dominate top-level and watch help. |
| Default Quality | 4 | Defaults are useful: commandless run, RSpec-first detection, explicit target inference, watch previews, dry-run plans. Human dry-run for large suites is still a verbose worker-command view. |
| Conceptual Coherence | 4 | `plur test` no longer looks like a command, parser/help behavior is closer, and dry-run JSON has clearer field roles. `spec` remains both a command name and the generic run path. |
| Feedback Quality | 4 | Plain `Error: ...` is a major improvement. Parser errors still use `plur: error: ...`, and bare `plur test --dry-run` still gets a generic file-not-found message. |
| Composability | 4 | JSON stdout remains clean, `argv`/`env` are canonical, and `shell` is now quoted. JSON-mode error paths are still human stderr only. |
| Config/API Cleanliness | 4 | The CLI/config boundaries are cleaner, but output-contract exit-code wording and configuration docs still need tightening. |

## External Reviewer Summary

CLI reviewer:
- Scores: Obviousness 4, Brevity 4, Default 4, Conceptual 4, Feedback 4,
  Composability 5, Config/API 4.
- Direction: trending right.
- Main concerns: bare `plur test --dry-run` still has generic target feedback,
  human dry-run is verbose, error styles are not fully unified, and top-level
  help remains flag-heavy.

Docs/API reviewer:
- Scores: Obviousness 5, Brevity 4, Default 4, Conceptual 4, Feedback 5,
  Composability 4, Config/API 4.
- Direction: docs-focused DEV should remain paused.
- Main concerns: `docs/output-contracts.md` should clarify JSON error paths,
  exit-code meaning, and that `shell` is copy/paste only; `docs/configuration.md`
  remains broad.

Automation reviewer:
- Scores: all 4s.
- Direction: stronger but not a 5.
- Main concerns: `docs/output-contracts.md` overstates exit code 1 as selected
  work failure when command/config errors also exit 1; JSON-mode command errors
  are stderr-only; `watch find` can select a job before checking empty watches;
  `shell` remains present in structured JSON.

## Evidence

Top-level help now shows explicit Minitest target paths:

```text
Common workflows:
  plur                                Run the detected test suite
  plur spec/calculator_spec.rb        Run one target
  plur test/calculator_test.rb        Run one Minitest target
```

`plur test --help` no longer renders `plur spec` help:

```text
exit=1
Error: `test` is a target path, not a Plur command.
Use `plur test/calculator_test.rb` to run a Minitest target.
Use `plur --help` to list Plur commands.
```

Command errors are plain:

```text
exit=1
STDOUT:
STDERR:
Error: failed to select watch job: job 'does-not-exist' not found. Available jobs: build, go-test, minitest, plur, rails, rake, rspec
```

Human dry-run shell strings quote unsafe args:

```text
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=2 TEST_ENV_NUMBER=1 bundle exec rspec ... --tag '~type:system' spec/models/system_spec.rb
```

Dry-run JSON keeps structured execution details on stdout and the version on
stderr:

```text
STDOUT first lines:
{
  "version": 1,
  "mode": "spec",
  "workers": [
    {
      "argv": ["bundle", "exec", "rspec", "..."],
      "env": ["PARALLEL_TEST_GROUPS=1"],
      "shell": "PARALLEL_TEST_GROUPS=1 bundle exec rspec ..."
    }
  ]
}
STDERR:
plur version=v0.56.1-0.20260523134522-d8193caf1b0d+dirty
```

T29 verification before commit:

```text
bin/rake
```

`bin/rake` passed with 363 examples, 0 failures, and 4 existing pending
examples.

## Are We Moving In The Right Direction?

Yes. T27-T29 improved the places where the interface made a false promise:
`plur test` looked command-like, expected errors looked like internal logs, and
`shell` looked more executable than it actually was. These changes are
coherent with the existing model and do not add new CLI concepts.

## Top Design Problems

1. Bare `plur test --dry-run` still gives `Error: file not found: test` when
   no `test/` directory exists. The command now explains `plur test --help`,
   but the likely user mistake still needs better feedback.
2. `docs/output-contracts.md` needs to distinguish selected-work failures from
   command/config errors, especially for JSON modes.
3. JSON-mode command errors are still plain stderr only. This may be fine, but
   the contract needs to say it explicitly or implement structured error JSON.
4. Human dry-run output remains worker-command-heavy for first contact.
5. Help remains flag-heavy at the top level and under `watch`.

## Recommended Next Changes

1. T31-DEV: improve bare `test` target feedback when the path does not exist.
   The error should connect the target-path concept to the missing directory.
2. T32-DEV: tighten output-contract docs around exit code 1, JSON-mode error
   paths, and the non-parseable nature of `workers[].shell`.
3. T33-DEV: reduce top-level/watch help flag noise, or make human dry-run less
   implementation-heavy while keeping copyable detail available.
4. Later: slim `docs/configuration.md` into a compact reference.

## Done-Done Status

Not done. The design is materially better and most criteria are stable 4s, with
some reviewer 5s, but remaining issues are still visible at everyday CLI and
automation boundaries. Continue the DEV loop.
