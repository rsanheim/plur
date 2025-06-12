# File Mapping

## Overview

File mapping in Rux determines which test files should run when source files change. This is primarily used by the `rux watch` command to automatically run relevant tests during development.

## Current Implementation

Rux implements a simple convention-based file mapping system that follows Ruby/Rails conventions:

### Basic Mapping Rules

1. **Spec files** (`*_spec.rb`) → Run themselves
2. **Helper files** (`spec_helper.rb`, `rails_helper.rb`) → Run all specs
3. **Lib files** (`lib/**/*.rb`) → Map to `spec/**/*_spec.rb`
4. **App files** (`app/**/*.rb`) → Map to `spec/**/*_spec.rb` (Rails convention)

### Examples

```
# Direct spec file mapping
spec/models/user_spec.rb → spec/models/user_spec.rb

# Lib to spec mapping
lib/validators/email.rb → spec/validators/email_spec.rb

# Rails app to spec mapping
app/models/user.rb → spec/models/user_spec.rb
app/controllers/users_controller.rb → spec/controllers/users_controller_spec.rb

# Helper files trigger all specs
spec/spec_helper.rb → spec/**/*_spec.rb
spec/rails_helper.rb → spec/**/*_spec.rb
```

### File Types Watched

The file mapper watches these extensions:
- `.rb` - Ruby source files
- `.erb` - ERB templates (Rails views)
- `.haml` - Haml templates
- `.slim` - Slim templates

## Implementation Details

### FileMapper Structure

```go
// FileMapper handles mapping between source files and their corresponding spec files
type FileMapper struct {
    // Configuration options
}

// Core mapping function
func (fm *FileMapper) MapFileToSpecs(changedFile string) []string {
    // Returns array of spec files to run
}

// Determines if a file should trigger spec runs
func (fm *FileMapper) ShouldWatchFile(filePath string) bool {
    // Returns true if file type should be watched
}
```

### Mapping Algorithm

1. **Normalize** the file path
2. **Check** if it's already a spec file
3. **Check** if it's a helper file (triggers all specs)
4. **Apply** convention-based mapping rules
5. **Return** list of spec files to run

## Usage

### With Watch Command

```bash
# Starts watching files and runs tests automatically
rux watch

# When you edit lib/user.rb, it automatically runs:
# rux spec/user_spec.rb
```

### Testing File Mappings

Rux includes a hidden command for testing file mappings:

```bash
# Test what specs would run for given files
rux file-mapper lib/user.rb app/models/post.rb
# Output:
# lib/user.rb -> spec/user_spec.rb
# app/models/post.rb -> spec/models/post_spec.rb
```


## Advanced Patterns

### 1. Multiple Spec Files

Some files might map to multiple specs:

```
# One file can map to multiple specs (conceptual example)
app/models/user.rb → [
  "spec/models/user_spec.rb",
  "spec/requests/users_spec.rb",
  "spec/system/user_flows_spec.rb"
]
```

### 2. Regex-Based Mapping

More flexible pattern matching:

```ruby
# Conceptual example
mappings = [
  {
    pattern: /app\/services\/(.+)\.rb/,
    spec: 'spec/services/\1_spec.rb'
  },
  {
    pattern: /app\/(.+)\.rb/,
    spec: ['spec/\1_spec.rb', 'spec/requests/\1_spec.rb']
  }
]
```

### 3. Dynamic Mapping

Map based on file content or git history:

```ruby
# Conceptual: Find specs that import this file
def find_dependent_specs(changed_file)
  specs = []
  Dir.glob("spec/**/*_spec.rb").each do |spec|
    content = File.read(spec)
    if content.include?(changed_file.basename)
      specs << spec
    end
  end
  specs
end
```

## Comparison with Other Tools

### Guard

Guard uses a Guardfile with Ruby DSL:

```ruby
guard :rspec do
  watch(%r{^spec/.+_spec\.rb$})
  watch(%r{^lib/(.+)\.rb$}) { |m| "spec/#{m[1]}_spec.rb" }
  watch('spec/spec_helper.rb') { "spec" }
end
```

### Watchman

Facebook's Watchman uses trigger configurations:

```json
{
  "trigger": "rspec",
  "expression": ["anyof",
    ["match", "*.rb"],
    ["match", "*.erb"]
  ],
  "command": ["rux", "spec"]
}
```

### Entr

Entr uses shell patterns:

```bash
find . -name '*.rb' | entr -c rux spec/
```

## Best Practices

### 1. Follow Conventions

Structure your project to follow standard Ruby/Rails conventions:

```
project/
├── lib/
│   └── my_class.rb
├── spec/
│   ├── my_class_spec.rb
│   └── spec_helper.rb
└── app/  # Rails only
    └── models/
        └── user.rb
```

### 2. Explicit Spec Naming

Name spec files to match source files:

```
# Good
lib/validators/email.rb → spec/validators/email_spec.rb

# Avoid
lib/validators/email.rb → spec/email_validation_spec.rb
```

### 3. Group Related Tests

When one file affects multiple specs, consider grouping:

```
app/models/user.rb → spec/models/user/
├── validations_spec.rb
├── associations_spec.rb
└── callbacks_spec.rb
```

## Troubleshooting

### Specs Not Running

1. **Check file extension**: Ensure files have watched extensions
2. **Check path**: Verify file follows expected conventions
3. **Check spec exists**: Ensure corresponding spec file exists

### Too Many Specs Running

1. **Be specific**: Edit spec files directly when focused on one test
2. **Use focus**: Use RSpec's `focus: true` or `fdescribe`/`fit`
3. **Filter**: Pass specific files to rux directly

### Performance Issues

1. **Limit scope**: Configure ignore patterns for large directories
2. **Use debouncing**: Wait for file changes to settle
3. **Run subset**: Use RSpec tags to run only relevant specs

## See Also

- [Watch Mode Documentation](/docs/features/watch-mode.md)
- [Configuration Guide](/docs/configuration.md)
- [Architecture Overview](index.md)