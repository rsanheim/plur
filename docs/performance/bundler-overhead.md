# Bundler Overhead in Rux vs Turbo Tests

## The Issue

We discovered that turbo_tests shows significantly better performance than rux on the example-project repository due to a key difference in how tests are executed:

- **turbo_tests**: Runs `rspec` directly
- **rux**: Runs `bundle exec rspec`

The example-project project is unique in that it **does not require bundler** to run tests, making the bundler overhead particularly noticeable.

## Evidence

From turbo_tests verbose output:
```bash
Process 1: ... rspec --format TurboTests::JsonRowsFormatter ...
```

Notice: No `bundle exec` prefix - just direct `rspec` execution.

## Performance Impact

The bundler overhead includes:
1. Loading bundler itself (~100-200ms)
2. Resolving dependencies
3. Setting up the bundle environment
4. Additional Ruby require overhead

For projects that don't need bundler (like example-project), this is pure overhead multiplied by the number of workers.

## Potential Solutions

### 1. Auto-detect Bundler Requirement
- Check for `Gemfile` presence
- Analyze spec_helper.rb for `require 'bundler/setup'`
- Skip bundle exec if not needed

### 2. Configuration Option
```yaml
# .rux.yml
bundler:
  enabled: false  # Skip bundle exec
```

### 3. Smart Detection via Dry Run
- Run a quick test without bundle exec
- If it fails with gem loading errors, fall back to bundle exec
- Cache the result for subsequent runs

### 4. Follow RSpec's Own Detection
- RSpec itself can detect if it needs bundler
- We could leverage similar logic

### 5. Explicit Flag
```bash
rux --no-bundler  # Skip bundle exec
```

## Recommendation

The best approach would be a combination:
1. Smart auto-detection as the default
2. Configuration file override for explicit control
3. Command-line flag for one-off runs

This would maintain compatibility while optimizing for projects that don't need bundler overhead.