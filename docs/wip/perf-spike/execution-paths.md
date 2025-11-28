# Execution Paths Analysis

Analysis of duplicate code paths in plur's test execution flow.

## Summary

The plur codebase has **two separate code paths** that independently load runtime data and group test files:

* **Path A**: `executeDryRun()` in `execution.go` (for `--dry-run` flag)
* **Path B**: `RunTestsInParallel()` in `runner.go` (for real execution)

This creates:

* Duplicated logic (DRY violation)
* Inconsistent log messages
* Risk of the two paths diverging

## Architecture Diagram

```mermaid
flowchart TD
    subgraph CLI["CLI Entry"]
        main["main()"] --> kong["Kong CLI Parsing"]
        kong --> afterApply["PlurCLI.AfterApply()"]
        afterApply --> specCmd["SpecCmd.Run()"]
    end

    subgraph Setup["Test Setup"]
        specCmd --> resolveJob["autodetect.ResolveJob()"]
        resolveJob --> findFiles["FindFilesFromJob()"]
        findFiles --> newExecutor["NewTestExecutor()"]
        newExecutor --> newTracker["NewRuntimeTracker()"]
        newTracker --> execute["executor.Execute()"]
    end

    execute --> dryRunCheck{{"globalConfig.DryRun?"}}

    subgraph DryRunPath["DRY-RUN PATH (execution.go)"]
        dryRunCheck -->|Yes| dryRun["executeDryRun()"]
        dryRun --> loadRT1["LoadRuntimeData()"]
        loadRT1 --> groupCheck1{{"len(runtimeData) > 0?"}}
        groupCheck1 -->|Yes| groupRT1["GroupSpecFilesByRuntime()"]
        groupCheck1 -->|No| groupSize1["GroupSpecFilesBySize()"]
        groupRT1 --> log1["Log: 'runtime-based grouped execution'"]
        groupSize1 --> log2["Log: 'size-based grouped execution'"]
        log1 --> printCmds["Print Worker Commands"]
        log2 --> printCmds
        printCmds --> returnNil["return nil"]
    end

    subgraph RealPath["REAL EXECUTION PATH (runner.go)"]
        dryRunCheck -->|No| realExec["executeTests()"]
        realExec --> runParallel["RunTestsInParallel()"]
        runParallel --> loadRT2["LoadRuntimeData()"]
        loadRT2 --> groupCheck2{{"len(runtimeData) > 0?"}}
        groupCheck2 -->|Yes| groupRT2["GroupSpecFilesByRuntime()"]
        groupCheck2 -->|No| groupSize2["GroupSpecFilesBySize()"]
        groupRT2 --> log3["Log: 'runtime-based grouping'"]
        groupSize2 --> log4["Log: 'size-based grouping'"]
        log3 --> launchWorkers["Launch Worker Goroutines"]
        log4 --> launchWorkers
        launchWorkers --> runTestFiles["RunTestFiles() per worker"]
        runTestFiles --> collectResults["Collect Results"]
        collectResults --> saveRT["runtimeTracker.SaveToFile()"]
        saveRT --> printResults["PrintResults()"]
    end

```

## Call Graph

```
main()
  └─→ Kong CLI parsing
      └─→ PlurCLI.AfterApply()
      └─→ SpecCmd.Run()
          ├─→ autodetect.ResolveJob()
          ├─→ FindFilesFromJob()
          ├─→ NewTestExecutor()
          │   └─→ NewRuntimeTracker() [ALWAYS created, unused in dry-run]
          │
          └─→ TestExecutor.Execute()
              │
              ├─→ IF globalConfig.DryRun ────────────────────────────┐
              │   └─→ executeDryRun()                                │
              │       ├─→ LoadRuntimeData()        ◄── CALL #1       │
              │       ├─→ GroupSpecFilesByRuntime()                  │ PATH A
              │       │   OR GroupSpecFilesBySize()                  │
              │       ├─→ Log: "Using X-based grouped execution"     │
              │       └─→ Print commands, return                     │
              │                                                      │
              └─→ ELSE ──────────────────────────────────────────────┤
                  └─→ executeTests()                                 │
                      └─→ RunTestsInParallel()                       │
                          ├─→ LoadRuntimeData()    ◄── CALL #2       │ PATH B
                          ├─→ GroupSpecFilesByRuntime()              │
                          │   OR GroupSpecFilesBySize()              │
                          ├─→ Log: "Using X-based grouping"          │
                          ├─→ Launch workers, run tests              │
                          └─→ Collect results                        │
                                                                     │
                      └─→ runtimeTracker.SaveToFile()                │
                      └─→ PrintResults()                             ┘
```

## Duplication Points

### 1. Runtime Data Loading

| Location | File:Line | When Called |
|----------|-----------|-------------|
| CALL #1 | `execution.go:72` | `--dry-run` only |
| CALL #2 | `runner.go:268` | Real execution only |

Both call `LoadRuntimeData()` which reads `~/.plur/runtime/<project-hash>.json`.

### 2. Grouping Decision Logic

Both paths have identical if/else:

```go
// execution.go:79-85
if len(runtimeData) > 0 {
    groups = GroupSpecFilesByRuntime(testFiles, WorkerCount, runtimeData)
} else {
    groups = GroupSpecFilesBySize(testFiles, WorkerCount)
}

// runner.go:279-285
if len(runtimeData) > 0 {
    groups = GroupSpecFilesByRuntime(testFiles, maxWorkers, runtimeData)
} else {
    groups = GroupSpecFilesBySize(testFiles, maxWorkers)
}
```

### 3. Inconsistent Log Messages

| Path | Strategy | Message |
|------|----------|---------|
| Dry-run | Runtime | "Using runtime-based grouped execution" |
| Dry-run | Size | "Using size-based grouped execution" |
| Real | Runtime | "Using runtime-based grouping" |
| Real | Size | "Using size-based grouping (no runtime data available)" |

### 4. RuntimeTracker Creation

`NewTestExecutor()` always creates a `RuntimeTracker`, but `executeDryRun()` never uses it.

## Root Cause

The duplication happened because `executeDryRun()` was added as a **completely separate code path** rather than as a mode within the existing execution flow.

## Future Fix: Extract ExecutionPlan

The recommended fix is to extract a shared `ExecutionPlan` struct:

```go
type ExecutionPlan struct {
    Groups      []FileGroup
    RuntimeData map[string]float64
    Strategy    string // "runtime" or "size"
}

func BuildExecutionPlan(testFiles []string, workerCount int) (*ExecutionPlan, error) {
    // Single place for:
    // 1. Loading runtime data
    // 2. Grouping files
    // 3. Logging strategy
}
```

Then both `executeDryRun()` and `RunTestsInParallel()` would call this single function, eliminating the duplication.
