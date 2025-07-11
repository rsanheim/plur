# TurboTests Performance Analysis

## Key Finding

The ~48% performance advantage of turbo_tests over rux is entirely due to **Bundler overhead**. When both tools use the same command, they perform identically.

## Benchmark Results (example-project project, 4 workers)

### Default Configurations
- **turbo_tests**: 6.148s (uses bare `rspec`)
- **rux**: 9.073s (uses `bundle exec rspec`)
- **Difference**: 47.6% slower

### With Same Command (`rspec`)
- **turbo_tests**: 6.159s
- **rux**: 6.063s  
- **Difference**: 1.6% faster (within margin of error)

## Root Cause: Bundler Overhead

Single file benchmark shows Bundler adds significant overhead:
- `rspec spec/dx_spec.rb`: 155.6ms
- `bundle exec rspec spec/dx_spec.rb`: 221.5ms
- **Overhead**: 42% slower (66ms per process)

With 4 parallel workers running multiple test files, this overhead compounds dramatically.

## How TurboTests Avoids Bundler

```ruby
# From turbo_tests/lib/turbo_tests/runner.rb
if ENV["RSPEC_EXECUTABLE"]
  command_name = ENV["RSPEC_EXECUTABLE"].split
elsif ENV["BUNDLE_BIN_PATH"]
  command_name = [ENV["BUNDLE_BIN_PATH"], "exec", "rspec"]
else
  command_name = "rspec"  # <-- Default to bare rspec
end
```

TurboTests defaults to bare `rspec` unless explicitly configured otherwise, while rux defaults to `bundle exec rspec` for broader compatibility.

## Performance Implications

1. **Bundler overhead is per-process**: Each worker process pays the ~66ms startup penalty
2. **Scales with parallelism**: More workers = more Bundler overhead
3. **Affects short-running tests more**: The overhead is a larger percentage of total time

## Recommendations

### For rux users:
1. If your project supports it, configure rux to use bare `rspec`:
   ```toml
   # .rux.toml
   command = "rspec"
   ```

2. This requires:
   - RSpec gem installed globally or in system Ruby
   - No Bundler-specific gem dependencies
   - Compatible gem versions between system and Gemfile

### For projects requiring Bundler:
- The performance penalty is unavoidable
- Consider using Spring or Bootsnap to reduce Ruby startup time
- Use fewer, larger test files to amortize startup costs

## Conclusion

Rux and turbo_tests have equivalent core performance. The difference comes from their default commands:
- turbo_tests optimizes for speed with bare `rspec`
- rux optimizes for compatibility with `bundle exec rspec`

Users can configure rux to match turbo_tests performance by using `command = "rspec"` in their configuration.