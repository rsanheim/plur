# Plur Performance Optimization Plan

## Current Performance Gap

For the plur-ruby test suite (11 spec files, ~450ms total):
- **turbo_tests**: 455ms (baseline)
- **plur -n 4**: 535ms (18% slower)

## Root Cause Analysis

### 1. Process Spawn Strategy Difference
**turbo_tests**: Groups multiple spec files per process
- 11 spec files → 11 processes (current behavior)
- Could be fewer with better grouping

**plur**: One process per spec file
- 11 spec files → 11 processes always
- More process spawn overhead
- More Ruby initialization overhead

### 2. Overhead Breakdown (from analysis)
- Process spawn: ~1.3ms per process (negligible)
- Ruby startup: ~31ms 
- RSpec load: ~45ms
- Per-spec execution: ~175ms average
- **Key insight**: 65.6% "plur overhead" - most time is NOT in actual test execution

### 3. Architecture Differences

**turbo_tests advantages:**
- Uses `ParallelTests` grouping logic (by runtime or filesize)
- Single formatter loaded once in parent process
- Threads handle I/O from child processes
- Message queue pattern for result aggregation

**plur current issues:**
- Worker pool pattern may add coordination overhead
- 2 goroutines created per spec file
- No test grouping - always one file per process
- More abstraction layers

## Optimization Strategy

### Option 1: Quick Win - Batch Small Files (Recommended First Step)
Implement file batching for small test suites to reduce process overhead:

```go
// Instead of one process per file, batch files together
// Target: ~100-200ms worth of tests per process
func BatchSpecFiles(files []string, targetBatchDuration time.Duration) [][]string {
    // Use file size or historical runtime to estimate
    // Group files to minimize number of processes
}
```

Benefits:
- Reduces process spawn count
- Amortizes Ruby/RSpec startup cost
- Should immediately close gap with turbo_tests

### Option 2: Goroutine Pool
Replace per-file goroutine creation with a pool:

```go
type IOHandler struct {
    pool *ants.Pool
}

// Reuse goroutines instead of creating 2 per file
```

### Option 3: Direct popen Implementation
Use lower-level process spawning like turbo_tests:

```go
// Replace exec.Command with direct syscall
// Reduces abstraction overhead
```

### Option 4: Smart Test Distribution
Port ParallelTests grouping logic:
- Group by file size for first run
- Use runtime data for subsequent runs
- Balance work across workers better

## Implementation Plan

### Phase 1: File Batching (Immediate Impact)
1. Add `--batch-size` flag (default: auto-detect based on file count)
2. Implement batching logic that groups multiple spec files
3. Modify `RunSpecFile` to accept multiple files
4. Pass array of files to RSpec command

Expected impact: **10-15% performance improvement** for small suites

### Phase 2: Reduce Goroutine Overhead
1. Implement goroutine pool (e.g., using ants library)
2. Pre-allocate output buffers
3. Reduce channel buffer allocations

Expected impact: **2-5% improvement**

### Phase 3: Smart Distribution
1. Implement runtime tracking
2. Add filesize-based grouping
3. Balance work across workers

Expected impact: **5-10% improvement** for uneven test suites

## Validation Metrics

For plur-ruby benchmark:
- Target: ≤ 470ms (within 5% of turbo_tests)
- Stretch goal: ≤ 455ms (match turbo_tests)

For larger suites:
- Maintain current advantages at scale
- Test with 100+, 1000+ spec files

## Quick Experiment

Before implementing, we can validate the batching hypothesis:

```bash
# Test running multiple files in one RSpec process
cd plur-ruby
time bundle exec rspec spec/calculator_spec.rb spec/counter_spec.rb spec/validator_spec.rb

# vs individual runs
time (bundle exec rspec spec/calculator_spec.rb; bundle exec rspec spec/counter_spec.rb; bundle exec rspec spec/validator_spec.rb)
```

This will show if batching provides the expected benefit.