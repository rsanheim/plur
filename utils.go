package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rsanheim/plur/config"
)

// pluralize returns the singular or plural form of a word based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// workerCountPhrase formats the worker count for run-summary lines.
// IsSerial() (single knob: -n 1) decides serial vs parallel wording;
// count is the actual worker count for the run, which may be clamped
// below WorkerCount when fewer files than workers exist.
func workerCountPhrase(cfg *config.GlobalConfig, count int) string {
	if cfg.IsSerial() {
		return "serially"
	}
	return fmt.Sprintf("in parallel using %d %s", count, pluralize(count, "worker", "workers"))
}

func toStdErr(dryRun bool, format string, args ...any) {
	if dryRun {
		format = "[dry-run] " + format
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// dryRunString returns a shell-executable representation of the command,
// including only the env vars that plur sets (not the full inherited env).
func dryRunString(cmd *exec.Cmd) string {
	cmdStr := strings.Join(shellQuoteArgs(cmd.Args), " ")
	extras := dryRunEnv(cmd)
	if len(extras) > 0 {
		return strings.Join(extras, " ") + " " + cmdStr
	}
	return cmdStr
}

func shellQuoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = shellQuoteArg(arg)
	}
	return quoted
}

func shellQuoteArg(arg string) string {
	if isShellSafeWord(arg) {
		return arg
	}
	return shellSingleQuote(arg)
}

func isShellSafeWord(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			strings.ContainsRune("@%_+=:,./-", r) {
			continue
		}
		return false
	}
	return true
}

func dryRunEnv(cmd *exec.Cmd) []string {
	var envs []string
	if cmd.Env != nil {
		envs = cmd.Environ()
	}
	var extras []string
	for _, env := range envs {
		if strings.HasPrefix(env, EnvTestEnvNumber+"=") ||
			strings.HasPrefix(env, EnvParallelTestGroups+"=") ||
			strings.HasPrefix(env, "RAILS_ENV=") {
			extras = append(extras, env)
		}
	}
	return extras
}

func printDryRunCommand(dryRun bool, cmd *exec.Cmd) {
	if !dryRun {
		return
	}
	toStdErr(true, "%s\n", dryRunString(cmd))
}

func printDryRunWorker(dryRun bool, workerIndex int, cmd *exec.Cmd) {
	if !dryRun {
		return
	}
	toStdErr(true, "Worker %d: %s\n", workerIndex, dryRunString(cmd))
}
