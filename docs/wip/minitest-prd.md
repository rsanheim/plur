# Minitest PRD

## Overview

Rux is a fast parallel test runner that currently only supports RSpec. This PRD outlines the addition of Minitest support to expand Rux's usefulness to the broader Ruby community.

## Problem Statement

Many Ruby projects use Minitest as their testing framework. Without Minitest support, Rux cannot be adopted by these projects, limiting its impact and adoption in the Ruby ecosystem.

## Goals

1. **Enable Minitest parallel execution** - Run Minitest tests in parallel with the same performance benefits as RSpec
2. **Maintain framework parity** - Minitest users should have the same features as RSpec users
3. **Zero performance regression** - Existing RSpec functionality must not be impacted
4. **Clean architecture** - Support multiple frameworks through proper abstractions

## Success Criteria

- Minitest tests can be discovered and executed in parallel
- Real-time progress reporting works (dots, F, E, S indicators)
- Test output is properly captured and aggregated
- Failure reporting includes file paths and line numbers
- All existing RSpec functionality continues to work unchanged
- Architecture supports easy addition of future frameworks (Test::Unit, Cucumber)

## Technical Approach

### Phase 1: Framework Abstraction (✅ COMPLETED)
Create framework-agnostic types to support multiple test frameworks:
- Generic TestFile, TestResult types
- Framework detection and routing
- Command builder abstraction

[Deep dive: Framework abstraction design](test-event-architecture.md#framework-abstraction)

### Phase 2: Basic Minitest Support (✅ COMPLETED)  
Implement core Minitest functionality:
- Minitest file detection
- Command generation for parallel execution
- Output capture and streaming
- Integration tests

[Deep dive: Minitest implementation details](minitest-integration-details.md)

### Phase 3: Event-Based Refactoring (🔄 IN PROGRESS)
Refactor to use event-driven architecture for better extensibility:
- TestEvent notification system
- Framework-specific parsers (RSpec JSON, Minitest text)
- Event accumulator for test results
- Unified runner using parser abstraction

[Deep dive: Event architecture](test-event-architecture.md)

### Phase 4: Future Enhancements
- Runtime-based test distribution
- Test::Unit support
- Cucumber support
- Performance optimizations

## Non-Goals

- Supporting every Ruby test framework immediately
- Minitest-specific features not related to parallel execution
- Changing the existing RSpec user experience

## Dependencies

- Minitest must be installed in target projects
- Ruby version compatibility (same as current RSpec support)

## Risks and Mitigations

**Risk**: Breaking existing RSpec functionality
**Mitigation**: Comprehensive test coverage, phased rollout

**Risk**: Performance regression
**Mitigation**: Benchmarking, profiling, maintain streaming architecture

**Risk**: Incomplete Minitest compatibility  
**Mitigation**: Start with core features, iterate based on user feedback

## Timeline

- Phase 1: ✅ Completed
- Phase 2: ✅ Completed  
- Phase 3: In Progress (Q1 2025)
- Phase 4: Future (Q2 2025+)

## Outcome

When complete, Rux will support both RSpec and Minitest, significantly expanding its potential user base and providing value to more Ruby projects.