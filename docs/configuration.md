Rux aims for zero-configuration operation, but provides options for customization when needed.

## Configuration Methods

Configuration precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Built-in defaults

## Worker Configuration

Rux uses intelligent distribution of specs/tests across workers:
- **Runtime-based**: When historical runtime data exists, tests are distributed based on previous execution times for optimal load balancing
- **Size-based**: When no runtime data exists, tests are distributed based on file sizes as a heuristic for complexity

Note: Watch mode (`rux watch`) runs tests serially without parallel execution.

### Specifying Number of Workers

```bash
# Auto-detection (default)
rux

# specify number of workers
rux -n 8
rux --workers 8

# or via environment variable
export PARALLEL_TEST_PROCESSORS=8
rux
```

## Output Configuration

### Formatters

Rux always uses dual formatters:
- Progress formatter (for visual feedback)
- JSON formatter (for result parsing)

### Verbosity

```bash
# Debug output
export RUX_DEBUG=1
rux

# Trace mode (performance profiling)
export RUX_TRACE=1
rux
# or
rux --trace
```

## File Discovery

### Current Behavior

- Recursively finds all `*_spec.rb` files
- Starts from current directory
- Excludes `vendor/` directory

## Performance Tuning

### Trace Output

```bash
# Enable tracing
rux --trace

# Trace files are saved to
/tmp/rux-traces/rux-trace-{timestamp}-{pid}/
```

## Watch Mode Configuration

### File Watching

Uses an embedded [e-dant/watcher binary](https://github.com/e-dant/watcher) with support for Ruby and Rails conventions. The watcher automatically detects changes in:
- `spec/` directory for test files
- `lib/` directory for source files (mapped to corresponding specs)
- `app/` directory for Rails applications

## Environment Variables

### Recognized Variables

- `PARALLEL_TEST_PROCESSORS` - Number of workers
- `RUX_DEBUG` - Enable debug output
- `RUX_TRACE` - Enable performance tracing

### RSpec Compatibility

Rux passes through RSpec-specific environment variables:
- `SPEC_OPTS`
- `RSPEC_OPTS`

## Best Practices

1. **Start with defaults** - Rux's auto-detection works well for most projects
2. **Use environment variables in CI** - Easier to adjust without code changes
3. **Enable tracing for optimization** - Identify bottlenecks before tuning
4. **Document your configuration** - Help teammates understand customizations

## Debugging Configuration

```bash
# Show what rux detects
rux --auto
```

## Next Steps

- See [Performance Tracing](architecture/performance-tracing.md) for optimization
- See [Usage](usage.md) for command examples
- See [Development](development/) for contributing