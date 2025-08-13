Plur aims for zero-configuration operation, but provides flexible configuration options through TOML files, environment variables, and command-line flags.

## Configuration Methods

Plur supports multiple configuration methods with the following precedence (highest to lowest):

1. Command-line flags
2. `.plur.toml` (project-specific configuration)
3. `~/.plur.toml` (user-specific configuration)
4. Environment variables
5. Built-in defaults

## Configuration Files (TOML)

Plur automatically loads configuration from TOML files using the following search order:

1. `.plur.toml` in the current directory (project-specific)
2. `~/.plur.toml` in your home directory (user-specific)

### Basic Example

```toml
# .plur.toml
workers = 4
color = true

[spec]
command = "bin/rspec"

[watch.run]
command = "bin/rspec --no-coverage"
debounce = 200
```

### Available Options

#### Global Settings
- `workers` - Number of parallel workers (default: auto-detect)
- `color` - Enable colored output (default: true)
- `verbose` - Enable verbose output (default: false)
- `command` - Default test command (default: "bundle exec rspec")

#### Command-Specific Settings

##### `[spec]` section
Settings for `plur` or `plur spec` commands:
- `command` - Test command override
- `type` - Test framework ("rspec" or "minitest")

##### `[watch.run]` section
Settings for `plur watch` command:
- `command` - Test command override for watch mode
- `debounce` - Delay in milliseconds before running tests (default: 100)
- `type` - Test framework ("rspec" or "minitest")

### Configuration Examples

See the `examples/` directory for complete configuration examples:
- `plur.toml.example` - Comprehensive example with all options
- `plur.toml.simple` - Basic configuration for most projects
- `plur.toml.rails` - Rails-optimized configuration
- `plur.toml.minitest` - Minitest project configuration

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

### Glob Pattern Support

Plur supports advanced glob patterns for selecting test files:

- `**` - Matches any number of directories (e.g., `spec/**/*_spec.rb`)
- `*` - Matches any characters except path separator
- `?` - Matches single character
- `[abc]` - Matches any character in brackets
- `{models,controllers}` - Brace expansion (e.g., `spec/{models,controllers}/**/*_spec.rb`)

### Pattern Examples

```bash
# Run specific pattern
plur 'spec/**/*_spec.rb'          # All specs recursively
plur 'spec/*_spec.rb'              # Only top-level specs
plur 'spec/models/**/*_spec.rb'    # All model specs
plur 'spec/{models,controllers}/**/*_spec.rb'  # Multiple directories

# Directory shorthand
plur spec/                         # Expands to spec/**/*_spec.rb
plur spec/models/                  # Expands to spec/models/**/*_spec.rb

# Single files (passed through even if not *_spec.rb)
plur spec/user_spec.rb             # Specific file
plur spec/spec_helper.rb           # Warning shown but runs
```

### RSpec Compatibility

Plur matches RSpec's behavior:
- **Directories**: Automatically append `**/*_spec.rb` pattern
- **Single files**: Pass through with warning if not matching test suffix
- **Glob patterns**: Filter results to only test files

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