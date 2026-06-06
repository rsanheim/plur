# Usage

This page covers common command workflows. For the full configuration surface,
see [Configuration](configuration.md). For stable machine output, streams, and
exit codes, see [Output Contracts](output-contracts.md).

## Run Tests

```bash
# Run the detected test suite with the default worker count
plur

# Run one target
plur spec/models/user_spec.rb

# Run targets matching a shell-safe glob
plur 'spec/models/**/*_spec.rb'

# Set the worker count
plur -n 4
plur --workers 8

# Run from another directory
plur -C path/to/project
```

`plur spec` is the explicit form of the default test runner:

```bash
plur spec
plur spec spec/models/user_spec.rb
```

## Preview A Run

Use dry-run before changing filters, worker counts, or custom jobs:

```bash
plur --dry-run
plur --dry-run spec/models/user_spec.rb
```

The text preview is for people. Scripts and agents should use JSON:

```bash
plur --dry-run --dry-run-format=json spec/models/user_spec.rb
```

See [Output Contracts](output-contracts.md#dry-run-json) for the stable JSON
shape.

## Select A Job

Plur auto-detects common Ruby test layouts. A project with `spec/` uses RSpec,
and a project with `test/` uses Minitest. If both directories exist, Plur
defaults to RSpec.

```bash
plur --use=rspec
plur --use=minitest
```

Set a project default in `.plur.toml`:

```toml
use = "minitest"
```

See [Configuration](configuration.md#job-selection-priority) for job selection
details.

## Exclude Targets

Use `--exclude-pattern` to drop files from the discovered test plan. Patterns
use doublestar semantics.

```bash
# Skip one file
plur --exclude-pattern 'spec/legacy/old_spec.rb'

# Skip a directory of specs
plur --exclude-pattern 'spec/system/**/*_spec.rb'

# Combine exclusions
plur --exclude-pattern 'spec/system/**/*_spec.rb' \
     --exclude-pattern 'spec/legacy/**/*_spec.rb'
```

CLI excludes are additive on top of configured job excludes. See
[Configuration](configuration.md#exclude-patterns).

## Watch Files

```bash
# Watch files and run matching tests
plur watch

# Preview what a changed file would run
plur watch find spec/models/user_spec.rb

# Use JSON for scripts
plur watch find --format=json spec/models/user_spec.rb

# Customize debounce, timeout, and ignored paths
plur watch --debounce 250 --timeout 60 --ignore "vendor/**" --ignore "tmp/**"
```

See [Watch Mode](features/watch-mode.md) for file mapping and troubleshooting.

## Diagnose Problems

```bash
plur doctor
plur --debug
```

`plur doctor` checks installation and environment details. `--debug` is for
interactive troubleshooting; debug output is not stable for scripts.

## Rails And Rake

```bash
plur rails db:prepare -n 4
plur rails db:migrate VERSION=20260429000000 -n 4
plur rails db:migrate -n 4 -- --trace
plur rake db:setup -n 4
plur rake db:create db:migrate -n 4
plur rake -n 1 -- --tasks
```

Rails and Rake commands run once per worker with `PARALLEL_TEST_GROUPS` and
`TEST_ENV_NUMBER` set. Arguments are appended literally; they are not treated
as test file patterns. Put Plur flags like `-n` before `--`; arguments after
`--` pass through to Rails or Rake.

## Tune Workers

Plur uses 4 workers by default and respects `PARALLEL_TEST_PROCESSORS`.

```bash
plur -n 4
PARALLEL_TEST_PROCESSORS=8 plur
```

Runtime data is collected automatically so future full runs can balance files
across workers. See [Parallel Execution](features/parallel-execution.md) for
the runtime cache and advanced RSpec split behavior.

## Next Steps

- [Configuration](configuration.md) for `.plur.toml`
- [Output Contracts](output-contracts.md) for stable script and agent output
- [Watch Mode](features/watch-mode.md) for file-change workflows
- [Parallel Execution](features/parallel-execution.md) for balancing behavior
