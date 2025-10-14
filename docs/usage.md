# Usage

## Basic Commands

### Running Tests

```bash
# Run all specs with auto-detected parallelism
plur

# Specify number of workers
plur -n 4
plur --workers 8

# Dry run - see what would be executed
plur --dry-run

# Show auto-detected worker count
plur --auto
```

### Selecting Test Framework

Plur auto-detects your test framework (RSpec or Minitest) based on directory structure, but you can override this:

```bash
# Run RSpec tests (from spec/ directory)
plur --use=rspec

# Run Minitest tests (from test/ directory)
plur --use=minitest

# Set default in config file
echo 'use = "rspec"' > .plur.toml
plur  # Now runs RSpec by default
```

**Projects with both spec/ and test/ directories**:

When both exist, plur defaults to Minitest. Use `--use` to select:

```bash
plur --use=rspec     # Run RSpec tests
plur --use=minitest  # Run Minitest tests
```

Or set a permanent default in `.plur.toml`:

```toml
use = "rspec"  # or "minitest"
```

### Watch Mode

```bash
# Watch for changes and re-run tests
plur watch

# Watch with specific number of workers
plur watch -n 4
```

### Doctor Command

```bash
# Run diagnostics and troubleshooting
plur doctor
```

## Command Line Options

### Global Options

- `-n, --workers NUMBER` - Number of parallel workers (default: auto-detect)
- `--dry-run` - Show what would run without executing
- `--auto` - Show auto-detected worker count and exit
- `-h, --help` - Show help
- `-v, --version` - Show version

### Environment Variables

- `PARALLEL_TEST_PROCESSORS` - Override number of workers
- `PLUR_DEBUG` - Enable debug logging

## Parallelism

### Auto-Detection

Plur automatically detects the optimal number of workers:
- Default: `CPU cores - 2` (minimum 1)
- Leaves headroom for system responsiveness
- Respects `PARALLEL_TEST_PROCESSORS` if set

### Manual Control

```bash
# Use all cores
plur -n $(nproc)

# Conservative - half the cores
plur -n $(( $(nproc) / 2 ))

# CI environments often benefit from more workers
plur -n $(( $(nproc) + 2 ))
```

## Output Formats

### Progress Output (Default)

Shows dots for test progress:
```
....F...*...
```
- `.` - Passing test
- `F` - Failing test
- `*` - Pending test

### JSON Output

Plur uses dual formatters internally:
- Progress formatter for visual feedback
- JSON formatter for parsing results

## Performance Monitoring

### Basic Timing

Plur shows execution time after each run:
```
Finished in 12.34s (CPU: 45.67s)
```

### Debugging Test Failures

```bash
# Run with debug output
PLUR_DEBUG=1 plur

# Check which files would run
plur --dry-run | grep "file_spec.rb"

# Run doctor for diagnostics
plur doctor
```

### Performance Tuning

1. **Start with auto-detection**: Let Plur choose worker count
2. **Measure and adjust**: Experiment with different worker counts
3. **Consider test characteristics**:
   - Many small tests: More workers
   - Few large tests: Fewer workers
   - I/O heavy tests: More workers than CPU cores

## Next Steps

- See [Configuration](configuration.md) for customization
- See [Architecture](architecture/index.md) for technical details
- See [Features](features/index.md) for detailed feature documentation