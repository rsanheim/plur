# plur

[![CircleCI](https://dl.circleci.com/status-badge/img/gh/rsanheim/plur/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/rsanheim/plur/tree/main)

`plur` is a fast, parallel, drop-in test runner and watcher primarily targeting Ruby and Rails using RSpec or Minitest. Its written in Go, so just install once and use across all your projects.

## Quick Start

```
brew install rsanheim/tap/plur
cd my-rails-project
plur -n 4 --dry-run # preview what would run (no actual test execution)
plur -n 4           # run tests across four cores
plur                # run tests with auto-detected workers (cores - 2)
plur watch          # watch for changes and run tests automatically
```

## Supported Platforms

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* _Experimental_ Windows x86_64

Watch mode (`plur watch`) will install platform-specific binaries on first use. 

## Key Features

### Parallel Test Execution
```bash
plur -n 4                    # Run with specific worker count
plur                          # Auto-detect workers (cores-2)
plur --dry-run               # Preview execution plan
```

### Database Management
```bash
plur db:create -n 3          # Create test databases in parallel
plur db:migrate -n 3         # Run migrations across all test DBs
plur db:setup -n 3           # Full database setup
```

### Explicit Framework Selection

For projects where you have both rspec and minitest tests, you can explicitly select the framework you want to use.

```bash
plur --use=rspec             # Run RSpec tests explicitly
plur --use=minitest          # Run Minitest tests
```

If there is just one framework, omit the `--use` flag and plur will auto-detect the framework.

### Configuration

Plur supports TOML configuration files for persistent settings:
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

See `docs/examples/` directory for more configuration examples.

### Environment Variables
* `TEST_ENV_NUMBER`: Worker 0 gets `""`, worker N gets `"N+1"`
* `PARALLEL_TEST_GROUPS`: Total number of workers
* `PARALLEL_TEST_PROCESSORS`: Compatible with parallel_tests

More information in the [Documentation](docs/index.md).
