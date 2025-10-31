Plur aims for zero-configuration operation, but provides flexible configuration options through TOML files, environment variables, and command-line flags.

## Configuration Methods

Plur supports multiple configuration methods with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables (e.g., `PARALLEL_TEST_PROCESSORS`)
3. `PLUR_CONFIG_FILE` environment variable (if set)
4. `.plur.toml` (project-specific configuration)
5. `~/.plur.toml` (user-specific configuration)
6. Built-in defaults

## Configuration Files (TOML)

Plur automatically loads configuration from TOML files using the following search order:

1. `PLUR_CONFIG_FILE` environment variable (if set, takes highest priority)
2. `.plur.toml` in the current directory (project-specific)
3. `~/.plur.toml` in your home directory (user-specific)

### Basic Example

```toml
# .plur.toml
workers = 4
color = true

[task.rspec]
run = "bin/rspec"

[task.minitest]
run = "bin/rake test"

[watch.run]
debounce = 200
```

### Available Options

#### Global Settings

* `workers` - Number of parallel workers (default: auto-detect)
* `color` - Enable colored output (default: true)
* `verbose` - Enable verbose output (default: false)
* `use` - Default task to use (default: auto-detect based on project structure)

## Task Configuration

Tasks are the core of Plur's test execution system. They define how to run tests, linters, or other commands, and how to map source files to test files.

### Task Overview

A Task in Plur encapsulates:

* The command to run
* Which directories to watch or search
* How to map source files to test files
* File patterns to match

Plur comes with built-in tasks for RSpec and Minitest, but you can define custom tasks for any tool.

### Task Selection Priority

Tasks are selected in the following priority order:

1. CLI flag: `plur spec -t custom-task`
2. Config file: `use = "custom-task"` in `.plur.toml`
3. Auto-detection: Based on directory structure (spec/ → rspec, test/ → minitest)

> **💡 Tip for Projects with Multiple Frameworks**
>
> If your project has both `spec/` and `test/` directories, plur will default to RSpec.
> Use the `-t` flag or config file setting to select a different framework:
>
> ```bash
> plur                    # Runs RSpec tests (default)
> plur spec -t minitest   # Run Minitest tests instead
> plur spec -t rspec      # Explicitly run RSpec tests
> ```
>
> Or add to `.plur.toml`:
> ```toml
> use = "minitest"  # Override default to use Minitest
> ```

### Task Configuration Fields

| Field | Type | Description | Required | Default |
|-------|------|-------------|----------|------|
| `description` | string | Human-readable description of the task | No | "" |
| `run` | string | Command to execute | Yes | "" |
| `source_dirs` | string[] | Directories to watch/search | No | `["spec", "lib", "app"]` (rspec)<br>`["test", "lib", "app"]` (minitest) |
| `mappings` | MappingRule[] | File mapping rules | No | `[]` |
| `test_glob` | string | Glob pattern for test files | No | Depends on task |

### Built-in Tasks

#### RSpec (default)
```toml
[task.rspec]
run = "bundle exec rspec"
source_dirs = ["spec", "lib", "app"]
test_glob = "spec/**/*_spec.rb"
mappings = [
  { pattern = "lib/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "app/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "spec/**/*_spec.rb", target = "{{file}}" }
]
```

#### Minitest
```toml
[task.minitest]
run = "ruby -Itest"  # Plur handles test file arguments specially for minitest
source_dirs = ["test", "lib", "app"]
test_glob = "test/**/*_test.rb"
mappings = [
  { pattern = "lib/**/*.rb", target = "test/{{path}}/{{name}}_test.rb" },
  { pattern = "app/**/*.rb", target = "test/{{path}}/{{name}}_test.rb" },
  { pattern = "test/**/*_test.rb", target = "{{file}}" }
]
```

### Custom Task Examples

#### Custom RSpec with Spring
```toml
[task.spring-rspec]
description = "RSpec with Spring preloader"
run = "bin/spring rspec"
source_dirs = ["spec", "lib", "app"]
test_glob = "spec/**/*_spec.rb"
mappings = [
  { pattern = "lib/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "app/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "spec/**/*_spec.rb", target = "{{file}}" }
]
```

#### Linter Task
```toml
[task.rubocop]
description = "Run RuboCop linter"
run = "bundle exec rubocop"
source_dirs = ["lib", "spec", "app"]
test_glob = "**/*.rb"
# No mappings needed - rubocop will lint the files directly
```

#### JavaScript Test Runner
```toml
[task.jest]
description = "Run Jest tests"
run = "npm test"
source_dirs = ["src", "test"]
test_glob = "test/**/*.test.js"
mappings = [
  { pattern = "src/**/*.js", target = "test/{{path}}/{{name}}.test.js" },
  { pattern = "test/**/*.test.js", target = "{{file}}" }
]
```

