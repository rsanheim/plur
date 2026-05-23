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

Exit code 0 means all selected work passed. Exit code 1 means selected work ran
and failed. Other non-zero exit codes are command or configuration errors.

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
on stderr.

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
- `argv`: command argv
- `env`: Plur-managed environment entries
- `shell`: copyable command string

## Watch Find

`plur watch find <changed-file>` writes a human watch preview to stdout.

```text
[watch] Checking spec/spec_helper.rb
[watch] No matching rule for spec/spec_helper.rb
```

Exit code 0 means at least one existing target would run. Exit code 2 means no
runnable target exists for that changed file. Exit code 1 is reserved for
errors such as invalid configuration.

`watch find` human text is not yet a machine API. Prefer the dry-run JSON plan
for one-shot automation until watch gets structured output.

## Debug Output

`--debug` and `--verbose` are diagnostic modes. Their exact lines, fields, and
ordering are unstable and should not be parsed by scripts.
