# Output Contracts

Plur has two kinds of output:

- human text, meant for terminals and debugging
- structured JSON, meant for scripts and agents

Human text can change when the CLI becomes clearer. Use documented JSON output
when a script needs stable fields.

## One-Shot Runs

Normal test runs write Plur status lines to stderr and framework test output to
stdout. Worker stderr is streamed to stderr; it is not replayed on stdout when
a worker exits before producing test events. For example, the run summary
appears on stderr:

```text
Running 13 specs [rspec] in parallel using 4 workers
```

Warnings also use stderr and do not necessarily mean failure. A command can
exit 0 while printing a warning:

```text
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
```

Exit code 0 means all selected work passed or the requested plan/preview was
produced. Non-zero means selected work failed, Plur could not plan or run the
command, or a command-specific condition occurred. Plur uses exit code 1 for
failed work and many planning/runtime errors. Parser validation can return its
own non-zero code before a command runs. Command-specific non-zero codes are
documented below.

## Dry-Run Text

`plur --dry-run` writes a human preview to stderr and does not execute tests.
The text preview is intentionally copyable and skimmable, but it is not the
machine API.

```text
[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
[dry-run] Plan: 1 target across 1 worker; no commands will run
[dry-run] Commands:
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=1 TEST_ENV_NUMBER=1 bin/rspec ...
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

Successful JSON preview:

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
    {
      "index": 0,
      "targets": [
        "spec/integration/spec/help_spec.rb"
      ],
      "argv": [
        "bin/rspec",
        "-r",
        ".../json_rows_formatter.rb",
        "--format",
        "Plur::JsonRowsFormatter",
        "--force-color",
        "spec/integration/spec/help_spec.rb"
      ],
      "env": [
        "PARALLEL_TEST_GROUPS=1",
        "TEST_ENV_NUMBER=1"
      ],
      "shell": "PARALLEL_TEST_GROUPS=1 TEST_ENV_NUMBER=1 bin/rspec ..."
    }
  ]
}
stderr:
plur version=...
```

JSON-mode parser error:

```text
exit=80
stdout:

stderr:
plur: error: --dry-run-format requires --dry-run
```

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
- `env`: environment entries Plur adds or overrides, including configured job
  env; this is the canonical environment field for scripts and intentionally
  excludes unrelated inherited shell env. Each environment key appears at most
  once, and duplicate configured entries keep the final effective value.
- `shell`: quoted, copyable command string for humans.

Do not parse `shell`; use `argv` and `env` when executing from a script.

## Watch Find

`plur watch find <changed-file>` writes a human watch preview to stdout.

```text
[watch] Checking spec/spec_helper.rb
[watch] No matching rule for spec/spec_helper.rb
[watch] Hint: add a [[watch]] mapping for shared files if this change should run tests.
```

For runnable changes, text output also includes the command that live watch
would execute:

```text
[watch] Checking lib/calculator.rb
[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)
[watch] Would run job rspec with spec/calculator_spec.rb
[watch] Command: bundle exec rspec spec/calculator_spec.rb
```

Exit code 0 means at least one existing target would run. Exit code 2 means no
runnable target exists for that changed file, including when no watch mappings
are configured. Exit code 1 is reserved for errors such as invalid
configuration.

Use JSON when a script or agent needs a stable watch preview:

```bash
plur watch find --format=json spec/spec_helper.rb
```

Runnable changes include the matched rule, existing target, and final command
plan:

```text
exit=0
stdout:
{
  "version": 1,
  "mode": "watch_find",
  "file": "lib/calculator.rb",
  "matched_rules": [
    {
      "name": "lib-to-spec",
      "source": "lib/**/*.rb",
      "jobs": ["rspec"],
      "target": "spec/{{match}}_spec.rb"
    }
  ],
  "existing_targets": {
    "rspec": ["spec/calculator_spec.rb"]
  },
  "missing_targets": {},
  "job_plans": [
    {
      "job": "rspec",
      "targets": ["spec/calculator_spec.rb"],
      "argv": ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
      "env": [],
      "cwd": "/path/to/project",
      "shell": "bundle exec rspec spec/calculator_spec.rb"
    }
  ],
  "exit_code": 0
}
stderr:
```

Command and configuration errors in JSON modes still write plain text to stderr
and may leave stdout empty.

No matching watch target is still structured output. In JSON mode, no
configured watch mappings use the same empty JSON shape and exit code 2:

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
  "job_plans": [],
  "exit_code": 2
}
stderr:
```

Stable top-level keys:

- `version`: output contract version
- `mode`: `watch_find`
- `file`: changed file path
- `matched_rules`: watch rules that matched the file
- `existing_targets`: targets that exist, grouped by job
- `missing_targets`: targets that do not exist, grouped by job
- `job_plans`: runnable command plans, one per job, each with `job`,
  `targets`, `argv`, `env`, `cwd`, and `shell`
- `exit_code`: the exit code Plur will use for this preview

`job_plans[].argv` and `job_plans[].env` are the canonical command fields for
scripts. `job_plans[].shell` is a quoted, copyable string for humans; do not
parse it.

Human `watch find` text remains terminal-oriented and is not the machine API.

## Debug Output

`--debug` and `--verbose` are diagnostic modes. Their exact lines, fields, and
ordering are unstable and should not be parsed by scripts.
