Plur aims for zero-configuration operation, but provides options for customization when needed.

## Configuration Methods

Configuration precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Built-in defaults

## Worker Configuration

Plur uses intelligent distribution of specs/tests across workers:
- **Runtime-based**: When historical runtime data exists, tests are distributed based on previous execution times for optimal load balancing
- **Size-based**: When no runtime data exists, tests are distributed based on file sizes as a heuristic for complexity

Note: Watch mode (`plur watch`) runs tests serially without parallel execution.

### Specifying Number of Workers

```bash
# Auto-detection (default)
plur

# specify number of workers
 plur -n 8
 plur --workers 8

# or via environment variable
export PARALLEL_TEST_PROCESSORS=8
plur
```

## Output Configuration

### Formatters

Plur always uses dual formatters:
- Progress formatter (for visual feedback)
- JSON formatter (for result parsing)

### Verbosity

```bash
# Debug output
export PLUR_DEBUG=1
plur

```

## File Discovery

### Current Behavior

- Recursively finds all `*_spec.rb` files
- Starts from current directory
- Excludes `vendor/` directory

## Watch Mode Configuration

### File Watching

Uses an embedded [e-dant/watcher binary](https://github.com/e-dant/watcher) with support for Ruby and Rails conventions. The watcher automatically detects changes in:
- `spec/` directory for test files
- `lib/` directory for source files (mapped to corresponding specs)
- `app/` directory for Rails applications

## Environment Variables

### Recognized Variables

- `PARALLEL_TEST_PROCESSORS` - Number of workers
- `PLUR_DEBUG` - Enable debug output

## Next Steps

- See [Usage](usage.md) for command examples
- See [Development](development/index.md) for contributing