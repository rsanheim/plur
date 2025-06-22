# Minitest TODOs

This is the single source of truth for all Minitest support tasks. Tasks are organized by phase with links to relevant documentation.

## Phase 1: Framework Abstraction

- [x] Create generic TestFile type separate from RSpec
- [x] Create generic TestResult type  
- [x] Add Framework field to TestFile
- [x] Implement framework detection in FindTestFiles
- [x] Create CommandBuilder interface
- [x] Move RSpec command building to RSpecCommandBuilder
- [x] Add framework routing in RunSpecFile

## Phase 2: Basic Minitest Support

- [x] Create MinitestCommandBuilder implementation
- [x] Add minitest file pattern detection (`*_test.rb`, `test_*.rb`)
- [x] Implement RunMinitestFiles function
- [x] Fix output streaming (use pipes instead of CombinedOutput) → [Issue analysis](minitest-implementation-guide.md#the-streaming-problem)
- [x] Parse minitest output format ("runs" not "tests")
- [x] Capture progress indicators (., F, E, S)
- [x] Add minitest integration tests
- [x] Fix all failing integration tests

## Phase 3: Event-Based Refactoring

### Core Types
- [x] Create TestEvent enum (TestStarted, TestPassed, TestFailed, etc.) → [Architecture](test-event-architecture.md#test-events)
- [x] Create TestNotification interface
- [x] Create TestCaseNotification type
- [x] Create SuiteNotification type  
- [x] Create OutputNotification type

### Parser Implementation
- [x] Create TestOutputParser interface → [Parser design](test-event-architecture.md#parser-interface)
- [x] Implement RSpecOutputParser for JSON
- [x] Implement MinitestOutputParser for text output
- [x] Add progress indicator parsing to MinitestParser

### Integration Tasks
- [ ] Create NotificationAccumulator to collect events → [Accumulator spec](test-event-architecture.md#accumulator)
- [ ] Refactor RunRSpecFiles to use RSpecOutputParser → [Architecture](test-event-architecture.md#runner-integration)
- [ ] Refactor RunMinitestFiles to use MinitestOutputParser
- [ ] Extract common test execution logic to RunTestFiles
- [ ] Update StreamingResults to work with notifications

### Code Organization
- [ ] Fix package structure (notification types duplicated in 3 places)
- [ ] Move parsers to appropriate packages
- [ ] Consolidate duplicate type definitions
- [ ] Clean up dead code from refactoring

## Phase 4: Future Enhancements

- [ ] Add runtime tracking for better test distribution
- [ ] Support custom Minitest reporters
- [ ] Add Test::Unit support
- [ ] Add Cucumber support
- [ ] Optimize channel buffer sizes
- [ ] Add framework-specific configuration options

## Testing & Documentation

- [x] Integration tests for minitest success cases
- [x] Integration tests for minitest failure cases  
- [x] Unit tests for MinitestOutputParser
- [ ] Update user documentation for minitest usage
- [ ] Add minitest examples to README
- [ ] Document framework detection logic
- [ ] Add troubleshooting guide for minitest

## Known Issues to Address

- [ ] Type duplication between packages → [Issue details](minitest-implementation-guide.md#current-limitations)
- [ ] Parser integration incomplete → [Architecture](test-event-architecture.md#runner-integration)
- [ ] No error recovery for malformed output
- [ ] Limited minitest reporter support

## Next Steps

1. Create NotificationAccumulator implementation
2. Wire up parsers to runners
3. Test end-to-end with real minitest projects
4. Clean up package structure
5. Update documentation

---

Last updated: 2025-06-22