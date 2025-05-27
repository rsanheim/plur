# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Purpose

This is a research repository for `rux`, a Go-based parallel test runner for Ruby/RSpec projects designed to outperform existing solutions like turbo_tests and parallel_tests. The project is production-ready with demonstrated 13% performance improvements.

## Memories

- Do not add Claude attribution to commit messages

## Core Commands

### Building Rux
```bash
cd rux/
go install .
# This installs rux to $GOPATH/bin (usually ~/go/bin)
# Make sure $GOPATH/bin is in your PATH
```

### Running Tests
```bash
# Main rux usage
rux                          # Auto-detect workers (cores-2)
rux -n 4                    # Specific worker count (often optimal)
rux --dry-run               # Preview execution plan
rux spec/specific_spec.rb    # Run specific files

# Environment configuration
export PARALLEL_TEST_PROCESSORS=4
rux                          # Uses environment variable
```

### Performance Benchmarking
```bash
# Compare rux vs turbo_tests
./script/bench ./rux-ruby
./script/bench /path/to/any/ruby/project

# Results: bench-results.md and bench-results.json
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
- **rux-ruby/**: Test Ruby project (9 spec files across nested dirs)
- **parallel_tests/**: Reference Ruby implementation for study
- **turbo_tests/**: Reference Ruby implementation for comparison
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
- **Reference implementations**: parallel_tests and turbo_tests for comparison

All .git directories have been removed to focus on testing functionality rather than git operations.