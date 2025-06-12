# Configuration

Rux aims for zero-configuration operation, but provides options for customization when needed.

## Configuration Methods

Configuration precedence (highest to lowest):
1. Command-line flags
2. Environment variables
3. Configuration file (coming soon)
4. Defaults

## Worker Configuration

### Number of Workers

```bash
# Command line
rux -n 8
rux --workers 8

# Environment variable
export PARALLEL_TEST_PROCESSORS=8
rux

# Auto-detection (default)
rux  # Uses CPU cores - 2
```

### Worker Strategy

Currently, Rux uses a simple round-robin distribution. Future versions will support:
- Runtime-based distribution
- File-size based grouping
- Custom distribution strategies

## Output Configuration

### Formatters

Rux always uses dual formatters:
- Progress formatter (for visual feedback)
- JSON formatter (for result parsing)

Custom formatters coming in future releases.

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

### Future Options

```yaml
# .rux.yml (coming soon)
test_files:
  pattern: "**/*_spec.rb"
  exclude:
    - vendor/
    - tmp/
    - coverage/
```

## Performance Tuning

### Trace Output

```bash
# Enable tracing
rux --trace

# Trace files are saved to
/tmp/rux-traces/rux-trace-{timestamp}-{pid}/
```

### Performance Options (Future)

```yaml
# .rux.yml (coming soon)
performance:
  file_grouping:
    enabled: true
    min_group_size: 5
    max_group_size: 20
  
  runtime_tracking:
    enabled: true
    history_file: .rux-runtimes.json
```

## Watch Mode Configuration

### File Watching

Currently uses embedded watcher binary. Future configuration:

```yaml
# .rux.yml (coming soon)
watch:
  paths:
    - lib/
    - spec/
  ignore:
    - "*.log"
    - tmp/
  
  mappings:
    - pattern: "lib/(.*)\.rb"
      run: "spec/{1}_spec.rb"
```

## Environment Variables

### Recognized Variables

- `PARALLEL_TEST_PROCESSORS` - Number of workers
- `RUX_DEBUG` - Enable debug output
- `RUX_TRACE` - Enable performance tracing
- `RUX_NO_COLOR` - Disable colored output (future)

### RSpec Compatibility

Rux passes through RSpec-specific environment variables:
- `SPEC_OPTS`
- `RSPEC_OPTS`

## Configuration File (Coming Soon)

Future releases will support `.rux.yml`:

```yaml
# .rux.yml
version: 1

# Worker configuration
workers:
  count: auto  # auto, number, or percentage (e.g., "75%")
  strategy: round-robin  # round-robin, runtime-balanced, size-grouped

# Test discovery
test_files:
  pattern: "spec/**/*_spec.rb"
  exclude:
    - spec/fixtures/
    - spec/support/cassettes/

# Output preferences
output:
  format: progress  # progress, documentation, json
  color: auto      # auto, always, never
  
# Performance
performance:
  trace: false
  profile: false
  
# Watch mode
watch:
  enabled: true
  debounce: 100ms
```

## Best Practices

1. **Start with defaults** - Rux's auto-detection works well for most projects
2. **Use environment variables in CI** - Easier to adjust without code changes
3. **Enable tracing for optimization** - Identify bottlenecks before tuning
4. **Document your configuration** - Help teammates understand customizations

## Debugging Configuration

```bash
# Show effective configuration (coming soon)
rux config

# Validate configuration file (coming soon)
rux config --validate

# Show what rux detects
rux --auto
```

## Next Steps

- See [Performance Tracing](architecture/performance-tracing.md) for optimization
- See [Usage](usage.md) for command examples
- See [Development](development/) for contributing