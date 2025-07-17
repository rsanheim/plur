# Minitest Flow Sequence Diagram

This document provides a comprehensive view of how Plur processes Minitest test output, from the runner through all components including the parser, collector, and output aggregator.

**Updated**: Reflects current architecture with WorkerResult, ProgressEvent, unified test representation, and framework-aware formatting.

## Full System Flow - Minitest Execution

!!! tip "Viewing Large Diagrams"
    This diagram supports pan and zoom! Use your mouse wheel to zoom in/out and drag to pan around. Double-click to reset the view.

```mermaid
sequenceDiagram
    participant Main as Main/CLI
    participant Executor as TestExecutor
    participant Runner as RunSpecsInParallel
    participant Worker as Worker Goroutine
    participant RunSpec as RunSpecFile
    participant RunMini as RunMinitestFiles
    participant Cmd as exec.Command
    participant Stream as streamTestOutput
    participant Parser as MinitestParser
    participant Collector as TestCollector
    participant OutChan as outputChan
    participant Aggregator as outputAggregator
    participant Console as Console Output

    Main->>Executor: Execute(config, specFiles)
    Executor->>Runner: RunSpecsInParallel(config, specFiles)
    
    Note over Runner: Group files by runtime/size
    Runner->>Runner: Create outputChan (buffered)
    Runner->>Aggregator: Start outputAggregator goroutine
    
    Runner->>Worker: Launch worker goroutines
    
    loop For each file group
        Worker->>RunSpec: RunSpecFile(files, outputChan)
        
        alt Framework = Minitest
            RunSpec->>RunMini: RunMinitestFiles(files, outputChan)
            
            RunMini->>RunMini: Build command args
            Note over RunMini: ruby -Itest test1.rb test2.rb...
            
            RunMini->>Cmd: Create & start process
            RunMini->>Cmd: Set up stdout/stderr pipes
            
            RunMini->>Parser: NewTestOutputParser(Minitest)
            RunMini->>Collector: NewTestCollector()
            
            RunMini->>Stream: streamTestOutput(stdout, stderr, parser, collector, outputChan)
            
            rect rgb(240, 240, 255)
                Note over Stream,Aggregator: Concurrent Processing
                
                par Process stdout
                    loop For each line
                        Stream->>Parser: ParseLine(line)
                        
                        alt Line = "# Running:"
                            Parser->>Parser: testsRunning = true
                            Parser-->>Stream: [SuiteStarted], false
                        else Line = "Finished in..."
                            Parser->>Parser: testsRunning = false
                            Parser-->>Stream: [], false
                        else testsRunning && has progress chars
                            Parser->>Parser: parseProgressLine()
                            alt char = '.'
                                Parser-->>Stream: [ProgressEvent{'.'}], false
                            else char = 'F'
                                Parser-->>Stream: [ProgressEvent{'F'}], false
                            else char = 'E'
                                Parser-->>Stream: [ProgressEvent{'E'}], false
                            else char = 'S'
                                Parser-->>Stream: [ProgressEvent{'S'}], false
                            end
                        else Other lines
                            Parser-->>Stream: [], false
                        end
                        
                        Note over Stream: Always preserve raw output
                        Stream->>Collector: AddNotification(OutputNotification{line})
                        
                        loop For each notification
                            Stream->>Collector: AddNotification(notification)
                            
                            alt ProgressEvent with '.'
                                Stream->>OutChan: {Type: "dot", WorkerID: i}
                            else ProgressEvent with 'F' or 'E'
                                Stream->>OutChan: {Type: "failure", WorkerID: i}
                            else ProgressEvent with 'S'
                                Stream->>OutChan: {Type: "pending", WorkerID: i}
                            end
                        end
                    end
                and Process stderr
                    loop For each line
                        Stream->>OutChan: {Type: "stderr", Content: line}
                    end
                and Aggregate output
                    loop For each message
                        Aggregator->>Aggregator: Read from outputChan
                        alt Type = "dot"
                            Aggregator->>Console: Write green dot
                        else Type = "failure"
                            Aggregator->>Console: Write red F
                        else Type = "pending"
                            Aggregator->>Console: Write yellow *
                        else Type = "stderr"
                            Aggregator->>Console: Write to stderr
                        end
                    end
                end
            end
            
            Stream-->>RunMini: Return stderr output
            
            RunMini->>Cmd: Wait for completion
            RunMini->>Collector: BuildResult(testFile, duration)
            
            Note over Collector: Accumulate counts from notifications
            Collector-->>RunMini: WorkerResult{ExampleCount, FailureCount, Output, etc}
            
            RunMini->>RunMini: Convert to RSpec-compatible format
            Note over RunMini: Preserves framework context
            
            RunMini-->>Worker: WorkerResult
        end
        
        Worker-->>Runner: Send result to results channel
    end
    
    Runner->>Runner: Close outputChan
    Runner->>Aggregator: Wait for completion
    Runner->>Console: Print newline after dots
    
    Runner->>Runner: Collect all results
    Runner-->>Executor: []WorkerResult, wallTime
    
    Executor->>Executor: BuildTestSummary(results)
    Note over Executor: Summary includes Framework field
    Executor->>Console: PrintResults(summary)
    
    alt Framework = Minitest
        Console->>Parser: NewTestOutputParser(Minitest)
        Parser->>Parser: FormatSummary(...)
        Console->>Console: Display minitest format:<br/>"X runs, Y assertions, Z failures..."
    else Framework = RSpec
        Console->>Console: Display RSpec format:<br/>"X examples, Y failures"
    end
```

