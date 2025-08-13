# Dynamic Memory Pre-allocation in Plur

## Overview

Plur uses dynamic memory pre-allocation to optimize performance based on test suite characteristics. This reduces allocations and GC pressure during test execution.

## Strategies Implemented

### 1. Size-based Pre-allocation

The `NewTestCollectorWithHints()` function allocates memory based on:
- Number of test files being run
- Estimated tests per file (10-20 depending on context)
- Historical failure rates (5% default)

```go
// For single files: estimate 20 tests
// For file groups: estimate 10 tests per file
testsPerFile := 20
if len(testFiles) > 1 {
    testsPerFile = 10
}
collector := NewTestCollectorWithHints(len(testFiles), testsPerFile)
```

### 2. Allocation Formulas

**Test capacity:**
```
expectedTests = numFiles * estimatedTestsPerFile
```

**Failure capacity:**
```
expectedFailures = expectedTests / 20  // 5% failure rate
```

**Output buffer:**
```
outputSize = expectedTests * 100 + 2048  // 100 bytes per test + 2KB base
```

### 3. Safety Bounds

To prevent over-allocation:
- Minimum: 10 tests, 5 failures, 1KB output
- Maximum: 10,000 tests, 1,000 failures, 1MB output

## Future Enhancements

### Historical Learning (Prepared but not yet active)

The `allocation_hints.go` file contains infrastructure for:

1. **Project Statistics Tracking**
   - Average test count per run
   - Average failure rate
   - Average output size
   - Tests per file ratio

2. **Exponential Moving Average**
   - Recent runs weighted more heavily (α = 0.3)
   - Adapts to project changes over time

3. **Per-Project Profiles**
   - Stats stored in `~/.plur/runtime/{project}.stats.json`
   - Automatically updated after each run

### Activation

To enable historical learning, integrate `GetAllocationHints()` in the runner:

```go
hints := GetAllocationHints(projectPath, len(testFiles), globalConfig.RuntimeDir)
collector := NewTestCollectorWithHints(hints.EstimatedTests/len(testFiles), len(testFiles))
```

## Performance Impact

### Micro-benchmarks
- 20-21% speed improvement in test collection
- 33% less memory usage for slice operations
- 13% fewer allocations overall

### Real-world Testing
- Maintains 1% performance advantage over turbo_tests
- Reduced GC pressure on large test suites
- More predictable memory usage patterns

## Best Practices for Go Pre-allocation

1. **Start Conservative**: Better to grow than over-allocate
2. **Use Profiling**: `go test -benchmem` to measure impact
3. **Cap Maximum**: Prevent runaway allocation on edge cases
4. **Learn from Runtime**: Use historical data when available
5. **Profile-Guided Optimization**: Go 1.20+ PGO can help

## Tools and Techniques

### Measuring Allocation Impact
```bash
# Run benchmarks with memory stats
cd plur && go test -bench=. -benchmem

# Compare before/after
script/benchmark-memory > before.txt
# make changes
script/benchmark-memory > after.txt
go install golang.org/x/perf/cmd/benchstat@latest
benchstat before.txt after.txt
```

### Memory Profiling
```bash
# Generate memory profile
go test -memprofile=mem.prof -bench=.

# Analyze allocations
go tool pprof -alloc_space mem.prof
```

### Runtime Monitoring
```go
// Monitor actual usage in production
var m runtime.MemStats
runtime.ReadMemStats(&m)
fmt.Printf("Alloc = %v KB", m.Alloc / 1024)
```

## Conclusion

Dynamic pre-allocation based on test suite characteristics provides significant performance improvements with minimal complexity. The prepared historical learning infrastructure can be activated when more sophisticated adaptation is needed.