# Minitest Support & Framework Abstraction Plan

> **Note**: This document contains the original analysis and plan. For current implementation status, see:
> - [Minitest PRD](../wip/minitest-prd.md)
> - [Current TODOs](../wip/minitest-todo.md)
> - [Implementation Guide](../wip/minitest-implementation-guide.md)

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

## How parallel_tests Handles Minitest

Based on analysis of the parallel_tests codebase, here's their approach:

### Output Parsing
- **No verbose mode**: They parse standard minitest output, not `-v` verbose output
- **Pattern matching**: Look for summary line `"X tests, Y assertions, Z failures, W errors"`
- **Simple regex**: `line =~ /\d+ failure(?!:)/` to identify result lines
- **Color stripping**: Remove ANSI codes before parsing

### Command Execution
```ruby
# Single file approach would be:
ruby -Itest test/models/user_test.rb

# Multiple files (what parallel_tests does):
ruby -Itest -e "%w[test/file1.rb test/file2.rb].each { |f| require %{./\#{f}} }"
```

### Runtime Tracking
- Uses Ruby metaprogramming to hook into `Minitest::Runnable.run`
- Tracks runtime per test file (not individual tests)
- Format: `"test/models/user_test.rb:1.234"` (filename:seconds)
- No custom formatter needed

### Key Insight
parallel_tests keeps it simple - no custom formatters, just parse the standard output and use Ruby hooks for timing. This validates our approach of starting simple.

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

## Framework Type Configuration

### CLI and Config Design
```go
// New type for framework selection
type TestFramework string

const (
    FrameworkRSpec    TestFramework = "rspec"    // default
    FrameworkMinitest TestFramework = "minitest"
)

// Add to Config struct:
Framework TestFramework

// CLI flags:
-t, --type     Test framework type (rspec|minitest) [default: rspec]

// TOML config:
type = "minitest"  # or "rspec"
```

### Auto-Detection Logic
1. Check for explicit `-t` flag or config setting (highest priority)
2. Check directory structure:
   - `test/` directory → minitest
   - `spec/` directory → rspec
3. Check Gemfile for framework gems (future enhancement)
4. Default to rspec for backward compatibility

## Implementation Approach

### Phase 1: Refactor Types ✅ COMPLETED
1. Created framework-agnostic types (TestFile, TestFailure)
2. Updated existing code to use new types
3. Added conversion function (convertRSpecFailures) in runner.go
4. All existing tests pass

### Remaining RSpec Dependencies After Phase 1:
- `config.go` - GetFormatterPath (needed until Phase 2)
- `runtime_tracker.go` - Uses rspec.Example (convert in Phase 3)
- `result.go` - Uses rspec.ExtractFailingLine (consider utilities package)
- `runner.go` - Has rspec imports for parsing and conversion

### Phase 2: Add Minitest Support (IN PROGRESS)
Based on parallel_tests analysis and decisions:

1. **Add Framework Type Support** ✅ COMPLETED (2025-06-22)
   - Added TestFramework enum ("rspec", "minitest") 
   - Added `-t | --type` flag to spec and watch commands
   - Updated Config struct with Framework field
   - Added auto-detection based on test/ vs spec/ directories
   - TOML config support via `spec.type = "minitest"`

2. **Create Minitest Module** (`plur/minitest/`) ✅ COMPLETED (2025-06-22)
   - Created output parser for standard minitest format (not verbose)
   - Parses: `"X tests, Y assertions, Z failures, W errors, Z skips"`
   - Strips ANSI color codes like parallel_tests
   - Command builder using `ruby -Itest` pattern
   - Single file: `ruby -Itest test/file.rb`
   - Multiple files: `ruby -Itest -e "[files].each { |f| require f }"`
   - Extracts failure messages for reporting

3. **Refactor Command Building** ✅ COMPLETED (2025-06-22)
   - Extracted CommandBuilder interface
   - RSpecCommandBuilder: uses existing formatter and color logic
   - MinitestCommandBuilder: uses `ruby -Itest` pattern from minitest package
   - Updated RunSpecFile to use command builders
   - Framework-specific command building now properly dispatched

4. **Basic Execution First** ⚠️ PARTIALLY COMPLETE
   - ✅ Tests execute successfully with proper command building
   - ✅ No runtime tracking (as planned)
   - ✅ Duration tracked from Go side
   - ❌ Output capture not working properly (shows "0 tests, 0 assertions...")
   - ❌ Progress reporting (dots) not implemented

5. **Output Parsing** 🔄 NEEDS WORK
   - ✅ ANSI code stripping implemented
   - ✅ Summary line parsing implemented
   - ❌ Real-time output streaming not working
   - ❌ Progress indicators not captured


### Phase 3: Runtime Tracking & Refinements
1. Add runtime tracking from Go side (measure per-file execution)
2. Create generic Example type to replace rspec.Example
3. Convert RuntimeTracker to use generic types
4. Consider verbose mode for progress reporting (future)

## Key Decisions

1. **Start with basic execution** - Get minitest running first, add runtime tracking later (track from Go side, not Ruby hooks)

2. **Simple output parsing** - Follow parallel_tests approach: parse standard output, not verbose mode

3. **Use `ruby -Itest` pattern** - Consistent with parallel_tests, avoid Rails-specific commands for now

4. **Framework type flag** - Add `-t | --type` for explicit control, with auto-detection as convenience

5. **Prove abstraction early** - Implement minitest support in Phase 2 before more refactoring

## Success Criteria

### Phase 1 (COMPLETED ✅):
- RSpec projects continue to work exactly as before ✅
- No performance regression ✅
- Clean separation between generic types and RSpec-specific types ✅
- All tests passing ✅

### Phase 2 - Minitest Support (IN PROGRESS):
- ✅ `-t minitest` flag works correctly
- ✅ Auto-detection identifies test/ directories
- ✅ Command building follows parallel_tests pattern
- ⚠️ Minitest projects run but output capture needs fixing
- ❌ Progress reporting (dots) not implemented
- ❌ Standard minitest output not captured properly
- ✅ Existing RSpec functionality unchanged

### Phase 3 - Runtime & Refinements:
- Runtime tracking works for both frameworks
- Generic Example type replaces rspec.Example
- Further framework abstractions as needed

---

*This is a living document. As implementation proceeds, we'll update based on discoveries and feedback.*