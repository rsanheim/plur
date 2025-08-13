# Plur Performance Analysis - August 2025

## Executive Summary

Performance analysis of plur's Go codebase reveals good overall patterns but identifies several optimization opportunities, particularly in memory management, I/O operations, and string handling.

## Key Findings

### 1. Memory Allocation Hotspots

#### String Builder Pre-allocation

Files: `stream_helper.go:25`, `test_collector.go:16`

* Issue: Dynamic `strings.Builder` growth causes multiple reallocations
* Impact: HIGH - GC pressure during test execution
* Fix: Pre-allocate with `builder.Grow(4096)` or estimated capacity

#### Slice Capacity Hints

File: `test_collector.go:24-27`

* Issue: Slices initialized without capacity grow dynamically
* Impact: MEDIUM - Multiple reallocations as tests are collected
* Fix: `make([]types.TestCaseNotification, 0, 100)` with estimated capacity

#### Runtime Map Copying

File: `runtime_tracker.go:77-80`

* Issue: Full map copy on every `GetRuntimes()` call
* Impact: MEDIUM - O(n) allocation per access
* Fix: Consider read-only view or copy-on-write pattern

### 2. I/O Operation Inefficiencies

#### JSON File Loading

File: `rspec/json_output.go:65-68`

* Issue: `io.ReadAll()` loads entire JSON into memory
* Impact: HIGH - Memory scales with file size
* Fix: Use streaming decoder:

```go
json.NewDecoder(file).Decode(&output)
```

#### Repeated File Reading

File: `rspec/json_output.go:167-205` (ExtractFailingLine)

* Issue: Opens and scans file for each failure
* Impact: HIGH - O(failures × file_lines)
* Fix: Cache file contents or use memory-mapped files

#### Individual File Stats

File: `grouper.go:26-33`

* Issue: `os.Stat()` called per file, no batching
* Impact: HIGH with many files - System call overhead
* Fix: Use `filepath.Walk()` or batch operations

### 3. Concurrency Optimizations

#### Output Channel Buffering

File: `runner.go:354`

* Issue: Fixed buffer size `maxWorkers*10` may be insufficient
* Impact: HIGH - Goroutine blocking on full buffer
* Fix: Make configurable or scale with file count

#### Scanner Buffer Limits

File: `stream_helper.go:32-81`

* Issue: Default 64KB scanner buffer can fail on large output
* Impact: HIGH - Test failures on large output
* Fix:

```go
scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
```

#### Runtime Tracker Contention

File: `runtime_tracker.go:16-17`

* Issue: Single mutex for all operations
* Impact: MEDIUM - Contention with many workers
* Fix: Consider `sync.Map` or sharding

### 4. String Operation Overhead

#### Debug Logging Concatenation

Files: `runner.go:192,429`

* Issue: String building even when debug disabled
* Impact: MEDIUM - Only in debug mode
* Fix: Check debug flag before building strings

#### Failure Formatting

File: `rspec/json_output.go:117-156`

* Issue: Multiple string operations without pre-allocation
* Impact: MEDIUM - Once per failure
* Fix: Pre-allocate with `sb.Grow(512)`

## Positive Patterns Observed

1. Pre-compiled ANSI colors (`runner.go:99-104`) - Avoids repeated concatenation
2. Proper channel patterns - Clean closing and wait group usage
3. Minimal lock duration - Good defer patterns for mutex unlocking
4. Context usage - Proper timeout handling with context.Context

## Optimization Roadmap

### Phase 1: Quick Wins (1-2 days)

* [ ] Add string builder pre-allocation
* [ ] Fix JSON streaming decoder
* [ ] Increase scanner buffer sizes
* [ ] Add slice capacity hints

### Phase 2: I/O Improvements (3-5 days)

* [ ] Implement file content caching
* [ ] Batch file stat operations
* [ ] Optimize failure line extraction

### Phase 3: Concurrency Tuning (1 week)

* [ ] Configurable channel buffer sizes
* [ ] Evaluate sync.Map for runtime tracker
* [ ] Profile lock contention

### Phase 4: Advanced Optimizations (2+ weeks)

* [ ] Memory-mapped files for repeated access
* [ ] Object pools for frequent allocations
* [ ] Adaptive buffer sizing
* [ ] Comprehensive benchmarking suite

## Benchmarking Recommendations

Create benchmarks for:

1. `ExpandGlobPatterns` with large file sets
2. `RunSpecsInParallel` with varying worker counts
3. `ExtractFailingLine` with large files
4. Runtime tracker concurrent access

## Memory Profile Analysis

Recommended pprof points:

* Before/after test execution
* During peak worker activity
* After failure formatting

## Conclusion

Plur demonstrates solid Go performance patterns but has clear optimization opportunities. The suggested improvements focus on:

1. Reducing allocations through pre-sizing
2. Minimizing I/O through caching
3. Optimizing hot paths in test execution

Implementation priority should focus on high-impact, low-effort changes first, particularly string builder pre-allocation and JSON streaming, which will provide immediate benefits for large test suites.

## Appendix: Performance Testing Commands

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace analysis
go test -trace=trace.out -bench=.
go tool trace trace.out

# Race detection
go test -race ./...
```