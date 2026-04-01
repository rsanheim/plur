# Plur Project Status

## Overview

Plur is a Go-based parallel test runner for Ruby projects (RSpec and Minitest) that distributes test files across worker processes for faster execution. It's a fast alternative to turbo_tests and parallel_tests, typically ~13% faster on real-world projects.

## Core Features

* **Parallel test execution** using Go goroutines and a worker pool
* **Multi-framework support**: RSpec, Minitest, and Go test out of the box
* **Runtime-based distribution**: tracks historical test runtimes and balances worker loads accordingly, falling back to file-size estimation for new files
* **Tag filtering**: `--tag` flag passes RSpec tags through to workers
* **Exclude patterns**: `--exclude-pattern` removes matching test files before worker grouping
* **Argument passthrough**: `--` forwards arbitrary flags to the underlying test command
* **TOML configuration**: local `.plur.toml` and global `~/.plur.toml` for persistent settings
* **Custom jobs**: define your own jobs in config with arbitrary commands, frameworks, and target patterns
* **Watch mode**: automatically runs tests on file changes with configurable source-to-test mappings
* **Intelligent worker count**: defaults to cores-2, configurable via `-n` flag or `PARALLEL_TEST_PROCESSORS` env var
* **Glob-based file discovery** using doublestar for recursive `**` patterns
* **Streaming JSON output** via a custom `JsonRowsFormatter` embedded in the binary
* **Diagnostic command**: `plur doctor` for debugging installation and environment issues
* **Cross-version benchmarking**: `script/bench-git` compares performance across git refs

## CLI Interface

```bash
plur                          # Run with auto-detected workers (cores-2)
plur -n 4                     # Run with 4 workers
plur --dry-run                # Show what would run without execution
plur --tag focus              # Filter RSpec tests by tag
plur --exclude-pattern 'spec/system/**/*_spec.rb'  # Exclude matching files
plur --use minitest           # Override framework autodetection
plur spec/models              # Run a subset of specs
plur -- --seed 12345          # Pass flags through to the test command
plur watch                    # Watch for file changes and re-run tests
plur doctor                   # Diagnose installation issues
plur config init              # Generate a starter .plur.toml
```
