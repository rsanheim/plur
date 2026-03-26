# Watch Terminal Abstraction Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a small watch-scoped terminal abstraction that owns prompt visibility and synchronized watch-mode output without turning `plur watch` into a full TUI.

**Architecture:** Add a focused terminal owner in the `watch` package with generic prompt and print operations plus `io.Writer` adapters for stdout/stderr. Route watch-mode output through that owner, including watch-local logger output and command echo paths, while keeping `plur spec` and the rest of the CLI unchanged.

**Tech Stack:** Go, `io.Writer`, `log/slog`, testify

---

## File Structure

| File | Change | Purpose |
|------|--------|---------|
| `docs/plans/2026-03-26-watch-terminal-spike-plan.md` | Create | Record the implementation plan for the spike |
| `watch/terminal.go` | Create | Own watch prompt visibility, synchronized line printing, and writer adapters |
| `watch/terminal_test.go` | Create | TDD coverage for prompt suspension, line printing, and writer behavior |
| `watch/watcher.go` | Modify | Route command echo and watcher stderr through the terminal abstraction |
| `watch/watcher_test.go` | Modify | Update execution tests to use the new terminal-aware job execution path |
| `watch/file_event_handler.go` | Modify | Pass the terminal abstraction into job execution |
| `watch/file_event_handler_test.go` | Modify | Verify handler-driven execution still works with terminal injection |
| `watch/debouncer.go` | Optional small modify | Only if a small callback signature adjustment is needed |
| `cmd_watch.go` | Modify | Instantiate the watch terminal, use generic print operations, and keep prompt redraw localized |
| `cmd_watch_test.go` | Modify | Add small coverage for terminal-aware watch setup when practical |
| `logger/logger.go` | Modify | Add a small way to temporarily redirect the structured logger writer during watch mode |

## Design Constraints

- Keep scope limited to `plur watch`; do not change `plur spec` output behavior.
- Keep the abstraction generic. It should manage prompt visibility and formatting, not encode watch business semantics such as "job start" or "reload" as separate terminal concepts.
- Prefer line-oriented behavior over raw-mode editing in this spike.
- Follow simple library patterns rather than importing a full TUI dependency:
  - Bubble Tea pattern to follow: one runtime owns rendering and synchronized writes
  - Readline pattern to follow: prompt visibility and print-above-prompt are explicit operations
  - Simpler choice for this spike: internal abstraction only, no new third-party dependency

## Proposed API Shape

The spike should converge on a small API like:

```go
type Terminal struct { /* private fields */ }

func NewTerminal(stdout io.Writer, stderr io.Writer, prompt string) *Terminal

func (t *Terminal) ShowPrompt()
func (t *Terminal) SuspendPrompt()
func (t *Terminal) PrintLine(text string)
func (t *Terminal) Print(text string)
func (t *Terminal) Stdout() io.Writer
func (t *Terminal) Stderr() io.Writer
```

This is intentionally generic:

- `ShowPrompt` and `SuspendPrompt` manage prompt state
- `PrintLine` prints a full line above the prompt
- `Print` handles literal text that is already formatted
- `Stdout()` / `Stderr()` expose synchronized writer adapters for subprocess and logger integration

No `PrintJobStart`, `PrintJobEnd`, or `PrintReload` methods should exist in the terminal type.

### Task 1: Add the failing terminal tests

**Files:**
- Create: `watch/terminal_test.go`

- [ ] **Step 1: Write a failing test for prompt rendering**

```go
func TestTerminal_ShowPromptWritesPromptOnce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()

	assert.Equal(t, "[plur] > ", stdout.String())
}
```

- [ ] **Step 2: Write a failing test for suspending an active prompt before line output**

```go
func TestTerminal_PrintLineSuspendsPromptBeforeWriting(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()
	terminal.PrintLine("bundle exec rspec spec/foo_spec.rb")

	assert.Equal(t, "[plur] > \nbundle exec rspec spec/foo_spec.rb\n", stdout.String())
}
```

- [ ] **Step 3: Write a failing test for synchronized stderr writer behavior**

```go
func TestTerminal_StderrWriterSuspendsPromptBeforeWrite(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()

	_, err := io.WriteString(terminal.Stderr(), "04:33:19 - DEBUG - watch type=\"watcher\"\n")
	require.NoError(t, err)

	assert.Equal(t, "[plur] > \n", stdout.String())
	assert.Equal(t, "04:33:19 - DEBUG - watch type=\"watcher\"\n", stderr.String())
}
```

- [ ] **Step 4: Run the new tests to verify RED**

