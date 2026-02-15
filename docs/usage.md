# Usage

## Basic Commands

### Running Tests

```bash
# Run all specs with auto-detected parallelism
plur

# Explicit "spec" command (same as default)
plur spec

# Specify number of workers
plur -n 4
plur --workers 8

# Dry run - see what would be executed
plur --dry-run

# Run from another directory (like git -C)
plur -C path/to/project
```

### Selecting Test Framework

Plur auto-detects your test framework (RSpec or Minitest) based on directory structure, but you can override this:

```bash
# Run RSpec tests explicitly
plur --use=rspec

# Run Minitest tests
plur --use=minitest

# Set default in config file
echo 'use = "rspec"' > .plur.toml
plur  # Now runs RSpec by default
```

**Projects with both spec/ and test/ directories**:

When both exist, plur defaults to RSpec. Use the `--use` flag to select:

```bash
plur                    # Runs RSpec tests (default)
plur --use=rspec        # Explicitly run RSpec tests
plur --use=minitest     # Run Minitest tests
```

Or set a permanent default in `.plur.toml`:

```toml
use = "minitest"  # Override default to use Minitest
```

### Watch Mode

```bash
# Watch for changes and re-run tests
plur watch

# Install the watcher binary if needed
plur watch install

# Customize debounce/timeout and ignore patterns
plur watch run --debounce 250 --timeout 60 --ignore "vendor/**" --ignore "tmp/**"
```

### Doctor Command

```bash
# Run diagnostics and troubleshooting
plur doctor
```

## Command Line Options

### Global Options

* `-n, --workers NUMBER` - Number of parallel workers (default: auto-detect)
* `--dry-run` - Show what would run without executing
* `-h, --help` - Show help
* `-v, --verbose` - Enable verbose logging
* `--version` - Show version

### Environment Variables

* `PARALLEL_TEST_PROCESSORS` - Override number of workers
* `PLUR_DEBUG` - Enable debug logging
* `PLUR_CONFIG_FILE` - Load a specific config file
* `PLUR_HOME` - Override Plur's home directory (`~/.plur`)

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

RSpec output includes timing details, for example:
```
Finished in 0.35 seconds (files took 0.12 seconds to load)
```

### Debugging Test Failures

```bash
# Run with debug output
plur --debug

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
