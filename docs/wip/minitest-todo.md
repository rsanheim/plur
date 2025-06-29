# Minitest TODOs

This is the single source of truth for all Minitest support tasks. Tasks are organized by phase with links to relevant documentation.

## Phase 1: Framework Abstraction [COMPLETE]

- [x] Create generic TestFile type separate from RSpec
- [x] Create generic TestResult type  
- [x] Add Framework field to TestFile
- [x] Implement framework detection in FindTestFiles
- [x] Create CommandBuilder interface
- [x] Move RSpec command building to RSpecCommandBuilder
- [x] Add framework routing in RunSpecFile

## Phase 2: Basic Minitest Support [COMPLETE]

- [x] Create MinitestCommandBuilder implementation
- [x] Add minitest file pattern detection (`*_test.rb`, `test_*.rb`)
- [x] Implement RunMinitestFiles function
- [x] Fix output streaming (use pipes instead of CombinedOutput) → [Issue analysis](minitest-implementation-guide.md#the-streaming-problem)
- [x] Parse minitest output format ("runs" not "tests")
- [x] Capture progress indicators (., F, E, S)
- [x] Add minitest integration tests
- [x] Fix all failing integration tests

## Phase 3: Event-Based Refactoring [MOSTLY COMPLETE]

**Status**: Architecture refactored, failure parsing implemented, but output format issues remain. See [Refactoring Summary](minitest-refactoring-summary.md) for details.

### Core Types [COMPLETE]
- [x] Create TestEvent enum (TestStarted, TestPassed, TestFailed, etc.) → [Architecture](test-event-architecture.md#test-events)
- [x] Create TestNotification interface
- [x] Create TestCaseNotification type
- [x] Create SuiteNotification type  
- [x] Create OutputNotification type

### Parser Implementation [COMPLETE]
- [x] Create TestOutputParser interface → [Parser design](test-event-architecture.md#parser-interface)
- [x] Implement RSpecOutputParser for JSON
- [x] Implement MinitestOutputParser for text output
- [x] Add progress indicator parsing to MinitestParser
- [x] Add failure detail parsing to MinitestParser → **NEW** (2025-06-27)

### Integration Tasks [MOSTLY COMPLETE]
- [x] Create TestCollector to collect events → [Accumulator spec](test-event-architecture.md#accumulator)
- [x] Refactor RunRSpecFiles to use RSpecOutputParser → **COMPLETE!** Successfully refactored
- [x] Refactor RunMinitestFiles to use event-based architecture → **COMPLETE!** But has issues - see below

#### Minitest Event-Based Refactoring Status

**Completed Steps:**
- [x] Renamed `minitest.NotificationParser` to `minitest.OutputParser`
- [x] Created `RSpecParser` and `MinitestParser` wrapper types
- [x] Created `parser_factory.go` with `NewTestOutputParser(framework TestFramework)`
- [x] Removed `rspec_parser.go` after moving logic to factory
- [x] Extracted common streaming logic to `stream_helper.go`
- [x] Updated RunRSpecFiles to use factory and shared helper
- [x] Updated RunMinitestFiles to use factory and shared helper
- [x] Removed `parseMinitestOutput()` and `isProgressLine()` functions
- [x] Removed unused imports and dead code
- [x] Fix package structure (notification types duplicated in 3 places) → Moved to `types` package
- [x] Move parsers to appropriate packages → Parser interface in `types` package
- [x] Consolidate duplicate type definitions → All using shared `types` package
- [x] Clean up dead code from refactoring → Removed duplicate functions and types
- [x] Failures not being detected properly in minitest-failures tests → **FIXED** (2025-06-27)

**Outstanding Issues:**
- [x] Add a -C global flag to rux to change the working dir → **COMPLETE** (2025-06-28)

- [ ] Fix "Running" prelude output for minitest tests - we still say 'spec files' when its 'test files'. Also, if its really 1 spec or test file, we should never say "in parallel" - that doesn't make sense. If its one file we can't parallelize.

Example: 
  ```
    > rux test/calculator_test.rb 
    rux version v0.7.6-0.20250628065354-2392e7f82dde
    Running 1 spec files in parallel using 1 workers (20 cores available)...
    Using size-based grouped execution: 1 file across 18 workers
  ```

- [ ] Create an enum for "dot", "fail", "error", etc - and use that everywhere for progress tracking. No more repeating those strings throughout for progress tracking.
- [x] Fix output format for minitest → **COMPLETE** (2025-06-28)
  - Extended TestOutputParser interface with FormatSummary method
  - Each parser provides framework-specific formatting
  - RSpec shows: "X examples, Y failures"  
  - Minitest shows: "X runs, Y assertions, Z failures, W errors, V skips"
  - Framework tracked through TestResult to TestSummary
- [x] Fix minitest integration spec → **COMPLETE** (2025-06-28)

## Phase 4: Future Enhancements

- [ ] Add runtime tracking for better test distribution
- [ ] Support custom Minitest reporters
- [ ] Add Test::Unit support
- [ ] Optimize channel buffer sizes
- [ ] Add framework-specific configuration options

## Testing & Documentation

- [x] Integration tests for minitest success cases
- [x] Integration tests for minitest failure cases  
- [ ] Unit tests for MinitestOutputParser (renamed from NotificationParser)
- [x] Unit tests for TestCollector
- [ ] Update user documentation for minitest usage
- [ ] Add minitest examples to README
- [ ] Document framework detection logic
- [ ] Add troubleshooting guide for minitest

## Known Issues to Address

- [x] Type duplication between packages → **RESOLVED**: Created shared `types` package
- [x] Parser integration incomplete → **RESOLVED**: Both parsers integrated
- [x] Output format issues → **RESOLVED** (2025-06-28): Minitest output now shows native format
- [x] Failure detection issues → **RESOLVED** (2025-06-27): Minitest failures now properly parsed
- [ ] No error recovery for malformed output
- [ ] Limited minitest reporter support

See [Refactoring Summary](minitest-refactoring-summary.md#current-challenges) for details.

## Progress Summary (2025-06-24)

### Major Achievements:
1. **Created TestCollector** - Fully implemented with test coverage
2. **Fixed Package Structure** - Created `types` package to eliminate code duplication
3. **Code Cleanup Complete** - Removed duplicate code, moved special notification types to types package
4. **RSpec Runner Refactored** - RunRSpecFiles now uses event-based architecture with parser + accumulator
5. **Minitest Runner Refactored** - RunMinitestFiles now uses event-based architecture
6. **Parser Factory Created** - Single factory method for all test framework parsers
7. **Shared Streaming Logic** - Extracted common output streaming to helper function
8. **RuntimeTracker Updated** - Now uses TestNotification types instead of custom structs
9. **Old Parsing Code Removed** - Cleaned up legacy minitest parsing functions

### Current State:
- [x] Minitest basic support is **functional** but has output format issues
- [x] Event-based architecture is **implemented** for both frameworks
- [x] Package structure is **clean and organized**
- [x] Both runners use new event-based architecture
- [ ] Output formatting needs to be framework-aware
- [x] Minitest failure parsing → **FIXED** (2025-06-27)

See [Refactoring Summary](minitest-refactoring-summary.md) for analysis of current issues.

### Next Steps

1. Fix output format issues - preserve minitest's native output format
2. ~~Fix failure detection in minitest parser~~ → **COMPLETED** (2025-06-27)
3. Make PrintResults framework-aware
4. Add framework context to TestResult or TestSummary
5. Test end-to-end with real minitest projects
6. Update documentation

## Progress Summary (2025-06-27)

### Minitest Failure Parsing Implementation:

Successfully added failure parsing to the minitest OutputParser:

1. **State Management** - Added `afterFinished` and `inFailureDetails` states to track parsing phase
2. **Failure Header Detection** - Parses patterns like "1) Failure:" and "2) Error:"
3. **Test Location Extraction** - Extracts test class, method, file path, and line number from failure headers
4. **Message Accumulation** - Collects assertion failure messages across multiple lines
5. **TestNotification Creation** - Creates proper TestCaseNotification objects with TestException details

The parser now correctly:
- Detects when test execution completes and failure details begin
- Parses each failure block independently
- Extracts all relevant failure information
- Maintains the raw output for display
- Creates notifications that integrate with the existing event system

### Double-Counting Fix Implementation: ✅ COMPLETE

Successfully fixed the critical issue where failures were being counted twice:

1. **Refactored to State Machine** - Replaced multiple boolean flags with a clean state machine
2. **Added ProgressCounts** - Track test progress separately from actual results
3. **Index-Based Tracking** - Map progress indicators to test notifications by index
4. ~~**In-Place Updates** - Update existing notifications with failure details~~ **REPLACED**
5. **Test-Driven Development** - Created comprehensive tests before implementation

### ProgressEvent Refactoring: ✅ COMPLETE (2025-06-27)

Implemented a cleaner approach using ProgressEvent for real-time display:

1. **Created ProgressEvent Type** - Minimal type for progress indicators only
2. **Separated Concerns** - Progress events for display, test notifications for results
3. **Eliminated Duplicates** - No more duplicate notifications or filtering needed
4. **Cleaner Architecture** - Removed all complex filtering logic from collectors and display

The parser now:
- Emits ProgressEvents during progress parsing for real-time feedback
- Creates complete TestCaseNotifications only during failure parsing and summary
- Correctly reports test counts (7 tests, 4 failures for single file)
- No duplicate display entries
- All Go tests passing
- Integration tests correctly show proper counts

### Remaining Issues:
- [x] Output format shows RSpec-style summaries → **FIXED** (2025-06-28)
- [ ] Need to handle Error exceptions differently from assertion Failures
- [x] Framework context passed through to PrintResults → **COMPLETE** (2025-06-28)
- [x] Multi-file minitest runs show "Failure number out of range" errors → **FIXED** (2025-06-28)
  - Removed index-based failure correlation entirely
  - Parser now uses actual test names as IDs
  - No more tracking failures across files with indices

## Progress Summary (2025-06-28)

### Major Fixes:

1. **Added -C Flag** - Change directory before running (like git -C)
2. **Fixed Output Format** - Minitest now shows native format using FormatSummary method
3. **Fixed "Failure number out of range"** - Removed problematic index-based tracking
4. **All Integration Tests Pass** - Including minitest_integration_spec.rb

### Architectural Improvements:

1. **"Tell Don't Ask" Pattern** - Parsers tell the runner how to format output
2. **Simplified Failure Tracking** - Use actual test names instead of generic indices
3. **Framework Context** - Added Framework field to TestResult and TestSummary

---

Last updated: 2025-06-28