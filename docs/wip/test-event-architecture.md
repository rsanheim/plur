# Test Event Architecture

## Overview

The Test Event architecture refactors Rux's test execution to use an event-based abstraction that decouples framework-specific logic from generic test running concerns. This design enables support for multiple test frameworks while maintaining a clean separation of responsibilities.

## Current State Analysis

The existing `runner.go` implementation has tight RSpec coupling throughout. The `RunRSpecFiles` function combines multiple concerns:

```go
// Current implementation mixes:
// 1. RSpec-specific JSON parsing
if strings.HasPrefix(line, "RUX_JSON:") {
    jsonStr := strings.TrimPrefix(line, "RUX_JSON:")
    // Parse RSpec JSON format
}

// 2. RSpec-specific result accumulation
streamingResults.PassedFiles = append(streamingResults.PassedFiles, file)
streamingResults.ExampleCount += summary.ExampleCount

// 3. RSpec-specific failure extraction
failures := extractFailures(streamingResults)

// 4. RSpec-specific result building
return TestResult{
    Output:       buffer.String(),
    ExampleCount: streamingResults.ExampleCount,
    FailureCount: streamingResults.FailureCount,
}
```

This coupling creates three mixed concerns:
1. **Process Management** - Running commands, handling I/O
2. **Output Interpretation** - Framework-specific parsing
3. **Result Building** - Creating final test results

The event-based abstraction addresses each concern separately.

## Goals

1. Fix minitest output streaming (currently broken)
2. Create reusable abstractions for multiple test frameworks
3. Separate parsing, accumulation, and reporting concerns
4. Enable easier addition of new frameworks (Test::Unit, pytest, etc.)
5. Maintain real-time progress reporting across all frameworks

## Architecture Overview

### Data Flow

```
Raw Output Line → Parser → TestNotification(s) → Accumulator → Final Result
                     ↓
                 Progress Updates → Output Channel → Terminal Display
```

### Component Responsibilities

- **Parser**: Translates framework-specific output into generic notifications
- **Notifications**: Framework-agnostic event objects carrying test information
- **Accumulator**: Collects notifications and builds final test results
- **Runner**: Orchestrates execution and coordinates components

## Core Types

### Test Events

```go
// Event type enum - our internal representation
type TestEvent string

const (
    TestPassed    TestEvent = "test_passed"
    TestFailed    TestEvent = "test_failed"
    TestPending   TestEvent = "test_pending"
    TestStarted   TestEvent = "test_started"
    SuiteStarted  TestEvent = "suite_started"
    SuiteFinished TestEvent = "suite_finished"
    RawOutput     TestEvent = "raw_output"
)
```

### Notification Interface

```go
// Interface that all notifications implement
type TestNotification interface {
    GetEvent() TestEvent
    GetTestID() string
}
```

### Concrete Notification Types

#### 1. TestCaseNotification
For individual test cases (passed/failed/pending):

```go
type TestCaseNotification struct {
    Event           TestEvent  // Our enum (TestPassed, TestFailed, etc.)
    TestID          string     // Unique identifier
    Description     string
    FullDescription string
    Location        string     // e.g. "./spec/foo_spec.rb:42"
    FilePath        string
    LineNumber      int
    Status          string     // Original status from framework
    Duration        time.Duration
    
    // Only populated for failures
    Exception       *TestException
    
    // Only populated for pending tests
    PendingMessage  string
}

type TestException struct {
    Class     string
    Message   string
    Backtrace []string
}
```

#### 2. SuiteNotification
For suite-level events:

```go
type SuiteNotification struct {
    Event        TestEvent
    TestCount    int
    FailureCount int
    PendingCount int
    LoadTime     time.Duration
    Duration     time.Duration
}
```

#### 3. OutputNotification
For raw output that doesn't match patterns:

```go
type OutputNotification struct {
    Event   TestEvent // Always RawOutput
    Content string
}
```

## Parser Interface

```go
type TestOutputParser interface {
    // ParseLine processes a line of output and returns:
    // - notifications: Zero or more test events detected
    // - consumed: Whether the line was fully processed (should be hidden from output)
    ParseLine(line string) (notifications []TestNotification, consumed bool)
}
```

### Key Parser Concepts

1. **Single Responsibility**: Parsers only parse, they don't accumulate or format
2. **Stateful Parsing**: Parsers may maintain state between lines (e.g., for multi-line failures)
3. **Consumption Model**: Parsers indicate whether a line should be hidden from final output
4. **Multiple Notifications**: A single line may produce multiple notifications

## Framework Implementation Examples

### RSpec Parser

RSpec outputs structured JSON that maps cleanly to our notification types:

```go
type RSpecOutputParser struct {
    testCounter int
}

func (p *RSpecOutputParser) ParseLine(line string) ([]TestNotification, bool) {
    if strings.HasPrefix(line, "RUX_JSON:") {
        jsonStr := strings.TrimPrefix(line, "RUX_JSON:")
        // Parse JSON and create appropriate notifications
        // Return notifications and true (line consumed)
    }
    
    // Non-JSON lines become OutputNotifications
    return []TestNotification{OutputNotification{Event: RawOutput, Content: line}}, false
}
```

Type Mappings:
- `"example_passed"` → `TestPassed` event
- `"example_failed"` → `TestFailed` event
- `"example_pending"` → `TestPending` event
- `"load_summary"` → `SuiteStarted` event
- `"dump_summary"` → `SuiteFinished` event

### Minitest Parser

Minitest uses text-based output requiring stateful parsing:

