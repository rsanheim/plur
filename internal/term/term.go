// Package term resolves plur's color mode against terminal state and the
// color-related environment conventions (NO_COLOR, FORCE_COLOR, CLICOLOR_FORCE).
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

// ResolveColor turns a color mode (auto, always, never — with on/off accepted
// as aliases) into the final on/off decision, plus a short source tag for
// doctor/verbose output. Precedence within auto: FORCE_COLOR/CLICOLOR_FORCE,
// then NO_COLOR (https://no-color.org), then TTY detection.
func ResolveColor(mode string, lookup LookupEnv, stdoutIsTTY bool) (bool, string) {
	switch mode {
	case "always", "on":
		return true, "always"
	case "never", "off":
		return false, "never"
	}
	if name, ok := forceColorVar(lookup); ok {
		return true, name
	}
	if _, ok := lookup("NO_COLOR"); ok {
		return false, "NO_COLOR"
	}
	if stdoutIsTTY {
		return true, "tty"
	}
	return false, "not a tty"
}

// EnvDecidesColor reports whether the color env conventions would decide the
// outcome in auto mode. Config-file loading uses this to let env outrank the
// config file's color value (env > config, per plur's precedence ladder).
func EnvDecidesColor(lookup LookupEnv) bool {
	if _, ok := forceColorVar(lookup); ok {
		return true
	}
	_, ok := lookup("NO_COLOR")
	return ok
}

// forceColorVar returns which force variable (if any) is actively forcing
// color on. Values "", "0", and "false" mean not-forcing, per ecosystem
// convention; NO_COLOR by contrast is presence-based (any value counts).
func forceColorVar(lookup LookupEnv) (string, bool) {
	for _, name := range []string{"FORCE_COLOR", "CLICOLOR_FORCE"} {
		if v, ok := lookup(name); ok && v != "" && v != "0" && v != "false" {
			return name, true
		}
	}
	return "", false
}
