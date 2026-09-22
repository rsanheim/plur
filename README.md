# plur

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/rsanheim/plur/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/rsanheim/plur/tree/main)

Plur runs RSpec and Minitest tests in parallel and reruns them when files change.
It is a single Go binary you can install once and use across your Ruby and Rails projects.

## Installation

### Homebrew (macOS)

```bash
brew install rsanheim/tap/plur
```

### Shell script (macOS / Linux)

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | sh
```

Installs the latest release to `~/.local/bin` by default. If that directory doesn't exist, it uses `/usr/local/bin` when that's present and writable, otherwise it creates `~/.local/bin`. Pin a version or change the destination (`PLUR_INSTALL_PATH`) with environment variables:

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_VERSION=v0.60.0 sh
```

### Manual binary download

Download the latest release for your platform from [GitHub Releases](https://github.com/rsanheim/plur/releases), extract, and put the `plur` binary somewhere on your PATH.

See [Getting Started](docs/getting-started.md) for first-run details.

## Quick Start

```bash
cd my-rails-project
plur -n 4 --dry-run # Preview what would run
plur -n 4           # Run tests with four workers
plur                # Run tests with the default four workers
plur watch          # Rerun tests when files change
```

## Supported Platforms

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* _Experimental_ Windows x86_64

Watch mode (`plur watch`) installs the bundled watcher binary on first use.

## Key Features

### Parallel Test Execution

```bash
plur -n 4                    # Run with specific worker count
plur                        # Run with the default 4 workers
plur --dry-run               # Preview execution plan
```

### Rake (and Rails) Tasks

```bash
plur rails db:test:prepare      # Prepare test DBs for your configured worker count
plur rails db:test:prepare -n 8 # Prepare test DBs for eight test databases
plur rails db:drop db:create RAILS_ENV=test   # Run drop and create n times for our test env
plur rails app:my_task  # Run an app Rake task n times
plur rake app:my_task   # Run the same task with bundle exec rake
plur rake app:my_task -n 1 -- --option1      # Pass Rake-specific flags after --
```

`plur rails <args>` and `plur rake <args>` run the task once per worker and set `PARALLEL_TEST_GROUPS` and `TEST_ENV_NUMBER`. Plur does not set or alter `RAILS_ENV`.

Arguments are appended literally; put Plur flags like `-n` before `--`, and use `--` to pass flags through to Rails/Rake.

### Explicit Framework Selection

Use `--use` to choose a framework in projects with both RSpec and Minitest tests.

```bash
plur --use=rspec             # Run RSpec tests explicitly
plur --use=minitest          # Run Minitest tests
```

Otherwise, Plur detects the framework automatically.

### Configuration

Save project settings in `.plur.toml` or shared defaults in `~/.plur.toml`:

```toml
# .plur.toml or ~/.plur.toml
workers = 4

[job.rspec]
cmd = ["bin/rspec"]

[[watch]]
name = "lib-to-spec"
source = "lib/**/*.rb"
targets = ["spec/{{match}}_spec.rb"]
jobs = ["rspec"]
```

Config files load in this order (later files override earlier values):

1) `~/.plur.toml`
2) `.plur.toml`
3) `PLUR_CONFIG_FILE` (if set)

See `docs/examples/` for more configuration examples.

### Environment Variables

* `PLUR_WORKERS`: Number of workers
* `TEST_ENV_NUMBER`: Workers receive `"1"`, `"2"`, etc. Use `--no-first-is1` to give the first worker `""` instead
* `PARALLEL_TEST_GROUPS`: Total number of workers
* `PARALLEL_TEST_PROCESSORS`: Legacy fallback for `PLUR_WORKERS` (parallel_tests compatibility)

See the [documentation](docs/index.md) for more.
