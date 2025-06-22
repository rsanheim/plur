# Test Execution Refactoring Plan

## Overview
Refactor test execution to use an event-based abstraction that decouples framework-specific logic from generic test running concerns.

## Goals
1. Fix minitest output streaming (currently broken)
2. Create reusable abstractions for multiple test frameworks
3. Separate parsing, accumulation, and reporting concerns
4. Enable easier addition of new frameworks (Test::Unit, etc.)

## Implementation Steps

### Phase 1: Core Types ✅ COMPLETE
1. **Created notifications.go** with:
   - TestEvent enum (TestPassed, TestFailed, etc.)
   - TestNotification interface
   - Three concrete types: TestCaseNotification, SuiteNotification, OutputNotification

2. **Created parser.go** with:
   - TestOutputParser interface

### Phase 2: Framework Parsers ✅ COMPLETE
1. **Created rspec/parser.go**:
   - Implemented JSON parsing logic
   - Implemented TestOutputParser interface
   - Parses RUX_JSON lines into notifications

2. **Created minitest/notification_parser.go**:
   - Implemented text-based parsing
   - Handles progress indicators (., F, E, S)
   - Parses failure messages and summary

### Phase 3: Accumulator 🚧 NEXT
1. **Create accumulator.go**:
   - NotificationAccumulator struct
   - Collect test results, failures, counts
   - Build final TestResult

### Phase 4: Fix Minitest Streaming ✅ COMPLETE
1. **Updated RunMinitestFiles**:
   - Changed from cmd.CombinedOutput() to streaming with pipes
   - Uses the same pattern as RSpec
   - Processes output line-by-line
   - Fixed progress indicator parsing

### Phase 5: Refactor Runner 🚧 TODO
1. **Extract common logic**:
   - Create generic RunTestFiles function
   - Takes framework type, uses appropriate parser
   - Handles streaming, progress reporting, accumulation

2. **Update framework-specific functions**:
   - RunRSpecFiles becomes thin wrapper
   - RunMinitestFiles becomes thin wrapper

### Phase 6: Package Cleanup 🚧 TODO
1. **Fix duplicated types**:
   - Notification types are duplicated in rspec/parser.go and minitest/notification_parser.go
   - Need to reorganize imports or create shared package

2. **Consolidate parsers**:
   - Consider moving parsers to main package or creating parsing package

## Testing Strategy
1. ✅ Unit tests for parsers (minitest parser tests fixed)
2. ✅ Existing integration tests pass
3. 🚧 Add new tests for event flow
4. ✅ Manual testing with real projects

## Current Status
- ✅ Phase 1: Core types created
- ✅ Phase 2: Framework parsers implemented (with type duplication issue)
- ✅ Phase 4: Minitest streaming fixed and working
- ✅ All tests passing
- ✅ Command builders consolidated

## Next: Phase 3 - Create NotificationAccumulator