// Package term resolves plur's color mode against terminal state and the
// NO_COLOR convention (https://no-color.org).
package term

import (
	"os"

	xterm "golang.org/x/term"
)

// IsStdoutTTY reports whether stdout is attached to a terminal.
func IsStdoutTTY() bool {
	return xterm.IsTerminal(int(os.Stdout.Fd()))
}

// ResolveColor turns a color mode (auto|always|never, plus true/false aliases)
// into an on/off decision and a short source tag. In auto mode, NO_COLOR beats
// TTY detection.
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
