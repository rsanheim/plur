# Minitest Support & Framework Abstraction Plan

## Current Status

**Phase 0**: ✅ COMPLETED (2024-12-21)
- Created fixture projects for minitest and test-unit
- Implemented helper methods using inline Ruby script approach
- Discovered that minitest `-v` flag provides test-level granularity

**Phase 1**: ✅ COMPLETED (2025-06-22)
- Created framework-agnostic types (TestFile, TestFailure)
- Updated TestResult to use TestFile instead of string SpecFile
- Updated TestSummary to use TestFailure instead of rspec.FailureDetail
- Created conversion function (convertRSpecFailures) in runner.go
- Updated all code references throughout codebase
- Renamed OutputMessage.SpecFile to Files for clarity
- Created FormatTestFailure and FormatFailedExamples functions
- All tests passing with backward compatibility maintained

**Cleanup Notes**:
- Kept JSONOutput field temporarily for runtime tracking (to be refactored in Phase 2)
- TestFile has both Path and Filename (consider removing Filename in future)
- ExtractFailingLine exported from rspec package (consider moving to utilities)
- RuntimeTracker still uses rspec.Example (convert in Phase 2)

## Core Type Refactoring

### Phase 1 Implementation (COMPLETED)

The following types have been implemented:

```go
// NEW - Represents a test file
type TestFile struct {
    Path     string      // Full path to the file
    Filename string      // Just the filename (could be derived from Path)
}

// UPDATED - Framework-agnostic result
type TestResult struct {
    File         *TestFile
    State        TestState    // StateSuccess, StateFailed, StateError
    Output       string
    Error        error
    Duration     time.Duration
    FileLoadTime time.Duration
    JSONOutput   *rspec.JSONOutput  // Still RSpec-specific (Phase 2)
    
    // Counts
    ExampleCount int
    FailureCount int
    
    // Detailed failures
    Failures []TestFailure
    
    // Raw formatted output from framework
    FormattedFailures string
    FormattedSummary  string
}

// NEW - Framework-agnostic failure details
type TestFailure struct {
    File        *TestFile
    Description string
    LineNumber  int
    Message     string
    Backtrace   []string
}

// UPDATED - Already mostly framework-agnostic
type TestSummary struct {
    TotalExamples     int
    TotalFailures     int
    AllFailures       []TestFailure      // Changed from []rspec.FailureDetail
    TotalCPUTime      time.Duration
    WallTime          time.Duration
    TotalFileLoadTime time.Duration
    HasFailures       bool
    Success           bool
    ErroredFiles      []TestResult
    
    FormattedFailures string
    FormattedSummary  string
}
```

## Implementation Approach

### Phase 1: Refactor Types ✅ COMPLETED
1. Created framework-agnostic types (TestFile, TestFailure)
2. Updated existing code to use new types
3. Added conversion function (convertRSpecFailures) in runner.go
4. All existing tests pass

### Remaining RSpec Dependencies After Phase 1:
- `config.go` - GetFormatterPath (needed until Phase 2)
- `runtime_tracker.go` - Uses rspec.Example (convert in Phase 2)
- `result.go` - Uses rspec.ExtractFailingLine (consider utilities package)
- `runner.go` - Has rspec imports for parsing and conversion

### Phase 2: Add Minitest Support
1. Create generic Example type to replace rspec.Example
2. Convert RuntimeTracker to use generic types
3. Create minitest output parser for verbose output (`-v` flag)
4. Add convertMinitestOutput function similar to convertRSpecFailures
5. Add framework detection and dispatch logic
6. Reuse existing parallel execution logic

### Phase 3: Framework Detection
1. Auto-detect based on directory structure (spec/ vs test/)
2. Allow override via config or CLI flag
3. Default to RSpec for backward compatibility

## Key Decisions

1. **No TestCase collection** - We don't need individual test case tracking for current functionality. The formatted output handles display.

2. **Reuse existing parallelization** - The parallel execution logic doesn't need to change, just the command building and output parsing.

3. **Minimal changes** - Keep the existing architecture, just swap out the types and add parsing for minitest output.

## Success Criteria

### Phase 1 (COMPLETED ✅):
- RSpec projects continue to work exactly as before ✅
- No performance regression ✅
- Clean separation between generic types and RSpec-specific types ✅
- All tests passing ✅

### Phase 2 (TODO):
- Minitest projects can run with same parallelization
- Framework detection works automatically
- Runtime tracking works for both frameworks

---

*This is a living document. As implementation proceeds, we'll update based on discoveries and feedback.*