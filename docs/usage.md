# Usage

## Basic Commands

### Running Tests

```bash
# Run all specs with auto-detected parallelism
rux

# Specify number of workers
rux -n 4
rux --workers 8

# Dry run - see what would be executed
rux --dry-run

# Show auto-detected worker count
rux --auto
```

### Watch Mode

```bash
# Watch for changes and re-run tests
rux watch

# Watch with specific number of workers
rux watch -n 4
```

### Doctor Command

```bash
# Run diagnostics and troubleshooting
rux doctor
```

## Command Line Options

### Global Options

- `-n, --workers NUMBER` - Number of parallel workers (default: auto-detect)
- `--dry-run` - Show what would run without executing
- `--auto` - Show auto-detected worker count and exit
- `--trace` - Enable performance tracing
- `-h, --help` - Show help
- `-v, --version` - Show version

### Environment Variables

- `PARALLEL_TEST_PROCESSORS` - Override number of workers
- `RUX_TRACE` - Enable trace output (same as --trace)
- `RUX_DEBUG` - Enable debug logging

## Parallelism

### Auto-Detection

Rux automatically detects the optimal number of workers:
- Default: `CPU cores - 2` (minimum 1)
- Leaves headroom for system responsiveness
- Respects `PARALLEL_TEST_PROCESSORS` if set

### Manual Control

```bash
# Use all cores
rux -n $(nproc)

# Conservative - half the cores
rux -n $(( $(nproc) / 2 ))

# CI environments often benefit from more workers
rux -n $(( $(nproc) + 2 ))
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

Rux uses dual formatters internally:
- Progress formatter for visual feedback
- JSON formatter for parsing results

## Performance Monitoring

### Basic Timing

Rux shows execution time after each run:
```
Finished in 12.34s (CPU: 45.67s)
```

### Trace Mode

Enable detailed performance tracing:
```bash
rux --trace
# or
RUX_TRACE=1 rux
```

Creates trace files in `/tmp/rux-traces/` for analysis.

## Integration

### CI/CD

```yaml
# GitHub Actions
- name: Run tests
  run: rux -n ${{ steps.cpu-cores.outputs.count }}

# CircleCI
test:
  parallelism: 4
  steps:
    - run: rux -n $CIRCLE_NODE_TOTAL

# GitLab CI
test:
  script:
    - rux -n $(nproc)
```

### Git Hooks

```bash
# .git/hooks/pre-push
#!/bin/sh
rux --dry-run && rux -n 4
```

## Advanced Usage

### Test Discovery

Rux discovers all `*_spec.rb` files recursively from the current directory.

### Debugging Test Failures

```bash
# Run with debug output
RUX_DEBUG=1 rux

# Check which files would run
rux --dry-run | grep "file_spec.rb"

# Run doctor for diagnostics
rux doctor
```

### Performance Tuning

1. **Start with auto-detection**: Let Rux choose worker count
2. **Measure and adjust**: Use `--trace` to identify bottlenecks
3. **Consider test characteristics**:
   - Many small tests: More workers
   - Few large tests: Fewer workers
   - I/O heavy tests: More workers than CPU cores

## Next Steps

- See [Configuration](configuration.md) for customization
- See [Architecture](architecture/) for technical details
- See [Features](features/) for detailed feature documentation