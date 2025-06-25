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

## Phase 3: Event-Based Refactoring [PARTIALLY COMPLETE]

**Status**: Architecture refactored but output format issues remain. See [Refactoring Summary](minitest-refactoring-summary.md) for details.

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

**Outstanding Issues:**
- [ ] Output format mismatch - showing RSpec-style instead of minitest-style
- [ ] Failures not being detected properly in minitest-failures tests
- [ ] Framework context lost by the time we print results

See [Refactoring Summary](minitest-refactoring-summary.md) for detailed analysis.

### Code Organization [COMPLETE]
- [x] Fix package structure (notification types duplicated in 3 places) → Moved to `types` package
- [x] Move parsers to appropriate packages → Parser interface in `types` package
- [x] Consolidate duplicate type definitions → All using shared `types` package
- [x] Clean up dead code from refactoring → Removed duplicate functions and types

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
- [ ] Output format issues → Minitest output being reformatted as RSpec style
- [ ] Failure detection issues → Minitest failures not properly parsed
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
- [ ] Minitest failure parsing needs fixes

See [Refactoring Summary](minitest-refactoring-summary.md) for analysis of current issues.

### Next Steps

1. Fix output format issues - preserve minitest's native output format
2. Fix failure detection in minitest parser
3. Make PrintResults framework-aware
4. Add framework context to TestResult or TestSummary
5. Test end-to-end with real minitest projects
6. Update documentation

---

Last updated: 2025-06-24