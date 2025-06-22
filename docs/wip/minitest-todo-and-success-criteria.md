# Minitest Support - TODOs and Success Criteria

## Implementation Phases

### Phase 1: Refactor Types ✅ COMPLETED (2025-06-22)
1. Created framework-agnostic types (TestFile, TestFailure)
2. Updated existing code to use new types
3. Added conversion function (convertRSpecFailures) in runner.go
4. All existing tests pass

### Phase 2: Add Minitest Support (IN PROGRESS)

1. **Add Framework Type Support** ✅ COMPLETED (2025-06-22)
   - Added TestFramework enum ("rspec", "minitest") 
   - Added `-t | --type` flag to spec and watch commands
   - Updated Config struct with Framework field
   - Added auto-detection based on test/ vs spec/ directories
   - TOML config support via `spec.type = "minitest"`

2. **Create Minitest Module** (`rux/minitest/`) ✅ COMPLETED (2025-06-22)
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

4. **Basic Execution First** ✅ COMPLETED (2025-06-22)
   - ✅ Tests execute successfully with proper command building
   - ✅ No runtime tracking (as planned)
   - ✅ Duration tracked from Go side
   - ✅ Output capture fixed with streaming implementation
   - ✅ Progress reporting (dots) implemented

5. **Output Parsing** ✅ COMPLETED (2025-06-22)
   - ✅ ANSI code stripping implemented
   - ✅ Summary line parsing implemented
   - ✅ Real-time output streaming working
   - ✅ Progress indicators captured and displayed

### Phase 3: Event-Based Refactoring (IN PROGRESS)

1. **Core Types** ✅ COMPLETED
   - Created notifications.go with TestEvent enum
   - Created TestNotification interface
   - Created concrete notification types

2. **Framework Parsers** ✅ COMPLETED
   - Created rspec/parser.go with JSON parsing
   - Created minitest/notification_parser.go with text parsing
   - Both implement TestOutputParser interface

3. **Notification Accumulator** 🔄 NEXT
   - Create accumulator to collect events
   - Build TestResult from accumulated notifications

4. **Parser Integration** 🔄 TODO
   - Update RunRSpecFiles to use parser
   - Update RunMinitestFiles to use parser
   - Extract common streaming logic

### Phase 4: Runtime Tracking & Refinements (FUTURE)
1. Add runtime tracking from Go side (measure per-file execution)
2. Create generic Example type to replace rspec.Example
3. Convert RuntimeTracker to use generic types
4. Consider verbose mode for progress reporting (future)

## Fixed Issues ✅

1. **Output Streaming in RunMinitestFiles** ✅:
   - Replaced `cmd.CombinedOutput()` with stdout/stderr pipes
   - Uses `bufio.Scanner` to read output line-by-line
   - Similar to RSpec implementation but parses plain text

2. **Progress Detection** ✅:
   - Added `isProgressLine()` function
   - Detects lines with only progress indicators
   - Sends OutputMessage events for dots/F/E/S

3. **Output Accumulation** ✅:
   - Builds output string while streaming
   - Parses summary with updated regex ("runs" not "tests")
   - Maintains real-time progress and final summary

4. **Integration Tests** ✅:
   - Updated expectations to match "runs" in summary
   - Fixed error output expectations
   - All minitest integration tests passing

## Success Criteria

### Phase 1 (COMPLETED ✅):
- RSpec projects continue to work exactly as before ✅
- No performance regression ✅
- Clean separation between generic types and RSpec-specific types ✅
- All tests passing ✅

### Phase 2 - Minitest Support (COMPLETED ✅):
- ✅ `-t minitest` flag works correctly
- ✅ Auto-detection identifies test/ directories
- ✅ Command building follows parallel_tests pattern
- ✅ Minitest projects run with proper output capture
- ✅ Progress reporting (dots) implemented
- ✅ Standard minitest output captured and displayed
- ✅ Existing RSpec functionality unchanged
- ✅ All integration tests passing

### Phase 3 - Runtime & Refinements:
- Runtime tracking works for both frameworks
- Generic Example type replaces rspec.Example
- Further framework abstractions as needed

## Remaining RSpec Dependencies After Phase 1:
- `config.go` - GetFormatterPath (needed until Phase 2)
- `runtime_tracker.go` - Uses rspec.Example (convert in Phase 3)
- `result.go` - Uses rspec.ExtractFailingLine (consider utilities package)
- `runner.go` - Has rspec imports for parsing and conversion