# Bundler Optimization Research

## Overview

This document consolidates research on bundler overhead in Plur and potential optimization strategies. The research originated from performance comparisons with turbo_tests on the example-project repository.

## The Problem

We discovered that turbo_tests shows significantly better performance than rux on the example-project repository due to a key difference in how tests are executed:

- **turbo_tests**: Runs `rspec` directly
- **plur**: Runs `bundle exec rspec`

The example-project project is unique in that it **does not require bundler** to run tests, making the bundler overhead particularly noticeable.

## Performance Impact

The bundler overhead includes:
1. Loading bundler itself (~100-200ms)
2. Resolving dependencies
3. Setting up the bundle environment
4. Additional Ruby require overhead

For projects that don't need bundler (like example-project), this is pure overhead multiplied by the number of workers.

## Current Implementation

In `rux/runner.go:147`, rux always uses bundle exec:
```go
args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Plur::JsonRowsFormatter"}
```

## Evidence

From turbo_tests verbose output:
```bash
Process 1: ... rspec --format TurboTests::JsonRowsFormatter ...
```

Notice: No `bundle exec` prefix - just direct `rspec` execution.

## Proposed Solutions

### 1. Auto-detect Bundler Requirement

Add bundler detection logic:

```go
func requiresBundler(projectDir string) bool {
    // Check 1: Gemfile exists
    gemfilePath := filepath.Join(projectDir, "Gemfile")
    if _, err := os.Stat(gemfilePath); err == nil {
        // Check 2: Does spec_helper require bundler?
        specHelper := filepath.Join(projectDir, "spec", "spec_helper.rb")
        if content, err := os.ReadFile(specHelper); err == nil {
            if bytes.Contains(content, []byte("bundler/setup")) {
                return true
            }
        }
        // Check 3: Does .rspec require bundler?
        rspecFile := filepath.Join(projectDir, ".rspec")
        if content, err := os.ReadFile(rspecFile); err == nil {
            if bytes.Contains(content, []byte("--require bundler/setup")) {
                return true
            }
        }
    }
    return false
}
```

### 2. Configuration Option

```yaml
# .plur.yml
bundler:
  enabled: false  # Skip bundle exec
```

Add to `config.go`:
```go
type Config struct {
    // existing fields...
    Bundler BundlerConfig `yaml:"bundler"`
}

type BundlerConfig struct {
    Enabled *bool `yaml:"enabled"` // nil means auto-detect
}
```

### 3. Command-line Flag

```bash
plur --no-bundler  # Skip bundle exec
```

### 4. Smart Detection via Dry Run

- Run a quick test without bundle exec
- If it fails with gem loading errors, fall back to bundle exec
- Cache the result for subsequent runs

### 5. Follow RSpec's Own Detection

- RSpec itself can detect if it needs bundler
- We could leverage similar logic

## Auto-Detection Algorithm

```
1. If --no-bundler flag: don't use bundler
2. If config.bundler.enabled is set: use that value
3. Otherwise auto-detect:
   - No Gemfile? → don't use bundler
   - Gemfile exists but no bundler/setup in specs? → try without bundler
   - Any bundler references? → use bundler
```

## Implementation Strategy

### Update RunSpecFile

Modify `runner.go:147`:
```go
var args []string
if r.shouldUseBundler() {
    args = []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Plur::JsonRowsFormatter"}
} else {
    args = []string{"rspec", "-r", formatterPath, "--format", "Plur::JsonRowsFormatter"}
}
```

## Migration Path

1. **v1**: Add detection logic, off by default (maintain current behavior)
2. **v2**: Enable auto-detection by default with --force-bundler flag
3. **v3**: Remove compatibility flags after user feedback

## Testing Strategy

Test matrix:
- Rails app (requires bundler)
- Plain Ruby project with Gemfile (may not require bundler)
- example-project-style project (no bundler needed)
- Project with bundler in Gemfile but not required for specs

## Recommendation

The best approach would be a combination:
1. Smart auto-detection as the default
2. Configuration file override for explicit control
3. Command-line flag for one-off runs

This would maintain compatibility while optimizing for projects that don't need bundler overhead.