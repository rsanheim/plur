# Test Processing Flow

This document provides a comprehensive view of how Plur processes test output, from the runner through all components including the parser, collector, and output aggregator.

The architecture is framework-agnostic: RSpec and Minitest use the same flow with framework-specific parsers.

## Full System Flow - Test Execution

```mermaid
sequenceDiagram
    participant CLI as SpecCmd.Run
    participant Runner as Runner
    participant Worker as Worker Goroutine
    participant Cmd as exec.Command
    participant Stream as streamTestOutput
    participant Parser as Parser (RSpec/Minitest)
    participant Collector as TestCollector
    participant OutChan as outputChan
    participant Agg as outputAggregator
    participant Console as Console

    CLI->>Runner: NewRunner(config, files, job)
    CLI->>Runner: Run()

    Note over Runner: Phase 1: Planning
    Runner->>Runner: groupFiles() - by runtime or size
    Runner->>Runner: buildCommands() - exec.Cmd per group

    Note over Runner: Phase 2: Execution
    Runner->>OutChan: Create buffered channel
    Runner->>Agg: Start goroutine

    loop For each command
        Runner->>Worker: Spawn goroutine
        Worker->>Worker: runCommand(cmd, outputChan)

        Worker->>Cmd: StdoutPipe(), StderrPipe()
        Worker->>Cmd: Start()
        Worker->>Parser: job.CreateParser()
        Worker->>Collector: NewTestCollector()
        Worker->>Stream: streamTestOutput(stdout, stderr, parser, collector, outputChan)

        rect rgb(240, 248, 255)
            Note over Stream,Agg: Concurrent Output Processing

            par stdout goroutine
                loop For each line
                    Stream->>Parser: ParseLine(line)
                    Parser-->>Stream: notifications, consumed

                    alt !consumed (raw output like puts)
                        Stream->>Collector: AddNotification(RawOutput)
                        alt RSpec (streamStdout=true)
                            Stream->>OutChan: {Type: "stdout", Content: line}
                        end
                    end

                    loop For each notification
                        Stream->>Collector: AddNotification(n)
                        alt ProgressEvent
                            Stream->>OutChan: {Type: dot/failure/pending}
                        end
                    end
                end
            and stderr goroutine
                loop For each line
                    Stream->>OutChan: {Type: "stderr", Content: line}
                end
            and aggregator goroutine
                loop For each message
                    alt dot
                        Agg->>Console: Write green .
                    else failure
                        Agg->>Console: Write red F
                    else pending
                        Agg->>Console: Write yellow *
                    else stderr
                        Agg->>Console: Write to stderr
                    else stdout
                        Agg->>Console: Write to stdout
                    end
                end
            end
        end

        Stream-->>Worker: stderrOutput
        Worker->>Cmd: Wait()
        Worker->>Collector: BuildResult(duration)
        Collector-->>Worker: WorkerResult
    end

    Runner->>Runner: Close outputChan, wait for aggregator
    Runner->>Console: Print newline

    Note over Runner: Phase 3: Summary
    Runner-->>CLI: []WorkerResult, wallTime
    CLI->>CLI: BuildTestSummary(results)
    CLI->>Console: PrintResults(summary)
```

## Key Components

### 1. **Runner** (runner.go)
Orchestrates the entire test execution:
* `Run()` - Entry point with three phases: planning, execution, results
* `groupFiles()` - Groups files by runtime data (preferred) or file size (fallback)
* `buildCommands()` - Creates `exec.Cmd` for each file group with framework-specific args
* `executeWorkers()` - Spawns worker goroutines, manages channels, waits for completion
* `runCommand()` - Runs a single command: pipes, parser, collector, streamTestOutput

### 2. **streamTestOutput** (stream_helper.go)
Handles real-time output processing with two concurrent goroutines:
* **stdout goroutine**: Parses lines via `parser.ParseLine()`, sends progress to `outputChan`, collects raw output
  * For RSpec: Also streams unconsumed stdout (puts/pp) in real-time via `outputChan`
  * For Minitest: Raw stdout is collected but NOT streamed (Minitest returns consumed=false for all lines)
* **stderr goroutine**: Passes through to `outputChan` with type "stderr"

### 3. **Parser** (rspec/ or minitest/)
Framework-specific output parsing:
* `ParseLine(line)` returns `(notifications, consumed)`
* Emits `ProgressEvent` for dots/failures, `TestCaseNotification` for test results
* `FormatSummary()` for framework-native summary output

### 4. **TestCollector** (test_collector.go)
Accumulates notifications from parser:
* Tracks tests, failures, pending counts
* Stores raw output in `rawOutput` string builder
* `BuildResult()` creates final `WorkerResult`

### 5. **outputAggregator** (runner.go)
Single goroutine that serializes all output:
* Reads from `outputChan`
* Writes colored progress indicators (., F, *) to stdout
* Writes raw stdout (puts/pp output) to stdout (RSpec only)
* Writes stderr lines to stderr

### 6. **PrintResults** (result.go)
Displays final summary:
* `BuildTestSummary()` aggregates all WorkerResults
* Framework-aware formatting via `parser.FormatSummary()`
* Shows failures, then summary line

## Architecture Highlights

* **Framework-agnostic flow**: Same Runner/streaming for RSpec and Minitest
* **Channel-based output**: No lock contention, single aggregator serializes output
* **Event-based parsing**: Parsers emit typed notifications, not raw strings
* **Unified WorkerResult**: Each worker returns one result covering multiple test files