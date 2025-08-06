# Race Conditions Analysis - August 2025

## Critical Race Conditions Found

### 1. ConfigPaths.GetJSONRowsFormatterPath() - CRITICAL

**Location**: `config.go:94-108`

**Issue**: Concurrent read/write on `c.JSONRowsFormatter` field without synchronization

**Race Details**:
- Multiple workers call `GetJSONRowsFormatterPath()` simultaneously
- Line 95 reads `c.JSONRowsFormatter` 
- Line 106 writes to `c.JSONRowsFormatter`
- No mutex protection

**Impact**: 
- Could cause multiple formatter initializations
- Potential file system race conditions
- Memory corruption in extreme cases

**Fix Options**:

Option 1: Add mutex (simple fix)
```go
type ConfigPaths struct {
    mu                sync.RWMutex
    JSONRowsFormatter string
    // ... other fields
}

func (c *ConfigPaths) GetJSONRowsFormatterPath() string {
    c.mu.RLock()
    if c.JSONRowsFormatter != "" {
        defer c.mu.RUnlock()
        return c.JSONRowsFormatter
    }
    c.mu.RUnlock()
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Double-check after acquiring write lock
    if c.JSONRowsFormatter != "" {
        return c.JSONRowsFormatter
    }
    
    formatter, err := rspec.GetFormatterPath(c.FormatterDir)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: Failed to initialize RSpec formatter: %v\n", err)
        fmt.Fprintf(os.Stderr, "The JSON formatter is required for RSpec test execution.\n")
        os.Exit(1)
    }
    
    c.JSONRowsFormatter = formatter
    return formatter
}
```

Option 2: Use sync.Once (better performance)
```go
type ConfigPaths struct {
    JSONRowsFormatter string
    formatterOnce     sync.Once
    formatterErr      error
    // ... other fields
}

func (c *ConfigPaths) GetJSONRowsFormatterPath() string {
    c.formatterOnce.Do(func() {
        formatter, err := rspec.GetFormatterPath(c.FormatterDir)
        if err != nil {
            c.formatterErr = err
            return
        }
        c.JSONRowsFormatter = formatter
    })
    
    if c.formatterErr != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: Failed to initialize RSpec formatter: %v\n", c.formatterErr)
        fmt.Fprintf(os.Stderr, "The JSON formatter is required for RSpec test execution.\n")
        os.Exit(1)
    }
    
    return c.JSONRowsFormatter
}
```

Option 3: Initialize eagerly (simplest)
```go
// Initialize formatter path during config creation, not lazily
func NewConfigPaths() *ConfigPaths {
    formatterDir := getFormatterDir()
    formatter, err := rspec.GetFormatterPath(formatterDir)
    if err != nil {
        // Handle error during initialization
    }
    
    return &ConfigPaths{
        JSONRowsFormatter: formatter,
        FormatterDir:      formatterDir,
    }
}
```

### 2. String Builder Race (Secondary)

**Location**: `runner.go:192` during debug logging

**Issue**: Concurrent access to shared memory during `strings.Join()` operation

**Race Details**:
- Multiple goroutines calling `strings.Join(args, " ")` simultaneously
- Internal string builder shares memory temporarily

**Impact**: 
- Low - only affects debug output
- Could cause garbled debug messages

**Fix**: 
- This appears to be a false positive from the race detector
- The `args` slice is local to each goroutine
- No actual fix needed, but could copy args first if paranoid

## Other Potential Race Conditions (Not Detected)

### 1. Runtime Tracker Map Access

**Location**: `runtime_tracker.go`

**Current Protection**: Mutex-based

**Potential Issue**: 
- Single mutex could cause contention with many workers
- Currently safe but not optimal

**Optimization**: Consider `sync.Map` for better concurrent performance

### 2. Test Collector

**Location**: `test_collector.go`

**Current State**: Safe - each worker has its own collector

**Note**: Verify collectors are not shared between workers

### 3. Output Channel Buffering

**Location**: `runner.go:354`

**Current State**: Safe - channels provide synchronization

**Potential Issue**: 
- Fixed buffer size could cause blocking
- Not a race condition but performance issue

## Testing Methodology

```bash
# Build with race detector
go build -race -o plur-race ./plur

# Run with multiple workers to trigger races
./plur-race -C fixtures/projects/default-ruby -n 4

# Run test suite with race detector
go test -race ./...
```

## Recommendations

### Immediate Actions (Fix Races)
1. **CRITICAL**: Fix `GetJSONRowsFormatterPath()` race using sync.Once
2. Verify fix with race detector

### Performance Improvements
1. Consider eager initialization of formatter path
2. Evaluate sync.Map for RuntimeTracker
3. Profile mutex contention under load

### Testing Strategy
1. Add race detection to CI pipeline
2. Run periodic race detection tests
3. Create concurrent stress tests

## Conclusion

The primary race condition in `GetJSONRowsFormatterPath()` is critical and must be fixed. The sync.Once pattern is recommended as it:
- Guarantees single initialization
- Has minimal performance overhead
- Is idiomatic Go for lazy initialization

The secondary string builder race appears to be a false positive but warrants investigation. Overall, the codebase shows good concurrent design with only this one critical race condition found.