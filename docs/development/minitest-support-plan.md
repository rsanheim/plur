# Minitest Support & Framework Abstraction Plan

## Current Status

**Phase 0**: ✅ COMPLETED (2024-12-21)
- Created fixture projects for minitest and test-unit
- Implemented helper methods using inline Ruby script approach
- Discovered that minitest `-v` flag provides test-level granularity

**Phase 1**: ✅ COMPLETED (2024-12-22)
- Created framework-agnostic types (TestFile, TestFailure)
- Updated TestResult to remove RSpec dependencies
- Updated TestSummary to use new TestFailure type
- Created temporary conversion functions in rspec package
- Updated all code references throughout codebase
- All tests passing with backward compatibility maintained

**Cleanup**: ✅ COMPLETED (2024-12-22)
- Removed unused rspec/conversion.go file
- Removed circular reference Result field from TestFile struct
- Renamed OutputMessage.SpecFile to Files for clarity
- Removed commented error handling code in result.go
- Kept JSONOutput field temporarily for runtime tracking (to be refactored in Phase 2)

## Core Type Refactoring

### Current State (RSpec-specific)

```go
// Current - tightly coupled to RSpec
type TestResult struct {
    SpecFile     string
    JSONOutput   *rspec.JSONOutput        // RSpec-specific
    Failures     []rspec.FailureDetail    // RSpec-specific
    // ... other fields
}

type TestSummary struct {
    AllFailures []rspec.FailureDetail     // RSpec-specific
    // ... other fields
}
```

### Proposed Framework-Agnostic Types

```go
// NEW - Represents a test file
type TestFile struct {
    Path     string      // Full path to the file
    Filename string      // Just the filename
    Result   *TestResult // Execution result for this file
}

// UPDATED - Framework-agnostic result
type TestResult struct {
    File         *TestFile
    Success      bool
    Output       string
    Error        error
    Duration     time.Duration
    FileLoadTime time.Duration
    
    // Counts
    ExampleCount int
    FailureCount int
    PendingCount int
    
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

### Phase 1: Refactor Types
1. Create framework-agnostic types
2. Update existing code to use new types
3. Add conversion functions in rspec package
4. Ensure all existing tests pass

### Phase 2: Add Minitest Support
1. Parse minitest verbose output (`-v` flag)
2. Convert parsed output to our generic types
3. Reuse existing parallel execution logic

### Phase 3: Framework Detection
1. Auto-detect based on directory structure (spec/ vs test/)
2. Allow override via config or CLI flag
3. Default to RSpec for backward compatibility

## Key Decisions

1. **No TestCase collection** - We don't need individual test case tracking for current functionality. The formatted output handles display.

2. **Reuse existing parallelization** - The parallel execution logic doesn't need to change, just the command building and output parsing.

3. **Minimal changes** - Keep the existing architecture, just swap out the types and add parsing for minitest output.

## Success Criteria

- RSpec projects continue to work exactly as before
- Minitest projects can run with same parallelization
- No performance regression
- Clean separation between framework-specific and generic code

---

*This is a living document. As implementation proceeds, we'll update based on discoveries and feedback.*