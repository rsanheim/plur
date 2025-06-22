# Test Execution Refactoring Plan

## Overview
Refactor test execution to use an event-based abstraction that decouples framework-specific logic from generic test running concerns.

## Goals
1. Fix minitest output streaming (currently broken)
2. Create reusable abstractions for multiple test frameworks
3. Separate parsing, accumulation, and reporting concerns
4. Enable easier addition of new frameworks (Test::Unit, etc.)

## Implementation Steps

### Phase 1: Core Types (Start Here)
1. **Create notifications.go** with:
   - TestEvent enum (TestPassed, TestFailed, etc.)
   - TestNotification interface
   - Three concrete types: TestCaseNotification, SuiteNotification, OutputNotification
   - Helper functions

2. **Create parser.go** with:
   - TestOutputParser interface
   - Base parser functionality

### Phase 2: Framework Parsers
1. **Create rspec/parser.go**:
   - Move JSON parsing logic from runner.go
   - Implement TestOutputParser interface
   - Parse RUX_JSON lines into notifications

2. **Create minitest/parser.go**:
   - Implement text-based parsing
   - Handle progress indicators (., F, E)
   - Parse failure messages and summary

### Phase 3: Accumulator
1. **Create accumulator.go**:
   - NotificationAccumulator struct
   - Collect test results, failures, counts
   - Build final TestResult

### Phase 4: Fix Minitest Streaming
1. **Update RunMinitestFiles**:
   - Change from cmd.CombinedOutput() to streaming with pipes
   - Use the same pattern as RSpec
   - Process output line-by-line

### Phase 5: Refactor Runner
1. **Extract common logic**:
   - Create generic RunTestFiles function
   - Takes framework type, uses appropriate parser
   - Handles streaming, progress reporting, accumulation

2. **Update framework-specific functions**:
   - RunRSpecFiles becomes thin wrapper
   - RunMinitestFiles becomes thin wrapper

## Testing Strategy
1. Start with unit tests for parsers
2. Ensure existing integration tests pass
3. Add new tests for event flow
4. Manual testing with real projects

## Commit Points
- After each major component is working
- Keep tests passing at each step
- Document changes as we go

## Current Status
- Fixed minitest command builder tests ✓
- Ready to start Phase 1