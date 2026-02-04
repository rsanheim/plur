Plur aims for zero-configuration operation, but provides flexible configuration options through TOML files, environment variables, and command-line flags.

## Configuration Methods

Plur supports multiple configuration methods with the following precedence (highest to lowest):

1. Command-line flags
2. Configuration files (merged; later files override earlier values)
3. Environment variables (e.g., `PARALLEL_TEST_PROCESSORS`, `PLUR_DEBUG`)
4. Built-in defaults

## Configuration Files (TOML)

Plur automatically loads configuration from TOML files using the following order
(later files override earlier values):

1. `~/.plur.toml` in your home directory (user-specific)
2. `.plur.toml` in the current directory (project-specific)
3. `PLUR_CONFIG_FILE` (if set)

### Basic Example

```toml
# .plur.toml
workers = 4
color = true
use = "rspec"  # Default job to use

[job.rspec]
cmd = ["bin/rspec"]  # Override default command

[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest"]
```

### Available Options

#### Global Settings

* `workers` - Number of parallel workers (default: auto-detect)
* `color` - Enable colored output (default: true)
* `verbose` - Enable verbose output (default: false)
* `use` - Default job to use (default: auto-detect based on project structure)

## Job Configuration

Jobs are the core of Plur's test execution system. They define how to run tests, linters, or other commands.

### Job Overview

A Job in Plur encapsulates:

* The command to run (as an array)
* File patterns to match for test discovery

Plur comes with built-in jobs for RSpec, Minitest, and Go tests, but you can define custom jobs for any tool.

### Job Selection Priority

Jobs are selected in the following priority order:

1. CLI flag: `plur --use=custom-job`
2. Config file: `use = "custom-job"` in `.plur.toml`
3. Auto-detection: Based on directory structure (spec/ → rspec, test/ → minitest)

> **Tip for Projects with Multiple Frameworks**
>
> If your project has both `spec/` and `test/` directories, plur will default to RSpec.
> Use the `--use` flag or config file setting to select a different framework:
>
> ```bash
> plur                    # Runs RSpec tests (default)
> plur --use=minitest     # Run Minitest tests instead
> plur --use=rspec        # Explicitly run RSpec tests
> ```
>
> Or add to `.plur.toml`:
> ```toml
> use = "minitest"  # Override default to use Minitest
> ```

### Job Configuration Fields

| Field | Type | Description | Required | Default |
|-------|------|-------------|----------|---------|
| `cmd` | string[] | Command array to execute | Yes | Built-in default for canonical jobs (`rspec`, `minitest`, `go-test`) |
| `framework` | string | Framework identity (`rspec`, `minitest`, `go-test`, `passthrough`) | No | Built-in framework for canonical jobs, otherwise `passthrough` |
| `target_pattern` | string | Glob pattern for test files | No | Built-in default for canonical jobs; for custom jobs with a framework uses framework detect patterns; passthrough jobs default to empty |
| `env` | string[] | Environment variables (e.g., `["VAR=value"]`) | No | `[]` |

**Note**: In run mode (`plur` / `plur spec`), any `{{target}}` tokens in `cmd` are ignored and targets are always appended (or expanded into Minitest `-e` requires). In watch mode, `{{target}}` is honored.

### Framework Default File Patterns

When `target_pattern` is omitted:

* **Canonical jobs** (`rspec`, `minitest`, `go-test`) inherit the built-in defaults:
  * `rspec` → `spec/**/*_spec.rb`
  * `minitest` → `test/**/*_test.rb`
  * `go-test` → `**/*_test.go`
* **Custom jobs** with an explicit framework use the framework's detect patterns:
  * `rspec` → `**/*_spec.rb`
  * `minitest` → `**/*_test.rb`
  * `go-test` → `**/*_test.go`
* **Passthrough** jobs have no default pattern; set `target_pattern` or pass explicit paths.

Example:

```toml
[job.fast]
framework = "rspec"
cmd = ["bin/rspec", "--fail-fast"]
# target_pattern omitted → uses **/*_spec.rb
```

You can still override with an explicit `target_pattern`:

```toml
[job.rspec-api]
framework = "rspec"
cmd = ["bin/rspec"]
target_pattern = "spec/api/**/*_spec.rb"
```