## Key Components

### 1. **RunSpecsInParallel**
- Groups test files by runtime or size
- Creates output channel for progress updates
- Launches worker goroutines
- Starts output aggregator

### 2. **Worker Goroutines**
- Each worker processes a group of files
- Calls RunSpecFile which dispatches to framework-specific runner

### 3. **RunMinitestFiles**
- Builds minitest command (`ruby -Itest ...`)
- Creates pipes for stdout/stderr
- Instantiates parser and collector
- Calls streamTestOutput

### 4. **streamTestOutput (stream_helper.go)**
- Runs two concurrent goroutines:
  - One for stdout (parsing)
  - One for stderr (pass-through)
- Always preserves raw output as OutputNotification
- Sends progress indicators to outputChan

### 5. **MinitestParser (minitest/output_parser.go)**
- State machine with multiple states (parsingProgress, afterFinished, inFailureDetails)
- Emits ProgressEvent for real-time display (., F, E, S)
- Parses failure details after test execution
- Creates complete TestCaseNotification objects with failure information
- Implements FormatSummary for framework-specific output

### 6. **TestCollector**
- Accumulates all notifications
- Separates progress tracking from test results
- Stores complete TestCaseNotification objects
- Stores raw output lines
- BuildResult creates final WorkerResult with framework context

### 7. **outputAggregator**
- Reads from outputChan
- Writes colored progress indicators to console
- Handles stderr output
- Runs concurrently with test execution

### 8. **PrintResults (result.go)**
- Framework-aware formatting
- Uses parser.FormatSummary for framework-specific output
- Minitest shows: "X runs, Y assertions, Z failures, W errors, V skips"
- RSpec shows: "X examples, Y failures"
- Falls back to generic formatting if parser unavailable

## Key Improvements Since Initial Implementation

1. **Framework Context Preserved**: WorkerResult and TestSummary include Framework field
2. **Unified Test Representation**: Single TestCaseNotification type for all test results
3. **Progress Events**: Separate ProgressEvent type for real-time display without duplication
4. **Framework-Aware Formatting**: PrintResults uses parser.FormatSummary for native output
5. **Complete Failure Parsing**: Minitest parser extracts full failure details
6. **Better Naming**: TestResult renamed to WorkerResult to clarify it represents multiple files from one worker

## Current Architecture Highlights

1. **Event-Based Design**: Clean separation between parsing, accumulation, and display
2. **Parser Factory Pattern**: Framework-specific parsers created via factory
3. **Shared Streaming Logic**: Common output streaming for both frameworks
4. **Tell Don't Ask**: Parsers tell the runner how to format output
5. **No Type Proliferation**: Eliminated redundant TestFailure type