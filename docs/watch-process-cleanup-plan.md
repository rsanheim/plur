# Process Cleanup for plur watch

## Problem

When `plur watch` receives SIGINT/SIGTERM while test processes are running, those processes become orphaned. Two root causes:

1. **No cancellation mechanism**: `RunCommand` and `ExecuteJob` in `watcher.go` use `exec.Command` (no context). Once a test starts, nothing can stop it.
2. **Detached goroutines**: The debouncer fires callbacks via `time.AfterFunc` goroutines that block on `cmd.Run()`. When the main select loop exits on a signal, nothing cancels those goroutines or the child processes.
3. **No process group management**: rspec can spawn child processes (spring, database connections) that survive even if the top-level rspec process is killed.

### How it manifests

* User hits Ctrl+C during a test run in watch mode
* plur exits, watcher (C++ binary) processes get cleaned up via `defer manager.Stop()`
* But the rspec/ruby processes keep running as orphans
* Over long watch sessions, this accumulates

### Existing pattern in the codebase

`plur/database.go:18` already does it right:

```go
cmd := exec.CommandContext(ctx, "bundle", "exec", "rake", task)
```

We need to apply this pattern to watch mode execution.

## Scope

**In scope**: Watch mode process cleanup (`watch/` package and `cmd_watch.go`)

**Out of scope / follow-up**:
* Normal mode `runner.go` — same pattern but lower priority (user expects to wait)
* "Cancel and re-run" on file change — natural extension once context infra is in place
* Concurrent debounce runs — separate issue (debouncer can fire overlapping test runs)

## Solution: `cmd.Cancel` + `cmd.WaitDelay`

Go 1.20+ added `cmd.Cancel` and `cmd.WaitDelay` to `exec.Cmd`. Combined with `exec.CommandContext` and process groups, the shutdown sequence becomes:

1. Context is canceled (via `cancel()` in signal handler)
2. Go calls `cmd.Cancel` — our custom function that sends **SIGTERM** to the process group
3. Go waits up to `cmd.WaitDelay` (2s) for the process to exit gracefully
4. If still running, Go sends **SIGKILL** and force-kills
5. `cmd.Run()` returns, unblocking the debouncer goroutine

No orphaned processes. No sleep hacks.

## File-by-file changes

### 1. `plur/watch/watcher.go`

**New function — `setupProcessCleanup`**:

```go
// setupProcessCleanup configures a command for proper process group cleanup.
// Creates a process group and sets up graceful SIGTERM → SIGKILL escalation.
func setupProcessCleanup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Cancel = func() error {
		pgid := cmd.Process.Pid
		return syscall.Kill(-pgid, syscall.SIGTERM)
	}

	cmd.WaitDelay = 2 * time.Second
}
```

**Update `RunCommand` (line 212)** — add context, use CommandContext:

```go
// Before:
func RunCommand(args []string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
	}
}

// After:
func RunCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		return
	}
	fmt.Printf("\n[plur] %s\n", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setupProcessCleanup(cmd)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return
		}
		fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
	}
}
```

**Update `ExecuteJob` (line 229)** — add context, use CommandContext:

```go
// Before:
func ExecuteJob(j job.Job, targetFiles []string, cwd string) error {
	// ...
	execCmd := exec.Command(cmd[0], cmd[1:]...)
	// ...
}

// After:
func ExecuteJob(ctx context.Context, j job.Job, targetFiles []string, cwd string) error {
	// ...
	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Dir = cwd
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Env = append(os.Environ(), j.Env...)
	setupProcessCleanup(execCmd)

	if err := execCmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return err
		}
		logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
		return err
	}
	// ...
}
```

Both `exec.Command` call sites in `ExecuteJob` (lines 239 and 260) get the same treatment.

### 2. `plur/watch/file_event_handler.go`

**Update `JobExecutor` type (line 9)**:

```go
// Before:
type JobExecutor func(j job.Job, targets []string, cwd string) error

// After:
type JobExecutor func(ctx context.Context, j job.Job, targets []string, cwd string) error
```

**Add `Ctx` field to `FileEventHandler` (line 12)**:

```go
type FileEventHandler struct {
	Jobs    map[string]job.Job
	Watches []WatchMapping
	CWD     string
	Ctx     context.Context

	Executor JobExecutor
}
```

**Update executor call in `HandleBatch` (line 108)**:

```go
// Before:
if err := h.executor()(j, targets, h.CWD); err != nil {

// After:
ctx := h.Ctx
if ctx == nil {
	ctx = context.Background()
}
if err := h.executor()(ctx, j, targets, h.CWD); err != nil {
```

### 3. `plur/cmd_watch.go`

**Create cancelable context (~after line 209)**:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

// Cancel running jobs on shutdown
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

**Pass context to handler (~line 251)**:

```go
handler := &watch.FileEventHandler{
	Jobs:    jobs,
	Watches: watches,
	CWD:     cwd,
	Ctx:     ctx,
}
```

**Pass context to RunCommand (~line 265)**:

```go
case "":
	fmt.Println("Running all tests...")
	cmd := job.BuildJobAllCmd(resolvedJob)
	watch.RunCommand(ctx, cmd)
```

**Cancel before returning in signal handlers (~line 337)**:

```go
case syscall.SIGINT:
	fmt.Println("Received SIGINT, shutting down gracefully...")
	cancel()
	return nil
case syscall.SIGTERM:
	fmt.Println("Received SIGTERM, shutting down gracefully...")
	cancel()
	return nil
case syscall.SIGHUP:
	fmt.Println("Received SIGHUP, reloading plur...")
	cancel()
	if err := reload(manager); err != nil {
		// ...
	}
	return nil
```

### 4. `plur/watch/file_event_handler_test.go`

**Update mock signature (line 23)**:

```go
// Before:
func (m *mockExecutor) execute(j job.Job, targets []string, cwd string) error {

// After:
func (m *mockExecutor) execute(ctx context.Context, j job.Job, targets []string, cwd string) error {
```

## Summary of changes

| File | What changes |
|------|-------------|
| `plur/watch/watcher.go` | New `setupProcessCleanup()`, update `RunCommand()` and `ExecuteJob()` to accept `context.Context` and use `exec.CommandContext` |
| `plur/watch/file_event_handler.go` | `JobExecutor` type gets `ctx` param, `FileEventHandler` gets `Ctx` field, `HandleBatch` threads it through |
| `plur/cmd_watch.go` | Create cancelable context, pass to handler and RunCommand, cancel on signals |
| `plur/watch/file_event_handler_test.go` | Mock signature update (add `ctx` param) |

## Verification

* `go test ./plur/watch/...` — unit tests pass with updated signatures
* `bin/rake` — full build, lint, test suite
* Manual: start `plur watch`, trigger a test run, Ctrl+C, verify `pgrep -f rspec` finds nothing
* Existing integration specs in `spec/integration/plur_watch/` pass unchanged

## Future work (not in this PR)

* **Normal mode cleanup**: Apply same pattern to `runner.go:buildCommands()` and `runCommand()`
* **Cancel-and-rerun**: When a new file change arrives during a test run, cancel the current run and start a new one (classic guard/jest-watch behavior). The context infrastructure from this PR makes this straightforward.
* **Concurrent debounce guard**: The debouncer can fire overlapping test runs if `time.AfterFunc` triggers while a previous run is still going. A mutex or "cancel previous" pattern would fix this.
