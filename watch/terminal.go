package watch

import (
	"fmt"
	"io"
	"sync"
)

type terminalWriterTarget int

const (
	terminalWriterStdout terminalWriterTarget = iota
	terminalWriterStderr
)

// Terminal owns prompt visibility and synchronized watch-mode output.
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
	return terminalWriter{terminal: t, target: terminalWriterStdout}
}

func (t *Terminal) Stderr() io.Writer {
	return terminalWriter{terminal: t, target: terminalWriterStderr}
}

func (t *Terminal) suspendPromptLocked() {
	if !t.promptVisible {
		return
	}

	fmt.Fprint(t.stdout, "\n")
	t.promptVisible = false
}

type terminalWriter struct {
	terminal *Terminal
	target   terminalWriterTarget
}

func (w terminalWriter) Write(p []byte) (int, error) {
	w.terminal.mu.Lock()
	defer w.terminal.mu.Unlock()

	w.terminal.suspendPromptLocked()

	switch w.target {
	case terminalWriterStdout:
		return w.terminal.stdout.Write(p)
	default:
		return w.terminal.stderr.Write(p)
	}
}
