# Output Contracts

Plur has two kinds of output:

- human text, meant for terminals and debugging
- structured JSON, meant for scripts and agents

Human text can change when the CLI becomes clearer. Use documented JSON output
when a script needs stable fields.

## One-Shot Runs

Normal test runs write Plur status lines to stderr and framework test output to
stdout. For example, the run summary appears on stderr:

```text
Running 13 specs [rspec] in parallel using 4 workers
```

Warnings also use stderr and do not necessarily mean failure. A command can
exit 0 while printing a warning:

```text
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
```

Exit code 0 means all selected work passed or the requested plan/preview was
produced. Exit code 1 can mean selected work failed or Plur could not plan/run
the command because of user input, configuration, or environment.
Command-specific non-zero codes are documented below.

## Dry-Run Text

`plur --dry-run` writes a human preview to stderr and does not execute tests.
The text preview is intentionally copyable, but it is not the machine API.

```text
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=1 TEST_ENV_NUMBER=1 bundle exec rspec ...
```

## Dry-Run JSON

Use JSON when a script or agent needs a stable one-shot plan:

```bash
plur --dry-run --dry-run-format=json spec/calculator_spec.rb
```

The JSON plan is written to stdout. Human status, version, and warnings remain
on stderr. Structured JSON is emitted only after Plur successfully builds the
plan. Command and configuration errors in JSON modes still write plain text to
stderr and may leave stdout empty.

Stable top-level keys:

- `version`: output contract version
- `mode`: command mode, currently `spec`
- `job`: selected job details
- `targets`: selected runnable targets
- `warnings`: non-fatal warnings that also appear on stderr
- `workers`: worker command plan

Worker entries include:

- `index`: worker index
- `targets`: targets assigned to the worker
- `argv`: command argv; this is the canonical command field for scripts
- `env`: Plur-managed environment entries; this is the canonical environment
  field for scripts
- `shell`: quoted, copyable command string for humans.

Do not parse `shell`; use `argv` and `env` when executing from a script.

## Watch Find

`plur watch find <changed-file>` writes a human watch preview to stdout.

```text
[watch] Checking spec/spec_helper.rb
[watch] No matching rule for spec/spec_helper.rb
```

Exit code 0 means at least one existing target would run. Exit code 2 means no
runnable target exists for that changed file. Exit code 1 is reserved for
errors such as invalid configuration.

Use JSON when a script or agent needs a stable watch preview:

```bash
plur watch find --format=json spec/spec_helper.rb
```

Command and configuration errors in JSON modes still write plain text to stderr
and may leave stdout empty.

Stable top-level keys:

- `version`: output contract version
- `mode`: `watch_find`
- `file`: changed file path
- `matched_rules`: watch rules that matched the file
- `existing_targets`: targets that exist, grouped by job
- `missing_targets`: targets that do not exist, grouped by job
- `exit_code`: the exit code Plur will use for this preview

Human `watch find` text remains terminal-oriented and is not the machine API.

## Debug Output

`--debug` and `--verbose` are diagnostic modes. Their exact lines, fields, and
ordering are unstable and should not be parsed by scripts.
