// Package term resolves plur's color mode against terminal state and the
// NO_COLOR convention (https://no-color.org).
package term

import (
	"os"

	xterm "golang.org/x/term"
)

// LookupEnv is the subset of os.LookupEnv the resolver needs; a func type so
// tests can inject environments without mutating the process.
type LookupEnv func(string) (string, bool)

// IsStdoutTTY reports whether stdout is attached to a terminal.
func IsStdoutTTY() bool {
	return xterm.IsTerminal(int(os.Stdout.Fd()))
}

// ResolveColor turns a color mode (auto, always, never — with true/false
// accepted as aliases for always/never, so boolean config values work) into
// the final on/off decision, plus a short source tag for doctor/verbose
// output. Precedence within auto: NO_COLOR (https://no-color.org), then TTY
// detection.
func ResolveColor(mode string, lookup LookupEnv, stdoutIsTTY bool) (bool, string) {
	switch mode {
	case "always", "true":
		return true, "always"
	case "never", "false":
		return false, "never"
	}
	if _, ok := lookup("NO_COLOR"); ok {
		return false, "NO_COLOR"
	}
	if stdoutIsTTY {
		return true, "tty"
	}
	return false, "not a tty"
}

// EnvDecidesColor reports whether NO_COLOR is present and so would decide the
// outcome in auto mode. Config-file loading uses this to let env outrank the
// config file's color value (env > config, per plur's precedence ladder).
func EnvDecidesColor(lookup LookupEnv) bool {
	_, ok := lookup("NO_COLOR")
	return ok
}
