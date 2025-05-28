# Runtime-Based Grouping Implementation Plan

## Status: ✅ COMPLETED

Runtime-based grouping has been successfully implemented and is now the default behavior when runtime data is available.

## Problem Statement

The failing `efficiently handles many spec files` test shows that file-size-based grouping isn't optimal for larger test suites. Some files may be small but contain slow tests, while others may be large but run quickly.

## Solution: Runtime-Based Grouping

Implement runtime tracking similar to parallel_tests to distribute work based on actual execution time.

## Implementation Steps

### 1. Runtime Tracking During Execution
- Modify `RunSpecFile` to track actual runtime per file
- Store results in memory during execution
- Include both wall time and actual test time

### 2. Runtime Storage
```go
// ~/.cache/rux/runtimes/<project-hash>.log
// Format: relative/path/to/spec.rb:seconds
spec/models/user_spec.rb:2.341
spec/controllers/auth_spec.rb:5.123
spec/helpers/date_helper_spec.rb:0.234
```

### 3. Project Identification
- Use hash of project directory path for runtime file name
- Allows multiple projects to have separate runtime data
- Example: `~/.cache/rux/runtimes/a1b2c3d4.log`

### 4. Runtime-Based Grouping Algorithm
```go
func GroupSpecFilesByRuntime(specFiles []string, numWorkers int, runtimeData map[string]float64) []FileGroup {
    // Sort files by runtime (slowest first)
    // Distribute using "smallest group first" algorithm
    // Files without runtime data use file size as estimate
}
```

### 5. Integration Points
- Load runtime data at startup if available
- Fall back to file-size grouping if no data
- Update runtime data after each run
- Handle file renames/deletions gracefully

### 6. Benefits
- Better work distribution for heterogeneous test suites
- Adapts automatically as tests change
- Reduces total execution time by balancing slow tests
- Improves scalability for large test suites

## Expected Impact

For the failing test with many files:
- Current: 89.6% efficiency (file-size based)
- Expected: >95% efficiency (runtime based)

This should make rux competitive with turbo_tests even for large, complex test suites.

## Implementation Details (Completed)

### What Was Built

1. **Runtime Tracking**:
   - `RuntimeTracker` struct collects test execution times from RSpec JSON output
   - Accumulates runtime data for each spec file during execution
   - Saves data to `~/.cache/rux/runtime.json` after each run

2. **Runtime-Based Grouping**:
   - `GroupSpecFilesByRuntime` function distributes files based on historical runtime
   - Uses "smallest runtime first" algorithm for optimal load balancing
   - Falls back to size-based grouping when no runtime data exists

3. **Automatic Detection**:
   - Rux automatically loads runtime data if available
   - Prints informative messages about grouping strategy used
   - Seamlessly switches between runtime and size-based grouping

4. **Testing**:
   - Comprehensive integration tests verify runtime tracking behavior
   - Tests cover data collection, persistence, and grouping algorithms
   - Ensures proper fallback when runtime data is unavailable

### FOLLOW UP

* change the filename to be per-project -- I think using a hash of the project directory is a good idea
* consolidate where we build the runtime file path - we have it in two places
* update any related specs
* lint, run build, etc.