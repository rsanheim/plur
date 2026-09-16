// Package term resolves plur's formatter and color against terminal state and
// the NO_COLOR convention (https://no-color.org).
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

// Formatter is how plur renders a run on stdout. Kong maps --formatter, its
// env var, and the config key onto it directly.
type Formatter string

const (
	FormatterAuto     Formatter = "auto"     // progress on a terminal, summary otherwise
	FormatterProgress Formatter = "progress" // one marker per example: . F * E
	FormatterSummary  Formatter = "summary"  // no markers; results only
)

// ResolveFormatter turns auto into a concrete formatter; anything else is returned as is.
func ResolveFormatter(formatter Formatter, stdoutIsTTY bool) Formatter {
	if formatter != FormatterAuto {
		return formatter
	}
	if stdoutIsTTY {
		return FormatterProgress
	}
	return FormatterSummary
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
