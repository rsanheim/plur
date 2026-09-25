# Configuration

Plur provides defaults for common projects. Use TOML files, environment variables,
or command-line flags to change them.

## Configuration Methods

Settings take precedence in this order, from highest to lowest:

1. Command-line flags
2. Environment variables (e.g., `PLUR_WORKERS`, `PLUR_DEBUG`)
3. Configuration files (merged; later files override earlier values)
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
color = "auto"           # colorize when output is a terminal (the default)
formatter = "auto"       # progress markers on a terminal, summary output otherwise (the default)
use = "rspec"  # Default job to use

[job.rspec]
cmd = ["bin/rspec"]  # Override default command

[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest"]
```

### Available Options

#### Global Settings

* `workers` - Number of parallel workers (default: 4)
* `color` - When to colorize output: `"auto"` (default — on for a terminal, off when piped), `"always"`, or `"never"`
* `formatter` - How to render the run: `"auto"` (default — progress markers on a terminal, summary output when piped), `"progress"`, or `"summary"`. See [Output Formats](usage.md#output-formats)
* `verbose` - Enable verbose output (default: false)
* `use` - Default job to use (default: auto-detect based on project structure)

## Job Configuration

A job defines a command and the files it runs against. Plur includes jobs for
RSpec, Minitest, Go tests, Rails, and Rake. You can also define your own.

### Job Selection Priority

Jobs are selected in the following priority order:

1. CLI flag: `plur --use=custom-job`
2. Config file: `use = "custom-job"` in `.plur.toml`
3. Explicit paths: Infer the framework from the supplied files, directories, or globs
4. Auto-detection: Select the first job with matching files, in order: `rspec`, `minitest`, `go-test`

Select custom jobs with `--use` or `use`. To run files from multiple frameworks,
choose a job explicitly or run each framework separately.

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
| `cmd` | string[] | Command array to execute | For custom jobs | Inherited for built-in jobs (`rspec`, `minitest`, `go-test`, `rails`, `rake`) |
| `framework` | string | Framework identity (`rspec`, `minitest`, `go-test`, `passthrough`) | No | Built-in framework for canonical jobs, otherwise `passthrough` |
| `target_pattern` | string | Glob pattern for test files | No | Built-in default for canonical jobs; for custom jobs with a framework uses framework detect patterns; passthrough jobs default to empty |
| `exclude_patterns` | string[] | Glob patterns to exclude from discovered test files | No | `[]` |
| `env` | string[] | Environment variables (e.g., `["VAR=value"]`) | No | `[]` |

For `plur` and `plur spec`, set `cmd` to the executable and its fixed flags.
Plur adds the discovered files to the command. For Minitest, it loads them
through a Ruby `-e` script. Job commands must not contain the
`{{target}}` placeholder; target templates are only supported in watch target
mappings.

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

### Exclude Patterns

Use `exclude_patterns` to skip files that match any listed pattern. Patterns
support `**` to match across directories; unmatched patterns have no effect.

```toml
[job.rspec]
exclude_patterns = ["spec/system/**/*_spec.rb"]
```

Each `--exclude-pattern` flag adds to the configured exclusions. Given the config above, this command excludes both
`spec/system/**` and `spec/legacy/**`:

```bash
plur --exclude-pattern 'spec/legacy/**/*_spec.rb'
```

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

#### Rails And Rake
```toml
[job.rails]
cmd = ["bin/rails"]
framework = "passthrough"

[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "passthrough"
```

`plur rails <args>` and `plur rake <args>` run the configured command once per worker. Arguments are appended literally — Plur does not discover files or parse test output for these commands. Put Plur flags like `-n` before `--`; arguments after `--` are passed through to Rails/Rake.

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
| `name` | string | Optional identifier for the rule. If set, it must be unique across user-defined `[[watch]]` entries. A named user watch can override a built-in watch with the same name. | No |
| `source` | string | Glob pattern for files to watch | Yes |
| `targets` | string[] | Target patterns with placeholders. If omitted, the changed source file is used as the target. | No |
| `no_targets` | bool | Run matching jobs without appending any target args. Must not be combined with `targets`. | No |
| `jobs` | string[] | Jobs to trigger when source matches | Yes |
| `ignore` | string[] | Patterns to ignore from watching | No |

**Note**: `ignore` is per-watch mapping. For global ignore patterns, use `watch-ignore` in `.plur.toml` or the `plur watch --ignore` flag for one session.

**Note**: Named `[[watch]]` entries must be unique within user configuration. Plur rejects duplicate names during config loading.

### Placeholder Variables

* `{{match}}` - The matched portion of the source path (e.g., `lib/foo.rb` → `foo`)
* `{{dir_relative}}` - The relative directory of the matched file

Watch mode resolves target templates first, then appends those targets to the
job command. If `targets` is omitted, plur passes the changed source file. Use
`no_targets = true` for jobs that should run without file arguments.

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

# A job for the 'no-targets' use case below
[job.build]
cmd = ["script/build"]

# A watch to call `script/build` on any change with no target args
[[watch]]
source = "**/*.go"
jobs = ["build"]
no_targets = true

# Go: source files trigger package tests
[[watch]]
name = "go-source"
source = "**/*.go"
targets = ["./{{dir_relative}}"]
jobs = ["go-test"]
ignore = ["vendor/**", "**/testdata/**"]
```

### Using Watch Mode

```bash
plur watch                    # Watch with auto-detected job
plur watch --use=custom-job   # Watch with specific job
```

## Worker Configuration

Plur balances work using saved test runtimes, falling back to file sizes when
no timings are available.

Note: Watch mode (`plur watch`) can run independent job targets concurrently.
It skips a target that is already running in the same job.

### Specifying Number of Workers

```bash
# Default: 4 workers
plur

# specify number of workers
plur -n 8
plur --workers 8

# or via environment variable
export PLUR_WORKERS=8
plur
```

## Output Configuration

### Formatters

Set `formatter = "auto"`, `"progress"`, or `"summary"` to control progress
markers during test runs. The default, `auto`, shows markers on a terminal
and omits them when stdout is piped or redirected. Output from tests and
final results still print. See [Output Formats](usage.md#output-formats).

### Verbosity

```bash
# Debug output
export PLUR_DEBUG=1
plur

```

## File Discovery

### Glob Pattern Support

Use glob patterns to select test files:

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
plur spec/spec_helper.rb           # Runs as an explicit file
```

### RSpec Compatibility

Plur matches RSpec's behavior:

* **Directories**: Automatically append `**/*_spec.rb` pattern
* **Single files**: Pass through even if not matching test suffix
* **Glob patterns**: Expand matching files directly

## Environment Variables

### Recognized Variables

* `PLUR_WORKERS` - Number of workers
* `PARALLEL_TEST_PROCESSORS` - Number of workers (legacy fallback for `PLUR_WORKERS`; parallel_tests compatibility)
* `PLUR_DEBUG` - Enable debug output
* `PLUR_CONFIG_FILE` - Load an additional config file after `~/.plur.toml` and `.plur.toml`
* `PLUR_HOME` - Override Plur's home directory (default: `~/.plur`)
* `PLUR_COLOR` - Color mode from the environment: `auto`, `always`, or `never` (same values as `--color`; `true`/`false` aliases accepted)
* `NO_COLOR` - Disable colored output when set to any value ([no-color.org](https://no-color.org))
* `PLUR_FORMATTER` - Formatter from the environment: `auto`, `progress`, or `summary` (same values as `--formatter`)

Precedence: `--color` flag > `PLUR_COLOR` > `NO_COLOR` > config file > terminal detection. `NO_COLOR` and terminal detection decide only when the mode resolves to `auto`. `plur doctor` shows the resolved color decision and its source.

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

## Next Steps

* See [Usage](usage.md) for command examples
* See [Development](development/index.md) for contributing
