// Package term resolves plur's color and output modes against terminal state
// and the NO_COLOR convention (https://no-color.org).
package term

import (
	"os"

	xterm "golang.org/x/term"
)

func IsStdoutTTY() bool {
	return xterm.IsTerminal(int(os.Stdout.Fd()))
}

func IsStdinTTY() bool {
	return xterm.IsTerminal(int(os.Stdin.Fd()))
}

// OutputMode is what plur prints on stdout while examples run.
type OutputMode string

const (
	OutputProgress OutputMode = "progress" // one marker per example: . F * E
	OutputSummary  OutputMode = "summary"  // no markers; results only
)

// In auto mode, a terminal gets progress and anything else gets summary.
func ResolveOutput(mode string, stdoutIsTTY bool) OutputMode {
	switch mode {
	case "progress":
		return OutputProgress
	case "summary":
		return OutputSummary
	}
	if stdoutIsTTY {
		return OutputProgress
	}
	return OutputSummary
}

// In auto mode, NO_COLOR beats TTY detection.
func ResolveColor(mode string, stdoutIsTTY bool) (bool, string) {
	switch mode {
	case "always", "true":
		return true, "always"
	case "never", "false":
		return false, "never"
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false, "NO_COLOR"
	}
	if stdoutIsTTY {
		return true, "tty"
	}
	return false, "not a tty"
}
