# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Purpose

This is a research repository for `rux`, a Go-based parallel test runner for Ruby/RSpec projects designed to outperform existing solutions like turbo_tests and parallel_tests. The project is production-ready with demonstrated 13% performance improvements.

## Memories

- Do not add Claude attribution to commit messages

## Core Commands

### Building Rux
```bash
# Development build (with automatic version detection)
cd rux/
go build .

# Release build with version info (recommended)
rake build_release              # Uses default v0.5.0
VERSION=v1.0.0 rake build_release  # Custom version

# Install to $GOPATH/bin
rake install

# Check version
rux --version
# Output: v0.5.0-20250528-0822-3993bb087
```

### Running Tests
```bash
# Main rux usage
rux                          # Auto-detect workers (cores-2)
rux -n 4                    # Specific worker count (often optimal)
rux --dry-run               # Preview execution plan
rux spec/specific_spec.rb    # Run specific files
rux --trace                  # Enable performance tracing

# Environment configuration
export PARALLEL_TEST_PROCESSORS=4
rux                          # Uses environment variable
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

### Benchmarking New Changes
```bash
# Automated comparison
./script/bench ./rux-ruby

# Manual verification
cd rux-ruby/
time bundle exec turbo_tests
time ../rux/rux -n 4
```

## Performance Characteristics

Based on example-project benchmarks (24 spec files):
- **rux -n 4**: 9.04s (fastest)
- **rux default**: 10.15s (+12% slower)
- **turbo_tests**: 10.18s (+13% slower)

Worker count optimization is project-dependent - use benchmarking to find the sweet spot.

## Testing Infrastructure

The repository includes multiple test projects:
- **rux-ruby/**: Custom project with diverse spec patterns
- **Downloaded repos**: Use get-repo script for real-world testing
- **Reference implementations**: references/parallel_tests and references/turbo_tests for comparison

All .git directories have been removed to focus on testing functionality rather than git operations.

### Ruby Integration Tests

The `spec/` directory contains Ruby RSpec tests that exercise rux from the outside:
- **general_integration_spec.rb**: Core functionality tests (running specs, exit codes, worker counts)
- **parallel_execution_spec.rb**: Tests parallel execution behavior
- **error_handling_spec.rb**: Tests error scenarios and output
- **colorized_output_spec.rb**: Tests ANSI color output
- **performance_spec.rb**: Performance regression tests
- **database_tasks_spec.rb**: Database preparation tests

Key integration test for development:
```ruby
# Simple test that runs a single spec file with all passing tests
it "exits with zero status when all tests pass" do
  Dir.chdir(test_project_path) do
    system("#{rux_binary} spec/calculator_spec.rb 2>&1", out: File::NULL)
    expect($?.exitstatus).to eq(0)
  end
end
```

Use these integration tests as guideposts when making architectural changes to rux.