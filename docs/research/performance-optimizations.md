# Performance Optimizations

**Last Updated:** October 2025
**Status:** Active tracking of performance improvements and opportunities

This document tracks performance optimizations for Plur, combining completed work with actionable opportunities.

---

## ✅ Completed Optimizations

These optimizations have been successfully implemented:

- [x] **String builder pre-allocation** - `stream_helper.go:31` pre-allocates 8KB for stderr
- [x] **Slice capacity hints in TestCollector** - `test_collector.go:25-28` pre-allocates for ~100 tests, ~10 failures
- [x] **Scanner buffer increase** - `stream_helper.go:17` increased to 256KB (from 64KB default)
- [x] **JSON streaming removed** - No longer using `io.ReadAll`; already using streaming parsers
- [x] **Channel buffer sizing** - `runner.go:317` uses `maxWorkers*10` for output channel

---

## 🎯 High-Priority Optimizations

These optimizations are validated, high-impact, and ready to implement.

### 1. File Content Caching for Failures
**Location:** `rspec/json_output.go:132-173` (ExtractFailingLine)
**Impact:** HIGH
**Effort:** Medium (1-2 days)

**Issue:**
Opens and scans files repeatedly for each failure:
```go
file, err := os.Open(filePath)  // Called once per failure
```

**Problem:**
* N+1 file operations - 10 failures in 3 files = 10 file opens
* Full file scan from start each time
* No caching between failures in the same file

**Solution:**
```go
type FileLineCache struct {
    cache map[string][]string
    mu    sync.RWMutex
}

func (c *FileLineCache) GetLine(filePath string, lineNum int) (string, error) {
    // Check cache first
    // Read file once if not cached
    // Return specific line
}
```

**Expected Impact:**
* Reduces file I/O by ~70% in typical failure scenarios
* Eliminates redundant file scanning

---

### 2. Parallelize os.Stat() Calls
**Location:** `grouper.go:25-32` (GroupSpecFilesBySize)
**Impact:** MEDIUM-HIGH (for large projects)
**Effort:** Low (few hours)

**Issue:**
Sequential `os.Stat()` calls for all test files:
```go
for _, file := range specFiles {
    info, err := os.Stat(file)  // Sequential syscalls
    // ...
}
```

**Problem:**
* Sequential system calls
* Slow for projects with 100+ test files
* Not utilizing available parallelism

**Solution:**
Use goroutines to parallelize stat calls:
```go
type fileWithSize struct {
    path string
    size int64
}

// Parallel stat gathering
statChan := make(chan fileWithSize, len(specFiles))
var wg sync.WaitGroup
for _, file := range specFiles {
    wg.Add(1)
    go func(f string) {
        defer wg.Done()
        if info, err := os.Stat(f); err == nil {
            statChan <- fileWithSize{f, info.Size()}
        }
    }(file)
}
wg.Wait()
close(statChan)
```

**Expected Impact:**
* 2-3x faster file discovery for projects with 200+ files
* Better utilization of I/O concurrency

---

### 3. Slice Pre-allocation in Grouper
**Location:** `grouper.go:44,69,111,138`
**Impact:** MEDIUM
**Effort:** Very Low (< 30 minutes)

**Issue:**
Slices allocated without capacity hints:
```go
Files: make([]string, 0)  // Line 44, 111
nonEmptyGroups := make([]FileGroup, 0)  // Line 69, 138
```

**Solution:**
```go
// Line 44, 111 - estimate files per worker
Files: make([]string, 0, len(specFiles)/numWorkers)

// Line 69, 138 - won't exceed numWorkers
nonEmptyGroups := make([]FileGroup, 0, numWorkers)
```

**Expected Impact:**
* Reduces allocations during test grouping
* Minimal but measurable improvement in startup time

---

### 4. Runtime Map Pre-allocation
**Location:** `runtime_tracker.go:73-83` (GetRuntimes)
**Impact:** LOW-MEDIUM
**Effort:** Very Low (5 minutes)

**Issue:**
```go
result := make(map[string]float64)  // No capacity hint
for k, v := range rt.runtimes {
    result[k] = v
}
```

