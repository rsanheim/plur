# plur

`plur` is a Go-based parallel test runner for Ruby projects (RSpec and Minitest) designed to outperform existing solutions like turbo_tests and parallel_tests.

## Quick Start

_TODO_ this needs to be rewritten from a user's perspective, not a dev's

## What's Included

* Go-based CLI for parallel RSpec execution
* Database commands (db:create, db:migrate, db:setup, db:test:prepare)
* Performance optimized - 13% faster than turbo_tests (TODO: update this)

## Supported Platforms

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* _Experimental_ Windows x86_64

Watch mode (`plur watch`) requires platform-specific binaries. All platforms support standard test execution.

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

See `examples/` directory for more configuration examples.

### Environment Variables
* `TEST_ENV_NUMBER`: Worker 0 gets `""`, worker N gets `"N+1"`
* `PARALLEL_TEST_GROUPS`: Total number of workers
* `PARALLEL_TEST_PROCESSORS`: Compatible with parallel_tests

## Performance Results

TODO

## Testing

### Run All Tests
```bash
rake                         # Run ALL tests & lints before committing
rake test                    # Run full Ruby test suite
rake test:go                 # Run Go tests only
```

### Test with Default Projects
```bash
# Simple Ruby library
cd fixtures/projects/default-ruby
plur                        # Run all specs

# Rails application
cd fixtures/projects/default-rails
plur db:create -n 3        # Set up databases
plur db:migrate -n 3       # Run migrations
plur -n 3                  # Run RSpec tests in parallel
```

### Benchmarking
```bash
./script/bench              # Benchmark default projects
./script/bench -p fixtures/projects/default-rails    # Benchmark Rails app
./script/bench -p fixtures/projects/default-ruby     # Benchmark Ruby library
```
