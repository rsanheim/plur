## Plur - new design specs, plans, ideas

_for TX-DEV phases as part of `cli_goal.md`_

## T4-DEV - Explain Dry-Run Job Selection

Pain point: `plur --dry-run` shows exact worker commands, but it does not say
which job was selected or why. Users have to infer the selection from the
framework label and command argv. This is especially confusing in mixed
RSpec/Minitest projects and for custom jobs where the job name is not visible.

Change: add one compact dry-run line after file discovery and before worker
commands:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
```

The line should use the existing `runtime.SelectedJob` data rather than adding
a new selection abstraction. It should appear only in dry-run mode, because
normal test output should stay focused on execution and failures.

Acceptance criteria:
- `plur --dry-run` shows selected job, framework, and reason.
- Passing an explicit target such as `test/` shows the explicit-pattern reason.
- Passing `--use custom-job` shows the configured job name and explicit-name
  reason.
- Existing worker dry-run output remains copyable.
- Focused integration tests and Ruby lint pass.

Before evidence:
- T1 transcript `tmp/cli_inventory_demo.txt` showed dry-run jumping from
  `plur version=...` directly to `[dry-run] Running ...`, so job selection had
  to be inferred from the worker command.

After evidence:

```text
plur version=v0.56.1-...
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
```

```text
plur version=v0.56.1-...
[dry-run] Selected job: minitest (framework: minitest, reason: explicit patterns)
[dry-run] Running 1 test [minitest] in parallel using 1 worker
```

Tradeoff: dry-run gains one line of output. That is deliberate for human
clarity, and the worker command lines remain unchanged and copyable.

## T5-DEV - Warn When CLI Excludes Match Nothing

Pain point: `--exclude-pattern` is powerful but easy to mistype. A plausible
pattern such as `*user*/_spec.rb` can match no selected files while the command
still succeeds, so the user thinks a file was excluded when it was not.

Change: when an explicit CLI `--exclude-pattern` matches none of the files in
the selected test plan, print a compact warning before the run or dry-run
worker plan:

```text
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
```

The warning is limited to CLI-provided excludes. Configured job excludes may be
broad defaults that naturally do not match a focused target, so warning on those
would make normal focused runs noisy.

Acceptance criteria:
- A dry-run with a non-matching CLI `--exclude-pattern` exits successfully and
  prints a warning.
- A matching CLI `--exclude-pattern` does not warn.
- Exclude behavior itself is unchanged: matching excludes still remove files,
  and excluding every candidate still returns the existing error.
- Focused integration tests, Go tests for file discovery details, Ruby lint,
  and the full build pass.

Before evidence:
- T1 transcript `tmp/cli_inventory_demo.txt` showed
  `plur spec --exclude-pattern '*user*/_spec.rb'` keeping
  `spec/models/user_spec.rb` with no warning.

After evidence:

```text
plur version=v0.56.1-...
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 3 specs [rspec] in parallel using 3 workers
```

The refreshed transcript is available at `tmp/cli_inventory_demo.txt`.

Tradeoff: a successful command can now print a warning to stderr. That is
intentional for explicit CLI intent; it is not applied to config excludes.

## T6-DEV - Reject Dry-Run Watch With Guidance

Pain point: `plur --dry-run watch` currently starts watch mode. That is
surprising because dry-run means "show what would happen without doing it" for
normal test runs, while watch mode attaches a watcher and waits for input.

Change: when global `--dry-run` is combined with `watch run` or the default
`watch` command, exit quickly with a focused message:

```text
Error: plur watch does not support --dry-run yet.
Use `plur watch find <changed-file>` to preview which tests a file change would run.
Use `plur --dry-run [patterns...]` to preview a one-shot test run.
```

Acceptance criteria:
- `plur --dry-run watch` exits non-zero without installing or starting the
  watcher.
- The message points to `plur watch find <changed-file>` for watch previews.
- Normal `plur watch --timeout 1` behavior is unchanged.
- Focused watch dry-run tests and the full build pass.

Before evidence:
- `plur --dry-run watch --timeout 1` printed `ready and watching ...` and
  exited 0 after the timeout, so it did real watch setup despite the dry-run
  flag.

After evidence:

```text
Error: plur watch does not support --dry-run yet.
Use `plur watch find <changed-file>` to preview which tests a file change would run.
Use `plur --dry-run [patterns...]` to preview a one-shot test run.
```

Tradeoff: this does not implement a full watch plan. It chooses honest
guidance over a misleading partial dry-run.

## T8-DEV - Make Help And Watch Docs Match The Real Happy Paths

Pain point: `plur --help` still says `Usage: plur <command> [flags]`, even
though commandless `plur [patterns...]` is the daily run command. `plur watch
--help` likewise presents `watch <command>` first, and public watch docs still
recommend `plur watch --dry-run` even though T6 now rejects it.

Change: customize help output just enough to lead with real workflows:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur
  plur spec/calculator_spec.rb
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
```

For watch help, lead with `plur watch [flags]`, `plur watch find
<changed-file>`, and only then the subcommand form. Update
`docs/features/watch-mode.md` to remove `watch --dry-run` and point users to
`watch find`.