```go
type MinitestOutputParser struct {
    currentTest     string
    currentLocation string
    inFailure       bool
    failureBuffer   strings.Builder
    testCounter     int
}

func (p *MinitestOutputParser) ParseLine(line string) ([]TestNotification, bool) {
    // Handle progress indicators (., F, E, S)
    // Buffer multi-line failure messages
    // Parse summary lines
    // Return notifications and false (preserve output)
}
```

## Accumulator Design {#accumulator}

The TestCollector collects notifications and builds the final test result:

```go
type TestCollector struct {
    tests        []TestCaseNotification
    failures     []TestCaseNotification
    pending      []TestCaseNotification
    suiteInfo    *SuiteNotification
    rawOutput    strings.Builder
}

func (a *TestCollector) AddNotification(n TestNotification) {
    switch n.GetEvent() {
    case TestPassed, TestFailed, TestPending:
        if tc, ok := n.(TestCaseNotification); ok {
            a.tests = append(a.tests, tc)
            if n.GetEvent() == TestFailed {
                a.failures = append(a.failures, tc)
            }
        }
    case SuiteFinished:
        if suite, ok := n.(SuiteNotification); ok {
            a.suiteInfo = &suite
        }
    case RawOutput:
        if out, ok := n.(OutputNotification); ok {
            a.rawOutput.WriteString(out.Content + "\n")
        }
    }
}

func (a *TestCollector) BuildResult(file string, duration time.Duration) TestResult {
    return TestResult{
        File:         file,
        Output:       a.rawOutput.String(),
        Duration:     duration,
        Failures:     a.failures,
        ExampleCount: len(a.tests),
        FailureCount: len(a.failures),
        PendingCount: len(a.pending),
    }
}
```

## Runner Integration

The generic runner orchestrates all components:

```go
func RunTestFiles(ctx context.Context, config *Config, files []string, 
                  workerIndex int, outputChan chan<- OutputMessage) TestResult {
    
    // Select parser based on framework
    var parser TestOutputParser
    switch config.Framework {
    case FrameworkRSpec:
        parser = &RSpecOutputParser{}
    case FrameworkMinitest:
        parser = &MinitestOutputParser{}
    }
    
    // Create accumulator
    accumulator := NewTestCollector()
    
    // Setup command and pipes for streaming
    cmd := buildTestCommand(config, files)
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    
    // Start command
    cmd.Start()
    
    // Process output line by line
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        line := scanner.Text()
        
        // Parse line into notifications
        notifications, consumed := parser.ParseLine(line)
        
        // Process each notification
        for _, notification := range notifications {
            accumulator.AddNotification(notification)
            
            // Send real-time progress updates
            switch notification.GetEvent() {
            case TestPassed:
                outputChan <- OutputMessage{WorkerID: workerIndex, Type: "dot"}
            case TestFailed:
                outputChan <- OutputMessage{WorkerID: workerIndex, Type: "failure"}
            case TestPending:
                outputChan <- OutputMessage{WorkerID: workerIndex, Type: "pending"}
            }
        }
    }
    
    // Build and return final result
    return accumulator.BuildResult(files[0], time.Since(start))
}
```

## Implementation Phases

### Phase 1: Core Types ✅ COMPLETE
- Created notifications.go with event types and interfaces
- Created parser.go with TestOutputParser interface

### Phase 2: Framework Parsers ✅ COMPLETE
- Implemented RSpec JSON parser
- Implemented Minitest text parser
- Added comprehensive unit tests

### Phase 3: Accumulator 🚧 IN PROGRESS
- Create TestCollector struct
- Implement notification collection logic
- Build TestResult from accumulated data

### Phase 4: Fix Minitest Streaming ✅ COMPLETE
- Switched from CombinedOutput to streaming pipes
- Fixed real-time progress reporting
- Aligned with RSpec implementation

### Phase 5: Refactor Runner 🚧 TODO
- Extract common logic to generic RunTestFiles
- Update framework-specific functions to thin wrappers
- Consolidate command building

### Phase 6: Package Cleanup 🚧 TODO
- Resolve type duplication between packages
- Organize imports and package structure
- Consider unified parsing package

## Benefits

1. **Framework Agnostic**: Core logic works with any test framework
2. **Real-time Feedback**: Progress updates stream as tests run
3. **Clean Separation**: Parsing, accumulation, and reporting are independent
4. **Easy Extension**: Adding new frameworks requires only a new parser
5. **Testable**: Each component can be tested in isolation
6. **Preserves Information**: Original framework data is maintained alongside our abstractions

## Migration Strategy

The refactoring follows a phased approach to minimize risk:

### Phase 1: Create Abstractions
- Define interfaces and types without changing existing code
- Implement parsers and accumulator alongside current implementation
- Add comprehensive tests for new components

### Phase 2: Implement for RSpec  
- Update RunRSpecFiles to use new parser
- Verify identical behavior with existing tests
- Keep old code as fallback

### Phase 3: Fix Minitest
- Implement streaming with new parser
- Update RunMinitestFiles to match
- Verify all integration tests pass

### Phase 4: Remove Old Code
- Delete old parsing logic
- Clean up package structure
- Document new architecture

## Future Considerations

1. **Performance**: Event creation overhead is minimal compared to test execution
2. **Memory**: Accumulator stores all notifications but this matches current behavior
3. **Error Handling**: Parser errors should not crash the runner
4. **Configuration**: Parser behavior could be configurable (verbosity, etc.)
5. **Plugins**: Event system could enable plugin architecture for custom reporters