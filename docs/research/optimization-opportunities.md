# Optimization Opportunities - August 2025

## String Operations

### 1. Excessive fmt.Sprintf Usage
**Impact**: MEDIUM
**Files**: Throughout codebase (40+ occurrences)

**Issues**:
- Heavy use of `fmt.Sprintf` for simple concatenations
- String formatting in hot paths
- Multiple format operations that could be combined

**Examples**:
```go
// Current (doctor.go:40)
pwd = fmt.Sprintf("error: %v", err)

// Better
pwd = "error: " + err.Error()
```

```go
// Current (minitest/output_parser.go:92-93)
summary := fmt.Sprintf("\nFinished in %.6fs.\n", wallTime)
summary += fmt.Sprintf("%s, %s, %s, %s, %s", runText, assertionText, failureText, errorText, skipText)

// Better - single format operation
summary := fmt.Sprintf("\nFinished in %.6fs.\n%s, %s, %s, %s, %s",
    wallTime, runText, assertionText, failureText, errorText, skipText)
```

### 2. strings.Join in Debug Paths
**Impact**: LOW (debug only)
**Files**: `runner.go:192,429`, `watch.go:237`

**Issue**: Building strings even when debug is disabled

**Fix**:
```go
// Check debug flag first
if logger.IsDebugEnabled() {
    logger.Debug("command", strings.Join(args, " "))
}
```

## Memory Allocations

### 1. Slice Pre-allocation Missing
**Impact**: MEDIUM
**Files**: `test_collector.go:24-26`, `grouper.go:44,69,111,138`

**Current**:
```go
tests: make([]types.TestCaseNotification, 0)
```

**Better**:
```go
tests: make([]types.TestCaseNotification, 0, 100) // Estimate capacity
```

### 2. io.ReadAll Usage
**Impact**: HIGH
**File**: `rspec/json_output.go:65`

**Issue**: Loads entire JSON file into memory

**Fix**:
```go
// Instead of
data, err := io.ReadAll(file)
json.Unmarshal(data, &output)

// Use streaming
err := json.NewDecoder(file).Decode(&output)
```

### 3. Repeated Append Operations
**Impact**: MEDIUM
**Files**: Multiple locations

**Issue**: Growing slices without pre-allocation

**Fix**: Pre-allocate when size is known or estimable

## Concurrency Improvements

### 1. Channel Buffer Sizing
**Impact**: HIGH
**File**: `runner.go:354`

**Issue**: Fixed buffer `maxWorkers*10` may cause blocking

**Fix**:
```go
// Make configurable or scale with file count
bufferSize := max(maxWorkers*10, len(specFiles))
outputChan := make(chan OutputMessage, bufferSize)
```

### 2. Unnecessary Goroutine Creation
**Impact**: LOW
**File**: `stream_helper.go`

**Issue**: Creates goroutine for every command execution

**Consider**: Goroutine pool for command execution

## I/O Optimizations

### 1. Repeated File Operations
**Impact**: HIGH
**File**: `rspec/json_output.go:167-205`

**Issue**: Opens and reads file for each failure

**Fix**:
```go
// Cache file contents
var fileCache = make(map[string][]string)

func getCachedFileLines(path string) ([]string, error) {
    if lines, ok := fileCache[path]; ok {
        return lines, nil
    }
    // Read file and cache
    lines := readFileLines(path)
    fileCache[path] = lines
    return lines, nil
}
```

### 2. Individual Stat Calls
**Impact**: MEDIUM with many files
**File**: `grouper.go:26-33`

**Issue**: System call per file

**Fix**:
```go
// Use ReadDir for batch operations
entries, _ := os.ReadDir(dir)
for _, entry := range entries {
    info, _ := entry.Info()
    // Process file info
}
```

## Algorithm Improvements

### 1. Runtime Map Copying
**Impact**: MEDIUM
**File**: `runtime_tracker.go:77-80`

**Issue**: Full map copy on every access

**Options**:
1. Return read-only view
2. Use sync.Map
3. Copy-on-write pattern

### 2. Linear Search in Collections
**Impact**: LOW (small collections)
**Files**: Various

**Consider**: Using maps for lookups if collections grow

## Resource Management

### 1. Scanner Buffer Limits
**Impact**: HIGH for large output
**File**: `stream_helper.go:32-81`

**Issue**: Default 64KB buffer

**Fix**:
```go
scanner := bufio.NewScanner(stdout)
scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB buffer
```

### 2. Defer in Loops
**Impact**: LOW
**Files**: Various

**Issue**: Defer in loops delays cleanup

**Better**: Explicit cleanup or extract to function

## Quick Win Checklist

### Immediate (< 1 hour each)
- [ ] Fix race condition in `GetJSONRowsFormatterPath()`
- [ ] Add slice capacity hints in `test_collector.go`
- [ ] Switch to JSON streaming in `json_output.go`
- [ ] Increase scanner buffer sizes
- [ ] Pre-allocate string builders

### Short Term (< 1 day each)
- [ ] Implement file content caching
- [ ] Add debug flag checks before string building
- [ ] Optimize `fmt.Sprintf` usage
- [ ] Make channel buffer sizes dynamic

### Medium Term (1-3 days each)
- [ ] Batch file stat operations
- [ ] Implement sync.Map for runtime tracker
- [ ] Add goroutine pool for commands
- [ ] Profile and optimize hot paths

## Performance Testing Plan

### Benchmarks to Add
```go
// Benchmark file discovery
func BenchmarkFindTestFiles(b *testing.B)

// Benchmark parallel execution
func BenchmarkRunSpecsInParallel(b *testing.B)

// Benchmark output parsing
func BenchmarkParseRSpecOutput(b *testing.B)

// Benchmark runtime tracking
func BenchmarkRuntimeTracker(b *testing.B)
```

### Profiling Commands
```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./...
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./...
go tool pprof -http=:8080 mem.prof

# Execution trace
go test -trace=trace.out ./...
go tool trace trace.out
```

## Expected Impact

### Performance Gains
- **Memory**: 20-30% reduction in allocations
- **CPU**: 10-15% reduction in string operations
- **I/O**: 40-50% reduction in file operations
- **Latency**: 25% reduction in large test suite execution

### Risk Assessment
- **Low Risk**: String optimizations, pre-allocations
- **Medium Risk**: Caching, channel sizing
- **High Risk**: Concurrency changes, algorithm changes

## Conclusion

The codebase is well-structured but has clear optimization opportunities. Priority should be:

1. **Fix race condition** (critical)
2. **Memory pre-allocation** (easy wins)
3. **I/O optimization** (high impact)
4. **String operations** (cumulative benefit)

Most optimizations are low-risk and can be implemented incrementally with proper testing.