Run: `go test ./watch -run 'TestTerminal_' -v`
Expected: FAIL with `undefined: NewTerminal`

- [ ] **Step 5: Commit the red test file if desired for checkpointing**

```bash
git add watch/terminal_test.go
git commit -m "test: add failing watch terminal abstraction tests"
```

### Task 2: Implement the terminal abstraction minimally

**Files:**
- Create: `watch/terminal.go`
- Test: `watch/terminal_test.go`

- [ ] **Step 1: Write the minimal terminal implementation**

```go
package watch

import (
	"fmt"
	"io"
	"sync"
)

type Terminal struct {
	mu            sync.Mutex
	stdout        io.Writer
	stderr        io.Writer
	prompt        string
	promptVisible bool
}

func NewTerminal(stdout io.Writer, stderr io.Writer, prompt string) *Terminal {
	return &Terminal{
		stdout: stdout,
		stderr: stderr,
		prompt: prompt,
	}
}

func (t *Terminal) ShowPrompt() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.promptVisible {
		return
	}

	fmt.Fprint(t.stdout, t.prompt)
	t.promptVisible = true
}

func (t *Terminal) SuspendPrompt() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.suspendPromptLocked()
}

func (t *Terminal) Print(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.suspendPromptLocked()
	fmt.Fprint(t.stdout, text)
}

func (t *Terminal) PrintLine(text string) {
	t.Print(text + "\n")
}

func (t *Terminal) Stdout() io.Writer {
	return terminalWriter{terminal: t, target: terminalStdout}
}

func (t *Terminal) Stderr() io.Writer {
	return terminalWriter{terminal: t, target: terminalStderr}
}

func (t *Terminal) suspendPromptLocked() {
	if !t.promptVisible {
		return
	}

	fmt.Fprint(t.stdout, "\n")
	t.promptVisible = false
}
```

- [ ] **Step 2: Add the writer adapter type**

```go
type terminalTarget int

const (
	terminalStdout terminalTarget = iota
	terminalStderr
)

type terminalWriter struct {
	terminal *Terminal
	target   terminalTarget
}

func (w terminalWriter) Write(p []byte) (int, error) {
	w.terminal.mu.Lock()
	defer w.terminal.mu.Unlock()

	w.terminal.suspendPromptLocked()

	switch w.target {
	case terminalStdout:
		return w.terminal.stdout.Write(p)
	default:
		return w.terminal.stderr.Write(p)
	}
}
```

- [ ] **Step 3: Run the terminal tests to verify GREEN**

Run: `go test ./watch -run 'TestTerminal_' -v`
Expected: PASS

- [ ] **Step 4: Refactor only if needed, then keep tests green**

Run: `go test ./watch -run 'TestTerminal_' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add watch/terminal.go watch/terminal_test.go
git commit -m "feat: add watch terminal abstraction"
```

### Task 3: Redirect watch-local logger output through the terminal

**Files:**
- Modify: `logger/logger.go`
- Modify: `cmd_watch.go`

- [ ] **Step 1: Write a failing logger redirection test**

Add a focused test in `logger/logger_test.go` or `cmd_watch_test.go` that verifies a redirected logger writes to a provided buffer.

```go
func TestSetWriterRedirectsStructuredLogs(t *testing.T) {
	var buf bytes.Buffer

	restore := SetWriter(&buf)
	defer restore()

	Logger.Debug("watch", "type", "watcher")
	assert.Contains(t, buf.String(), `type="watcher"`)
}
```

- [ ] **Step 2: Run the targeted test to verify RED**

Run: `go test ./logger -run TestSetWriterRedirectsStructuredLogs -v`
Expected: FAIL with `undefined: SetWriter`

- [ ] **Step 3: Add a small writer override helper**

Implement a helper that swaps `Logger` to a new `slog.Logger` backed by the provided writer and returns a restore closure.

- [ ] **Step 4: Update `cmd_watch.go` to install a terminal-backed logger during watch mode**

Use:

```go
terminal := watch.NewTerminal(os.Stdout, os.Stderr, "[plur] > ")
restoreLogger := logger.SetWriter(terminal.Stderr())
defer restoreLogger()
```

- [ ] **Step 5: Run the logger test to verify GREEN**

Run: `go test ./logger -run TestSetWriterRedirectsStructuredLogs -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add logger/logger.go logger/logger_test.go cmd_watch.go
git commit -m "feat: route watch logs through terminal output"
```

### Task 4: Route watch command echo and watcher stderr through the terminal

**Files:**
- Modify: `watch/watcher.go`
- Modify: `watch/watcher_test.go`
- Modify: `watch/file_event_handler.go`
- Modify: `watch/file_event_handler_test.go`

