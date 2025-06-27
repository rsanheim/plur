# Minitest Flow Sequence Diagram

This document provides a comprehensive view of how Rux processes Minitest test output, from the runner through all components including the parser, collector, and output aggregator.

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
                                Parser-->>Stream: [TestPassed], false
                            else char = 'F'
                                Parser-->>Stream: [TestFailed], false
                            else char = 'E'
                                Parser-->>Stream: [TestError], false
                            else char = 'S'
                                Parser-->>Stream: [TestPending], false
                            end
                        else Other lines
                            Parser-->>Stream: [], false
                        end
                        
                        Note over Stream: Always preserve raw output
                        Stream->>Collector: AddNotification(OutputNotification{line})
                        
                        loop For each notification
                            Stream->>Collector: AddNotification(notification)
                            
                            alt TestPassed
                                Stream->>OutChan: {Type: "dot", WorkerID: i}
                            else TestFailed/TestError
                                Stream->>OutChan: {Type: "failure", WorkerID: i}
                            else TestPending
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
            Collector-->>RunMini: TestResult{ExampleCount, FailureCount, Output, etc}
            
            RunMini->>RunMini: Convert to RSpec-compatible format
            Note over RunMini: Currently loses framework context
            
            RunMini-->>Worker: TestResult
        end
        
        Worker-->>Runner: Send result to results channel
    end
    
    Runner->>Runner: Close outputChan
    Runner->>Aggregator: Wait for completion
    Runner->>Console: Print newline after dots
    
    Runner->>Runner: Collect all results
    Runner-->>Executor: []TestResult, wallTime
    
    Executor->>Executor: BuildTestSummary(results)
    Executor->>Console: PrintResults(summary)
    
    Note over Console: Shows RSpec format:<br/>"X examples, Y failures"<br/>Not minitest format:<br/>"X runs, Y assertions..."
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
- Simple state machine with `testsRunning` flag
- Parses progress indicators (., F, E, S)
- Always returns `consumed=false` to preserve output
- Missing: Never calls `parseSummaryLine`

### 6. **TestCollector**
- Accumulates all notifications
- Tracks counts (passed, failed, pending)
- Stores raw output lines
- BuildResult creates final TestResult

### 7. **outputAggregator**
- Reads from outputChan
- Writes colored progress indicators to console
- Handles stderr output
- Runs concurrently with test execution

### 8. **PrintResults (result.go)**
- Formats final summary
- Currently hardcoded to RSpec style
- Doesn't know which framework was used
- Missing: Framework-aware formatting

## Current Issues

1. **Framework Context Lost**: By the time we reach PrintResults, we don't know if tests were RSpec or Minitest
2. **Summary Line Not Parsed**: The minitest summary ("X runs, Y assertions...") is captured but not parsed
3. **Output Format**: Always shows RSpec style regardless of framework
4. **TestError Handling**: TODO comment indicates uncertainty about 'E' indicator handling

## Potential Solutions

1. Add Framework field to TestResult or TestSummary
2. Call parseSummaryLine when appropriate
3. Make PrintResults framework-aware
4. Store and display raw minitest output for authentic experience