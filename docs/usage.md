# Usage

## Basic Commands

### Running Tests

```bash
# Run all specs with the default worker count
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

### Excluding Tests From Discovery

Use `--exclude-pattern` (repeatable) to drop matching files from the test plan
before workers run. Patterns use doublestar semantics.

```bash
# Skip a single file
plur --exclude-pattern 'spec/legacy/old_spec.rb'

# Skip all system specs
plur --exclude-pattern 'spec/system/**/*_spec.rb'

# Multiple patterns OR together
plur --exclude-pattern 'spec/system/**/*_spec.rb' \
     --exclude-pattern 'spec/legacy/**/*_spec.rb'
```

Excludes can also be configured per-job in `.plur.toml`. CLI excludes are
*additive on top of* configured excludes — they do not replace them. See
[Configuration](configuration.md) for details.

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

### Rails And Rake Commands

```bash
plur rails db:prepare -n 4
plur rails db:migrate VERSION=20260429000000 -n 4
plur rails db:migrate -n 4 -- --trace
plur rake db:setup -n 4
plur rake db:create db:migrate -n 4
plur rake -n 1 -- --tasks
```

These commands run the configured job once per worker, with `PARALLEL_TEST_GROUPS` and `TEST_ENV_NUMBER` set. Arguments are appended literally — they're not treated as test file patterns. Put Plur flags like `-n` before `--`; arguments after `--` are passed through to Rails/Rake.

## Command Line Options

### Global Options

* `-n, --workers NUMBER` - Number of parallel workers (default: 4)
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

### Default Behavior

Plur uses 4 workers by default:
- Override with `-n` or `--workers`
- Respects `PARALLEL_TEST_PROCESSORS` if set
- Project config can set a different default

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

1. **Start with the default**: Try `4` workers first
2. **Measure and adjust**: Experiment with different worker counts
3. **Consider test characteristics**:
- Many small tests: More workers
- Few large tests: Fewer workers
- I/O heavy tests: More workers than CPU cores

## Next Steps

- See [Configuration](configuration.md) for customization
- See [Architecture](architecture/index.md) for technical details
- See [Features](features/index.md) for detailed feature documentation
