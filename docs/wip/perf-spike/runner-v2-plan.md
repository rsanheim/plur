# Plan: runner_v2.go - Clean Execution Architecture

## Goal

Create `runner_v2.go` with a cleaner architecture where:
- **Command** = data describing what to run - **important** this is just the go standard library's "exec.Command" struct
- **Worker** = thing that executes a Command
- **Dry-run seam** = right before `cmd.Start()`, not a high-level branch

## Success State
This comes **at the end**, after we have runner_v2 working and accepted.

```go
// In main.go or SpecCmd.Run()
runner := NewRunnerV2(cfg, testFiles, currentJob)
return runner.Run()  // handles dry-run vs real mode internally, with correct seam
```

## Architecture

```
Phase 1: PLAN (single-threaded, no goroutines)
  ├── Load runtime data
  ├── Group files into worker assignments
  └── For each group: build the Command ([]string args)

Phase 2: EXECUTE (the seam)
  ├── IF dry-run: print each command.String(), return
  └── ELSE: spawn workers to execute commands
            └── Workers receive Commands, NOT testFiles/globalConfig
```

## Key Design Decisions

1. **Commands are data** - Just `[]*exec.Cmd`, no wrapper struct needed
2. **Workers are executors** - Function that takes (ctx, index, cmd, outputChan)
3. **No config in workers** - All decisions made in Phase 1
4. **Single code path** - No `executeDryRun()` vs `executeTests()` split

## Implementation Steps

### Step 1: Copy runner.go → runner_v2.go

Start with existing code as base. Copy the `runner.go` file to `runner_v2.go` and refactor it to use the new architecture.

### Step 2: Create RunnerV2 struct

```go
type RunnerV2 struct {
    config     *config.GlobalConfig
    testFiles  []string
    job        job.Job
    tracker    *RuntimeTracker
}

func NewRunnerV2(cfg *config.GlobalConfig, testFiles []string, j job.Job) *RunnerV2
```

### Step 3: Implement Run() with clean phases

```go
func (r *RunnerV2) Run() error {
    // === PHASE 1: PLAN ===
    groups := r.groupFiles()           // load runtime, group by size/runtime
    commands := r.buildCommands(groups) // []*exec.Cmd

    r.printSummary(len(commands))

    // === PHASE 2: EXECUTE (the seam) ===
    if r.config.DryRun {
        for i, cmd := range commands {
            fmt.Fprintf(os.Stderr, "[dry-run] Worker %d: %s\n", i, strings.Join(cmd.Args, " "))
        }
        return nil
    }

    return r.executeWorkers(commands)
}
```

### Step 4: Implement groupFiles()

Consolidates LoadRuntimeData() + GroupSpecFilesByRuntime/Size.

### Step 5: Implement buildCommands()

Takes groups, returns `[]*exec.Cmd`. One command per group.

### Step 6: Implement executeWorkers()

```go
func (r *RunnerV2) executeWorkers(commands []*exec.Cmd) error {
    for i, cmd := range commands {
        go func(idx int, c *exec.Cmd) {
            // execute, stream output, send result
        }(i, cmd)
    }
    // collect results...
}
```

### Step 6.1: review new design for correctness and clarity
* can we remove code? do we have unnecessary abstractions?
* are there things we should inline based on the original runner.go?
* how do we feel about this? does it meet our goals?

### Step 7: Wire up in main.go

Replace current executor creation with RunnerV2.
Run full test suite (including ruby integration specs).
Run manual tests. Are there regressions?

### Step 8: Delete old code

Once RunnerV2 is working:
- Delete `executeDryRun()` from execution.go
- Delete or simplify TestExecutor
- Potentially consolidate execution.go into runner_v2.go

## Files to Modify

| File | Action |
|------|--------|
| `plur/runner_v2.go` | NEW - copy from runner.go, refactor |
| `plur/main.go` | Update SpecCmd.Run() to use RunnerV2 |
| `plur/execution.go` | Eventually delete or gut |
| `plur/runner.go` | Keep for reference, delete when v2 proven |

## Testing Strategy

1. Run existing integration tests against RunnerV2
2. Verify dry-run output matches current behavior
3. Verify real execution works identically
4. Once green, swap out old code

**Leverage dry-run for testing:** Since dry-run prints the commands that *would* run without executing them, we can use it to test:
- Command construction (correct args, flags, formatter paths)
- Test file splitting/grouping across workers
- Runtime-based vs size-based grouping behavior

This means we can write focused unit/integration tests that verify planning logic without needing real test files to execute. Dry-run output becomes a testable contract.

## Out of Scope (for now)

- Changing the grouping algorithms
- Changing output format
- Changing any logging output
- Changing RuntimeTracker behavior
- Watch mode (uses different entry point)
