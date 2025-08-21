# Task Configuration

Tasks are the core of Plur's test execution system. They define how to run tests, linters, or other commands, and how to map source files to test files.

## Overview

A Task in Plur encapsulates:
* The command to run
* Which directories to watch or search
* How to map source files to test files
* File patterns to match and ignore

Plur comes with built-in tasks for RSpec and Minitest, but you can define custom tasks for any tool.

## Task Selection Priority

Tasks are selected in the following priority order:
1. CLI flag: `plur --use=custom-task`
2. Config file: `use = "custom-task"` in `.plur.toml`
3. Auto-detection: Based on directory structure (spec/ → rspec, test/ → minitest)

## Task Configuration Fields

| Field | Type | Description | Required | Default |
|-------|------|-------------|----------|---------|
| `description` | string | Human-readable description of the task | No | "" |
| `run` | string | Command to execute | Yes | "" |
| `source_dirs` | string[] | Directories to watch/search | No | `["."]` |
| `mappings` | MappingRule[] | File mapping rules | No | `[]` |
| `ignore_patterns` | string[] | Patterns to ignore (watch mode) | No | `[".git"]` |
| `test_glob` | string | Glob pattern for test files | No | Depends on task |

## Built-in Tasks

### RSpec (default)
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

### Minitest
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

## Custom Task Examples

### Custom RSpec with Spring
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

### Linter Task
```toml
[task.rubocop]
description = "Run RuboCop linter"
run = "bundle exec rubocop"
source_dirs = ["lib", "spec", "app"]
test_glob = "**/*.rb"
# No mappings needed - rubocop will lint the files directly
```

### JavaScript Test Runner
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

### Go Tests
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
ignore_patterns = ["vendor", ".git"]
```

## Mapping Patterns

Mappings define how source files map to test files. They use a pattern matching system with tokens:

### Available Tokens
* `{{file}}` - The complete matched file path
* `{{path}}` - The directory path of the matched file (without filename)
* `{{name}}` - The base filename without extension

### Examples

Given a file `lib/models/user.rb`:
* `{{file}}` → `lib/models/user.rb`
* `{{path}}` → `lib/models`
* `{{name}}` → `user`

So the mapping:
```toml
{ pattern = "lib/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" }
```
Would map `lib/models/user.rb` → `spec/lib/models/user_spec.rb`

## Multiple Task Definitions

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

## Watch Mode Integration

Tasks work seamlessly with watch mode. The `source_dirs` determine which directories are watched, and `mappings` determine which tests run when files change:

```bash
plur watch --use=custom-task
```

When a file changes:
1. Plur checks if it matches any mapping patterns
2. If matched, runs the corresponding target test files
3. If no match, runs the changed file directly (if it matches `test_glob`)

## Tips and Best Practices

1. **Start Simple**: Begin with just overriding the `run` command for existing tasks
2. **Use Descriptive Names**: Name custom tasks clearly (e.g., `rspec-fast`, `integration-tests`)
3. **Test Mappings**: Use `plur watch find <file>` to test your mappings
4. **Leverage Glob Patterns**: Use standard glob patterns for maximum flexibility
5. **Document Complex Tasks**: Use the `description` field to explain what custom tasks do

## Troubleshooting

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