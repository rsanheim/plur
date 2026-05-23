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
