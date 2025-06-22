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

### Phase 3: Runtime Tracking & Refinements (FUTURE)
1. Add runtime tracking from Go side (measure per-file execution)
2. Create generic Example type to replace rspec.Example
3. Convert RuntimeTracker to use generic types
4. Consider verbose mode for progress reporting (future)

## Next Steps to Fix Output Issues

1. **Implement Output Streaming in RunMinitestFiles**:
   - Replace `cmd.CombinedOutput()` with stdout/stderr pipes
   - Use `bufio.Scanner` to read output line-by-line
   - Similar to RSpec implementation but parse plain text instead of JSON

2. **Add Progress Detection**:
   - Scan for minitest progress indicators in output
   - Look for patterns like:
     - Single dot (`.`) for passing test
     - `F` for failure
     - `E` for error
     - `in test_method_name` for current test
   - Send OutputMessage events to outputChan for each indicator

3. **Fix Output Accumulation**:
   - Build output string while streaming
   - Parse summary line when detected
   - Keep both real-time parsing and final summary

4. **Update Integration Tests**:
   - Change expectation from "tests" to "runs" in summary
   - Adjust error output expectations for minitest format

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

## Remaining RSpec Dependencies After Phase 1:
- `config.go` - GetFormatterPath (needed until Phase 2)
- `runtime_tracker.go` - Uses rspec.Example (convert in Phase 3)
- `result.go` - Uses rspec.ExtractFailingLine (consider utilities package)
- `runner.go` - Has rspec imports for parsing and conversion