Acceptance criteria:
- `plur --help` shows commandless usage first and includes common workflow
  examples.
- `plur watch --help` shows `plur watch [flags]` and `plur watch find
  <changed-file>` before subcommand details.
- Watch docs no longer recommend `plur watch --dry-run`.
- Focused help/doc tests and the full build pass.

Before evidence:
- `plur --help` printed `Usage: plur <command> [flags]`.
- `plur watch --help` printed `Usage: plur watch <command> [flags]`.
- `docs/features/watch-mode.md` recommended `plur watch --dry-run`.

After evidence:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur
  plur spec/calculator_spec.rb
  plur test
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
```

```text
Usage: plur watch [flags]
       plur watch find <changed-file> [flags]
       plur watch <command> [flags]
```

`docs/features/watch-mode.md` now uses `plur watch find` for preview examples.

Tradeoff: help gets a thin custom layer over Kong output. The detailed flags
and command lists still come from Kong so drift stays limited.

## T9-DEV - Surface Live Watch No-Op Feedback

Pain point: watch mode can observe a file change and then appear to do nothing.
For example, `spec/spec_helper.rb` is under a watched directory, but the
built-in spec rule does not map it to a runnable target. Today that is only
visible in debug logs or via a separate `watch find` command.

Change: after a debounced live watch batch, print concise normal-output
messages when a changed path has no runnable result:

```text
[watch] No matching rule for spec/spec_helper.rb
[watch] No existing targets for lib/missing.rb (missing: spec/missing_spec.rb)
```

The event handler should return no-op details rather than printing directly,
so watch mode remains responsible for user-facing output and tests can inspect
the planning result.

Acceptance criteria:
- A live watched change with no matching rule prints a normal-output no-op
  message.
- A matched watch rule with only missing targets prints the changed path and
  missing target names.
- Changes that run jobs do not print no-op messages.
- Focused watch tests and the full build pass.

Before evidence:
- `plur watch find spec/spec_helper.rb` reported `found rules count=0`, but
  live watch did not give the user a normal-output explanation for that no-op.

After evidence:

```text
[watch] No matching rule for spec/spec_helper.rb
```

The focused integration check is
`PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_integration_spec.rb:75`.

Tradeoff: watch mode may print more messages during noisy save bursts. Keep the
messages per debounced batch, concise, and only for paths that produce no run.

## T10-DEV - Warn When Explicit Targets Do Not Match The Selected Job

Pain point: Plur intentionally passes explicit existing files through to the
selected framework, but that can make a likely mistake look valid. For example,
`plur --dry-run lib/calculator.rb` in an RSpec project selects the `rspec` job
and plans to run `bundle exec rspec lib/calculator.rb`, even though the normal
target pattern is `spec/**/*_spec.rb`.

Change: after discovery succeeds, warn when a CLI positional argument is an
explicit existing file target that does not match the selected job's target
pattern:

```text
[warn] target 'lib/calculator.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'
```

The warning should not reject the command. RSpec and other jobs may accept
non-standard explicit files, so the safest UX improvement is visibility rather
than a hard failure.

Acceptance criteria:
- A dry-run with an explicit non-matching file exits successfully and prints a
  warning.
- A matching explicit spec file does not warn.
- Glob inputs and directory inputs do not warn per individual expanded file.
- Focused integration tests, file-set unit tests, and the full build pass.

Before evidence:
- Earlier inventory and review phases showed explicit non-test files becoming
  RSpec targets with no hint that they were outside the selected job's normal
  target pattern.

After evidence:

```text
[warn] target 'lib/calculator.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect after patterns)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
```

Tradeoff: the command remains permissive. This preserves advanced framework
behavior while making likely mistakes visible.

## T11-DEV - Humanize Watch Find Output

Pain point: T8 and T6 point users to `plur watch find <changed-file>` as the
safe watch preview command, but its output is still logger-shaped:

```text
level=INFO msg="found rules" name=lib-to-spec source=lib/**/*.rb
```

That is useful to scripts, but weaker for the human workflow now promoted by
help and docs. It also does not match T9's live watch no-op wording.

Change: print plain watch-preview lines from `watch find`:

```text
[watch] Checking lib/example.rb
[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)
[watch] Would run job rspec with spec/example_spec.rb
```

For no-op previews, reuse the live-watch language:

```text
[watch] No matching rule for spec/spec_helper.rb
[watch] No existing targets for lib/example/runner.rb (missing: spec/example/runner_spec.rb)
```

Acceptance criteria:
- `watch find` prints human-readable matched-rule and runnable-target lines.
- A no-rule result exits 2 and prints `No matching rule`.
- A matched rule with missing targets exits 2 and prints `No existing targets`.
- Existing watch-find exit codes remain unchanged.
- Focused watch-find specs and the full build pass.

Before evidence:
- T7 scorecard and Curie's review both identified `found rules count=0` and
  other logger-shaped records as a remaining feedback-quality gap.

After evidence:

```text
[watch] No matching rule for spec/spec_helper.rb
```

Tradeoff: `watch find` becomes more human-first. A structured plan format is
still a separate phase; this phase should not pretend the human text is a
stable machine API.
