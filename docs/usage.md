# Rux Usage Guide

## Installation

```bash
# Build from source
cd rux/
go build -o rux main.go

# Add to PATH (optional)
cp rux /usr/local/bin/
```

## Basic Usage

### Running Tests

```bash
# Run all specs with auto-detected workers (cores-2)
rux

# Run with specific number of workers
rux --workers 4
rux --workers 8

# See what would run without executing
rux --dry-run

# Show auto-detected worker count
rux --auto
```

### Environment Variables

```bash
# Set worker count via environment (compatible with parallel_tests)
export PARALLEL_TEST_PROCESSORS=4
rux

# Override environment with CLI flag
PARALLEL_TEST_PROCESSORS=8 rux --workers 4  # Uses 4 workers
```

## Performance Tuning

### Worker Count Guidelines

- **Default (cores-2)**: Good starting point for most systems
- **cores-1**: Maximum parallelism without overwhelming system
- **4 workers**: Often optimal for medium projects (our benchmarks show best results)
- **8+ workers**: May help on very large test suites or high-core systems

### Finding Optimal Workers

```bash
# Test different worker counts
rux --workers 1   # Baseline (sequential-ish)
rux --workers 2   # Light parallelism
rux --workers 4   # Often optimal
rux --workers 8   # High parallelism
rux               # Auto-detect (cores-2)
```

Use the benchmark script to find your project's sweet spot:

```bash
./script/bench /path/to/your/ruby/project
```

## Output Formats

### Progress Output (Default)
```
$ rux
Found 24 spec files
Running tests with 6 workers...
........................

Tests completed in 9.04s (wall time) vs 15.10s (CPU time)
```

### Dry Run
```
$ rux --dry-run
Found 24 spec files
Would run tests with 6 workers:
  spec/dx_spec.rb
  spec/integration/cli_integration_spec.rb
  spec/lib/example-project/cli_spec.rb
  ...
```

## Benchmarking

### Compare Against turbo_tests

```bash
# Benchmark your project
./script/bench /path/to/ruby/project

# Results saved to:
# - bench-results.md (table format)
# - bench-results.json (detailed data)
```

### Interpreting Results

The benchmark compares:
- `bundle exec turbo_tests` 
- `rux` (default workers)
- `rux --workers 4`
- `rux --workers 8`

Look for:
- **Wall time**: Total time to completion
- **Relative performance**: Which is fastest
- **Statistical outliers**: System interference warnings

## Troubleshooting

### No Spec Files Found
```bash
# Check if you're in a Ruby project root
ls spec/

# Verify spec files exist
find . -name "*_spec.rb"
```

### Performance Issues

```bash
# Try different worker counts
rux --workers 1   # Minimal parallelism
rux --workers 2   # Light parallelism

# Check system resources
top              # CPU usage
htop             # Better process viewer
```

### Memory Issues

```bash
# Reduce workers if running out of memory
rux --workers 2

# Monitor memory usage during tests
watch -n 1 'ps aux | grep rspec'
```

## Integration

### CI/CD Pipelines

```yaml
# GitHub Actions example
- name: Run tests with rux
  run: |
    cd my-ruby-project
    rux --workers 4
```

### Docker

```dockerfile
# Add rux binary to Ruby container
COPY rux /usr/local/bin/
RUN chmod +x /usr/local/bin/rux

# Use in test command
CMD ["rux", "--workers", "4"]
```

### Makefiles

```makefile
test:
	rux

test-fast:
	rux --workers 8

test-benchmark:
	./script/bench .
```

## Compatibility

### RSpec Versions
- Compatible with RSpec 3.x
- Uses standard `--format progress` and `--format json` options
- No special RSpec configuration required

### Ruby Versions
- Works with any Ruby version that supports RSpec
- Tested with Ruby 2.7+

### Project Structure
- Expects `spec/` directory with `*_spec.rb` files
- Handles nested directories automatically
- Works with Rails and non-Rails projects