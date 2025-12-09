# Performance Investigation: Plur vs turbo_tests

## Status: COMPLETE

## Summary

Plur was 16% slower than turbo_tests on the example-project suite. Root cause: example-project's `.plur.toml` used `bin/rspec` binstub which adds 6.6s CPU overhead from `require "bundler/setup"`. After fix, plur is within 0.2% of turbo_tests.

## Issues Fixed

### 1. Runtime Path Mismatch

**Cause:** RSpec outputs paths with `./` prefix, glob discovery returns paths without.

**Impact:** 100% cache miss on runtime lookups, causing random file distribution instead of optimized.

**Fix:** Strip `./` prefix in `plur/rspec/parser.go`

**Result:** 100% hit rate on runtime data lookups.

### 2. bin/rspec Binstub Overhead

**Cause:** example-project's `.plur.toml` configured `cmd = ["bin/rspec"]`. The binstub does `require "bundler/setup"` which adds 6.6s CPU overhead per process.

**Why turbo_tests wasn't affected:** turbo_tests uses `rspec` (gem command), not the binstub.

**Fix:** Changed example-project's `.plur.toml` to `cmd = ["rspec"]`

**Result:** plur now within 0.2% of turbo_tests on example-project.

## Benchmark Results

### Before Fix
```
turbo_tests -n 3: 17.1s
plur -n 3:        19.7s (16% slower)
```

### After Fix
```
turbo_tests -n 3: 17.1s
plur -n 3:        17.2s (0.2% - equivalent)
```

### Additional Finding: jsonv2 Not Beneficial

Tested Go 1.25's experimental jsonv2 on rubocop (29K tests):
- No measurable improvement (~0.3%, within noise)
- JSON parsing is only 60ms out of 41s total runtime (0.15%)
- Not worth pursuing for typical use cases

## Changes Made

1. `plur/rspec/parser.go` - Strip `./` prefix when parsing RSpec JSON
2. `plur/runtime_tracker.go` - Removed unnecessary mutex (single-threaded usage)
3. `plur/grouper.go` - Added hit/miss debug logging for runtime data
4. `plur/grouper_test.go` - New unit tests for runtime-based grouping
5. `references/example-project/.plur.toml` - Use `rspec` instead of `bin/rspec`
6. Updated integration tests for path normalization
