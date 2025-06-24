# Bundler Optimization Implementation Guide

## Current Implementation

In `rux/runner.go:147`, rux always uses bundle exec:
```go
args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
```

## Proposed Implementation

### Phase 1: Detection Logic

Add bundler detection in `runner.go`:

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

### Phase 2: Configuration Support

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

### Phase 3: Update RunSpecFile

Modify `runner.go:147`:
```go
var args []string
if r.shouldUseBundler() {
    args = []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
} else {
    args = []string{"rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
}
```

### Phase 4: CLI Flag

Add to CLI options:
```go
var noBundler bool
flag.BoolVar(&noBundler, "no-bundler", false, "Skip bundle exec when running tests")
```

## Auto-Detection Algorithm

```
1. If --no-bundler flag: don't use bundler
2. If config.bundler.enabled is set: use that value
3. Otherwise auto-detect:
   - No Gemfile? → don't use bundler
   - Gemfile exists but no bundler/setup in specs? → try without bundler
   - Any bundler references? → use bundler
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