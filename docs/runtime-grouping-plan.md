# Runtime-Based Grouping Implementation Plan

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