**Solution:**
```go
result := make(map[string]float64, len(rt.runtimes))
```

**Expected Impact:**
* Minor reduction in allocations
* Map is small, so impact is modest

---

## 🔍 Low-Priority Improvements

These are minor optimizations or cleanup opportunities.

### 1. Simplify Error Formatting in doctor.go
**Locations:** Lines 37, 44, 53, 59, 103
**Impact:** VERY LOW
**Effort:** Trivial

**Current:**
```go
pwd = fmt.Sprintf("error: %v", err)
```

**Better:**
```go
pwd = "error: " + err.Error()
```

**Note:** Not in a hot path, but unnecessary overhead.

---

### 2. Combine fmt.Sprintf Calls in Minitest Parser
**Location:** `minitest/output_parser.go:93-94`
**Impact:** VERY LOW
**Effort:** Trivial

**Current:**
```go
summary := fmt.Sprintf("\nFinished in %s.\n", format.FormatDuration(wallTime))
summary += fmt.Sprintf("%s, %s, %s, %s, %s", runText, assertionText, failureText, errorText, skipText)
```

**Better:**
```go
summary := fmt.Sprintf("\nFinished in %s.\n%s, %s, %s, %s, %s",
    format.FormatDuration(wallTime), runText, assertionText, failureText, errorText, skipText)
```

**Note:** Runs once per test suite, minimal impact.

---

## ❌ Not Worth Pursuing

These items were flagged but are either already correct or intentional design decisions.

* **strings.Join in debug paths** (`runner.go:177,183`) - Intentional for clarity; not hot paths
* **GetJSONRowsFormatterPath race condition** - No race condition exists; function is thread-safe with lazy init pattern
* **Goroutine creation overhead** - Current design is correct; goroutines are lightweight and reused via worker pool
* **Runtime map copying** - Intentional defensive programming to prevent external modification

---

## 🧪 Performance Testing

### Recommended Benchmarks

Add benchmarks for these critical paths:

```bash
# File discovery and grouping
go test -bench=BenchmarkGroupSpecFilesBySize

# Parallel execution
go test -bench=BenchmarkRunSpecsInParallel

# Output parsing
go test -bench=BenchmarkParseRSpecOutput

# Runtime tracking
go test -bench=BenchmarkRuntimeTracker
```

### Profiling Commands

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./...
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./...
go tool pprof -http=:8080 mem.prof

# Execution trace
go test -trace=trace.out ./...
go tool trace trace.out

# Race detection
go test -race ./...
```

---

## 📊 Expected Impact Summary

| Optimization | Impact | Effort | Priority |
|-------------|--------|--------|----------|
| File content caching | HIGH | Medium | 1 |
| Parallelize os.Stat | MEDIUM-HIGH | Low | 2 |
| Grouper slice pre-allocation | MEDIUM | Very Low | 3 |
| Runtime map pre-allocation | LOW-MEDIUM | Very Low | 4 |
| Doctor.go error formatting | VERY LOW | Trivial | 5 |
| Minitest fmt.Sprintf | VERY LOW | Trivial | 6 |

**Overall Expected Gains:**
* Memory: 15-25% reduction in allocations
* I/O: 60-70% reduction in redundant file operations
* Latency: 10-20% improvement for large test suites (200+ files)

---

## 🎯 Implementation Roadmap

### Phase 1: Quick Wins (< 1 day)
- [ ] Add slice capacity hints in grouper.go (4 locations)
- [ ] Pre-allocate runtime map in GetRuntimes()
- [ ] Simplify error formatting in doctor.go (5 locations)

### Phase 2: High Impact (2-3 days)
- [ ] Implement file content caching for failure extraction
- [ ] Parallelize os.Stat() calls in grouper

### Phase 3: Validation (1 week)
- [ ] Add comprehensive benchmarks
- [ ] Profile before/after changes
- [ ] Measure impact on real projects

---

## 📝 Notes

* Focus on high-impact, low-effort changes first
* All optimizations maintain thread safety
* Changes should not affect external behavior
* Profile to validate improvements before merging