- [ ] **Step 1: Write failing tests for terminal-aware command echo**

Add tests that verify the command echo is printed via the terminal abstraction and starts on a fresh line if a prompt is visible.

```go
func TestExecuteJob_PrintsCommandViaTerminal(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "ran.txt")

	j := job.Job{
		Name: "test-no-placeholder",
		Cmd:  []string{"sh", "-c", "echo executed > " + outputFile},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()

	err := ExecuteJob(j, []string{"ignored.rb"}, tmpDir, terminal)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "\nsh -c echo executed > ")
}
```

- [ ] **Step 2: Run the targeted watch tests to verify RED**

Run: `go test ./watch -run 'TestExecuteJob_|TestFileEventHandler_' -v`
Expected: FAIL due to the old `ExecuteJob` signature

- [ ] **Step 3: Change `ExecuteJob` to accept a terminal**

Target signature:

```go
func ExecuteJob(j job.Job, targetFiles []string, cwd string, terminal *Terminal) error
```

Behavior:

- if `terminal != nil`, use `terminal.PrintLine(...)` for command echo
- if `terminal != nil`, set `execCmd.Stdout = terminal.Stdout()` and `execCmd.Stderr = terminal.Stderr()`
- if `terminal == nil`, preserve current default stdout/stderr behavior

- [ ] **Step 4: Update `FileEventHandler` to carry the terminal**

Add:

```go
Terminal *Terminal
```

and pass it into `ExecuteJob`.

- [ ] **Step 5: Route watcher stderr through a configurable writer**

Add a writer field to `Watcher` and default it to `os.Stderr`. Use the terminal-backed stderr writer in watch mode.

- [ ] **Step 6: Run the targeted watch tests to verify GREEN**

Run: `go test ./watch -run 'TestExecuteJob_|TestFileEventHandler_' -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add watch/watcher.go watch/watcher_test.go watch/file_event_handler.go watch/file_event_handler_test.go
git commit -m "feat: route watch command output through terminal abstraction"
```

### Task 5: Update the watch loop to use generic terminal operations

**Files:**
- Modify: `cmd_watch.go`
- Test: `cmd_watch_test.go`

- [ ] **Step 1: Write a failing test for prompt suspension before watcher lifecycle logs**

Add a small test around the terminal-backed logger or extracted helper to cover the original startup interleaving case.

- [ ] **Step 2: Run the targeted watch-loop test to verify RED**

Run: `go test . -run 'Test.*Watch.*Prompt|Test.*Watch.*Logger' -v`
Expected: FAIL for the new case

- [ ] **Step 3: Replace direct watch-mode `fmt.Print*` prompt writes with terminal calls**

Examples:

```go
terminal.PrintLine("Running all tests...")
terminal.ShowPrompt()
terminal.SuspendPrompt()
```

Use the abstraction for:

- initial prompt display
- manual help/debug/exit informational lines
- timeout and signal messages in watch mode
- prompt redraw after watch-triggered execution completes

- [ ] **Step 4: Keep semantics out of the terminal**

The watch loop should decide *what* message to print. The terminal should only decide *how* to place it relative to the prompt.

- [ ] **Step 5: Run the targeted watch-loop tests to verify GREEN**

Run: `go test . -run 'Test.*Watch.*Prompt|Test.*Watch.*Logger|TestLoadWatchConfigurationMergesUserAndDefaultWatches' -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd_watch.go cmd_watch_test.go
git commit -m "feat: use terminal abstraction in watch loop"
```

### Task 6: Full verification for the spike

**Files:**
- Modify: any files needed from earlier tasks

- [ ] **Step 1: Run focused Go tests for watch, logger, and command package**

Run: `go test ./watch ./logger .`
Expected: PASS

- [ ] **Step 2: Run the watch-related Ruby integration specs if Ruby is available**

Run: `bin/rspec spec/integration/plur_watch/watch_spec.rb spec/integration/plur_watch/watch_integration_spec.rb`
Expected: PASS

- [ ] **Step 3: If Ruby is unavailable, record that clearly and verify the Go test suite only**

Run: `go test ./watch ./logger .`
Expected: PASS

- [ ] **Step 4: Commit the spike branch state**

```bash
git add -A
git commit -m "feat: spike watch terminal abstraction"
```

## Self-Review

- Spec coverage: the plan covers the small internal abstraction, generic API shape, logger hookup, command echo, and verification.
- Placeholder scan: no placeholders remain.
- Type consistency: the plan uses `Terminal` consistently and keeps semantics in the caller rather than the terminal type.