> **Note**: Passthrough jobs (like `rubocop` or `jest`) should define `target_pattern` or be run with explicit paths.

### Built-in Jobs

These examples show the built-in defaults. Targets are appended automatically in run mode.

#### RSpec (default)
```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec"]
target_pattern = "spec/**/*_spec.rb"
```

#### Minitest
```toml
[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest"]
target_pattern = "test/**/*_test.rb"
```

#### Go Tests
```toml
[job.go-test]
cmd = ["go", "test"]
target_pattern = "**/*_test.go"
```

### Custom Job Examples

#### Custom RSpec with Spring
```toml
[job.spring-rspec]
framework = "rspec"
cmd = ["bin/spring", "rspec"]
target_pattern = "spec/**/*_spec.rb"
```

#### Linter Job
```toml
[job.rubocop]
cmd = ["bundle", "exec", "rubocop"]
target_pattern = "**/*.rb"
```

#### JavaScript Test Runner
```toml
[job.jest]
cmd = ["npm", "test", "--"]
target_pattern = "test/**/*.test.js"
```

### Multiple Job Definitions

You can define multiple jobs and switch between them:

```toml
# .plur.toml
[job.rspec]
cmd = ["bundle", "exec", "rspec"]

[job.rspec-fast]
framework = "rspec"
cmd = ["bundle", "exec", "rspec", "--fail-fast"]

[job.integration]
framework = "rspec"
cmd = ["bundle", "exec", "rspec"]
target_pattern = "spec/integration/**/*_spec.rb"
```

Use them with:
```bash
plur --use=rspec-fast
plur --use=integration
```

## Watch Configuration

Watch mode uses `[[watch]]` entries to define file-to-test mappings. When a source file changes, plur finds the matching watch rule and runs the corresponding job.

### Watch Mapping Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Optional identifier for the rule | No |
| `source` | string | Glob pattern for files to watch | Yes |
| `targets` | string[] | Target patterns with placeholders | No |
| `jobs` | string[] | Jobs to trigger when source matches | Yes |
| `ignore` | string[] | Patterns to ignore from watching | No |
| `reload` | bool | Reload plur after jobs complete | No |

**Note**: `ignore` is per-watch mapping. For global ignore patterns during a watch session, use the `plur watch --ignore` flag.

### Placeholder Variables

* `{{match}}` - The matched portion of the source path (e.g., `lib/foo.rb` → `foo`)
* `{{dir_relative}}` - The relative directory of the matched file

### Watch Configuration Examples

```toml
# Ruby: lib files trigger corresponding spec files
[[watch]]
name = "lib-to-spec"
source = "lib/**/*.rb"
targets = ["spec/{{match}}_spec.rb"]
jobs = ["rspec"]

# Ruby: spec files run themselves
[[watch]]
name = "spec-files"
source = "spec/**/*_spec.rb"
jobs = ["rspec"]

# Go: source files trigger package tests
[[watch]]
name = "go-source"
source = "**/*.go"
targets = ["{{dir_relative}}"]
jobs = ["go-test"]
ignore = ["vendor/**", "**/testdata/**"]
```

### Using Watch Mode

```bash
plur watch                    # Watch with auto-detected job
plur watch --use=custom-job   # Watch with specific job
```

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

## Environment Variables

### Recognized Variables

* `PARALLEL_TEST_PROCESSORS` - Number of workers
* `PLUR_DEBUG` - Enable debug output

## Troubleshooting

### Tests Not Found

Check that your `target_pattern` matches your test files:

```bash
# List files that would be run
plur --dry-run --use=your-job
```

### Command Not Running

Ensure the first element of your `cmd` array is executable and in your PATH:

```bash
# Test the command directly
bundle exec rspec --version
```

## Tips and Best Practices

1. **Start Simple**: Begin with just overriding the `cmd` for existing jobs
2. **Use Descriptive Names**: Name custom jobs clearly (e.g., `rspec-fast`, `integration-tests`)
3. **Leverage Glob Patterns**: Use standard glob patterns for `target_pattern`

## Next Steps

* See [Usage](usage.md) for command examples
* See [Development](development/index.md) for contributing