#### Go Tests
```toml
[task.go-test]
description = "Run Go tests"
run = "go test"
source_dirs = ["."]
test_glob = "**/*_test.go"
mappings = [
  { pattern = "**/*.go", target = "{{path}}/{{name}}_test.go" },
  { pattern = "**/*_test.go", target = "{{file}}" }
]
```

### Mapping Patterns

Mappings define how source files map to test files. They use a pattern matching system with tokens:

#### Available Tokens

* `{{file}}` - The complete matched file path
* `{{path}}` - The directory path of the matched file (without filename)
* `{{name}}` - The base filename without extension

#### Mapping Examples

Given a file `lib/models/user.rb`:

* `{{file}}` → `lib/models/user.rb`
* `{{path}}` → `lib/models`
* `{{name}}` → `user`

So the mapping:
```toml
{ pattern = "lib/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" }
```
Would map `lib/models/user.rb` → `spec/lib/models/user_spec.rb`

### Multiple Task Definitions

You can define multiple tasks and switch between them:

```toml
# .plur.toml
[task.rspec]
run = "bundle exec rspec"

[task.rspec-fast]
description = "RSpec with fail-fast"
run = "bundle exec rspec --fail-fast"
source_dirs = ["spec", "lib"]
test_glob = "spec/**/*_spec.rb"

[task.integration]
description = "Integration tests only"
run = "bundle exec rspec"
source_dirs = ["spec/integration"]
test_glob = "spec/integration/**/*_spec.rb"
```

Use them with:
```bash
plur --use=rspec-fast
plur --use=integration
```

### Watch Mode Integration

Tasks work seamlessly with watch mode. The `source_dirs` determine which directories are watched, and `mappings` determine which tests run when files change:

```bash
plur watch --use=custom-task
```

When a file changes:

1. Plur checks if it matches any mapping patterns
2. If matched, runs the corresponding target test files
3. If no match, runs the changed file directly (if it matches `test_glob`)

### `[watch.run]` section

Settings for `plur watch` command:

* `debounce` - Delay in milliseconds before running tests (default: 100)

## Worker Configuration

Plur uses intelligent distribution of specs/tests across workers:

* **Runtime-based**: When historical runtime data exists, tests are distributed based on previous execution times for optimal load balancing
* **Size-based**: When no runtime data exists, tests are distributed based on file sizes as a heuristic for complexity

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

* Progress formatter (for visual feedback)
* JSON formatter (for result parsing)

### Verbosity

```bash
# Debug output
export PLUR_DEBUG=1
plur

```

## File Discovery

### Glob Pattern Support

Plur supports advanced glob patterns for selecting test files:

* `**` - Matches any number of directories (e.g., `spec/**/*_spec.rb`)
* `*` - Matches any characters except path separator
* `?` - Matches single character
* `[abc]` - Matches any character in brackets
* `{models,controllers}` - Brace expansion (e.g., `spec/{models,controllers}/**/*_spec.rb`)

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

* **Directories**: Automatically append `**/*_spec.rb` pattern
* **Single files**: Pass through with warning if not matching test suffix
* **Glob patterns**: Filter results to only test files

## Watch Mode Configuration

### File Watching

Uses an embedded [e-dant/watcher binary](https://github.com/e-dant/watcher) with support for Ruby and Rails conventions. The watcher automatically detects changes in:

* `spec/` directory for test files
* `lib/` directory for source files (mapped to corresponding specs)
* `app/` directory for Rails applications

## Environment Variables

### Recognized Variables

* `PARALLEL_TEST_PROCESSORS` - Number of workers
* `PLUR_DEBUG` - Enable debug output

## Task Troubleshooting

### Tests Not Found

Check that your `test_glob` pattern matches your test files:

```bash
# List files that would be run
plur --dry-run --use=your-task
```

### Mappings Not Working

Verify your mappings with the watch find command:

```bash
plur watch find lib/models/user.rb --use=your-task
```

### Command Not Running

Ensure the `run` command is executable and in your PATH:

```bash
# Test the command directly
bundle exec rspec --version
```

## Tips and Best Practices

1. **Start Simple**: Begin with just overriding the `run` command for existing tasks
2. **Use Descriptive Names**: Name custom tasks clearly (e.g., `rspec-fast`, `integration-tests`)
3. **Test Mappings**: Use `plur watch find <file>` to test your mappings
4. **Leverage Glob Patterns**: Use standard glob patterns for maximum flexibility
5. **Document Complex Tasks**: Use the `description` field to explain what custom tasks do

## Next Steps

* See [Usage](usage.md) for command examples
* See [Development](development/index.md) for contributing