# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Purpose

This is a research repository for `rux`, a Go-based parallel test runner for Ruby/RSpec projects designed to outperform existing solutions like turbo_tests and parallel_tests. The project is production-ready with demonstrated 13% performance improvements.

## Core Commands

### 🔨 IMPORTANT: Always Build with Rake Commands

**DO NOT use `go build` directly!** Always use the Rake commands below for proper builds:

### Building Rux

```bash
# Release build with version info (recommended)
rake build                      # Alias for build_release
rake build_release              # Uses version from VERSION file (currently 0.6.0)
VERSION=v1.0.0 rake build_release  # Override with custom version
rake install                    # Installs to $GOPATH/bin for global dev usage
rux --version                  # Check version
rake clean                      # Clean build artifacts
```


### Testing & Development Tasks
```bash
# Run all tests
rake test                    # Alias for test:all
rake test:all                # Run Go, Ruby, and Integration tests
rake test:go                 # Run Go unit tests
rake test:ruby               # Build rux and run Ruby tests
rake test:integration        # Build rux and run integration specs
VERBOSE=1 rake test:go       # Show detailed test output

# Individual test projects
rake test:rux_ruby           # Run rux-ruby specs with rux
rake test:rux_ruby_turbo     # Run rux-ruby specs with turbo_tests
rake test:test_app           # Run test_app specs with rux
```

### Testing from the Outside-in

Rux development should be driven from the outside-in via Ruby integration specs in the `spec/` directory.
Run these via `rake test:ruby` or directly via `rspec [filename]`.

You MUST use these integration tests as guardrails when making architectural changes to rux.

- **general_integration_spec.rb**: Core functionality tests (running specs, exit codes, worker counts)
- **parallel_execution_spec.rb**: Tests parallel execution behavior
- **error_handling_spec.rb**: Tests error scenarios and output
- **colorized_output_spec.rb**: Tests ANSI color output
- **performance_spec.rb**: Performance regression tests
- **database_tasks_spec.rb**: Database preparation tests

### Linting & Code Quality
```bash
# Run all linting
rake lint                    # Alias for lint:all  
rake lint:all                # Run Go and Ruby linting
rake lint:go                 # Run go fmt, go vet, and golint
rake lint:ruby               # Run Standard Ruby linter
rake lint:ruby_fix           # Auto-fix Ruby linting issues

# Default task (runs all tests and linting)
rake                         # Same as: rake test:all lint:all
```

### CI Tasks
```bash
# Run all CI checks
rake ci:all                  # Run all linting and tests
rake ci:go                   # Run Go linting and tests
rake ci:ruby                 # Run Ruby linting and tests
```

### Using Rux

```bash
rux                          # Auto-detect workers (cores-2)
rux -n 4                    # Specific worker count (often optimal)
rux --dry-run               # Preview execution plan
rux spec/specific_spec.rb    # Run specific files
rux --trace                  # Enable performance tracing
```

### Performance Tracing
```bash
# Enable tracing to analyze performance
rux --trace -n 4

# Trace files are written to repo tmp directory
# Output: "Tracing enabled, writing to: ./tmp/rux-traces/rux-trace-TIMESTAMP.json"

# Analyze trace results
ruby rux/analyze_trace.rb -v ./tmp/rux-traces/rux-trace-*.json

# Key metrics traced:
# - Process spawn time (~1ms per process)
# - Ruby startup time (~190ms from spawn to first output)
# - RSpec load time (~45ms as reported by RSpec)
# - Total overhead (typically <10ms or <1% for large test suites)
```

### Performance Benchmarking
```bash
# Compare rux vs turbo_tests
./script/bench ./rux-ruby
./script/bench /path/to/any/ruby/project

# Run benchmarks with tracing enabled
./script/bench --trace ./rux-ruby

# Using Rake tasks
rake bench:all               # Run all benchmarks
rake bench:rux_ruby          # Benchmark rux-ruby project
rake bench:test_app          # Benchmark test_app project

# Results: bench-results.md and bench-results.json
# Trace analysis is included when --trace is used
```

### Repository Testing
```bash
# Clone any GitHub repo for testing
./script/get-repo https://github.com/owner/repo
./script/get-repo https://github.com/owner/repo custom-name
```

## Architecture

### Project Structure
- **rux/**: Main Go implementation (production binary)
- **rux-ruby/**: Example Ruby project (9 spec files across nested dirs)
- **test_app**: Example Rails app for integration tests
- **references/parallel_tests/**: Reference Ruby implementation for study
- **references/turbo_tests/**: Reference Ruby implementation for comparison
- **script/**: Benchmarking and testing utilities
- **docs/**: Project status and usage documentation

### Rux Implementation Details
- **Language**: Go 1.22+ with urfave/cli/v2 framework
- **Concurrency**: Worker pool pattern using goroutines and sync.WaitGroup
- **File Discovery**: Recursive search for `*_spec.rb` using filepath.WalkDir
- **Output Strategy**: RSpec's dual formatters (`--format progress --format json`)
- **Worker Management**: Intelligent defaults (cores-2) with CLI/env overrides

### Key Technical Decisions
- Uses RSpec's built-in progress formatter for clean dot output
- Avoids verbose completion messages from individual processes
- Implements worker pool to prevent system overload
- Compatible with `PARALLEL_TEST_PROCESSORS` environment variable
- Shows wall time vs CPU time for accurate parallel performance metrics
- **Runtime-based test distribution**: Automatically tracks and uses test execution times for optimal load balancing
- **Channel-based output aggregation**: Eliminates lock contention for high worker counts (25-30+)

## Development Workflow

### Quick Test Cycle
```bash
cd rux/ && go build -o rux main.go
cd ../rux-ruby/
../rux/rux --dry-run        # Verify discovery
../rux/rux                  # Execute tests
```

### Finding Optimal Performance
```bash
# Test different worker counts
rux -n 1         # Sequential baseline
rux -n 4         # Often optimal (benchmark winner)  
rux -n 8         # High parallelism
rux               # Auto-detect default
```

## Testing Infrastructure

The repository includes multiple test projects:
- **rux-ruby/**: Custom project with diverse spec patterns
- **Downloaded repos**: Use get-repo script for real-world testing
- **Reference implementations**: references/parallel_tests and references/turbo_tests for comparison

All .git directories have been removed to focus on testing functionality rather than git operations.
