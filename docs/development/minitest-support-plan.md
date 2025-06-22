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

### Phase 2: Add Minitest Support (IMMEDIATE PRIORITY)
Based on parallel_tests analysis and decisions:

1. **Add Framework Type Support**
   - Add TestFramework enum and `-t | --type` flag
   - Update Config struct with Framework field
   - Add framework to TOML config support

2. **Create Minitest Module** (`rux/minitest/`)
   - Output parser for standard minitest format (not verbose)
   - Parse: `"X tests, Y assertions, Z failures, W errors"`
   - Command builder using `ruby -Itest` pattern

3. **Refactor Command Building**
   - Extract CommandBuilder interface
   - RSpecCommandBuilder: current logic
   - MinitestCommandBuilder: `ruby -Itest -e "require files"`

4. **Basic Execution First**
   - No runtime tracking initially
   - Track duration from Go side (before/after execution)
   - Focus on getting tests running in parallel

5. **Output Parsing**
   - Strip ANSI codes like parallel_tests
   - Extract test counts from summary line
   - Convert to TestResult format

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

### Phase 2 - Minitest Support (IMMEDIATE NEXT):
- Minitest projects can run with same parallelization as RSpec
- `-t minitest` flag works correctly
- Auto-detection identifies test/ directories
- Standard minitest output parsed correctly
- Test counts and failures extracted accurately
- Existing RSpec functionality unchanged

### Phase 3 - Runtime & Refinements:
- Runtime tracking works for both frameworks
- Generic Example type replaces rspec.Example
- Further framework abstractions as needed

---

*This is a living document. As implementation proceeds, we'll update based on discoveries and feedback.*