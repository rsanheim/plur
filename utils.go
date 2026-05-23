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
	if cmd.Env == nil {
		return nil
	}

	envs := cmd.Env
	if inherited := os.Environ(); hasInheritedEnvPrefix(envs, inherited) {
		return withInheritedManagedEnv(dedupeEnvByKey(validEnvEntries(envs[len(inherited):])), envs)
	}

	var extras []string
	for _, env := range cmd.Environ() {
		if isManagedDryRunEnvEntry(env) {
			extras = append(extras, env)
		}
	}
	return dedupeEnvByKey(extras)
}

func hasInheritedEnvPrefix(envs, inherited []string) bool {
	if len(envs) < len(inherited) {
		return false
	}
	for i, env := range inherited {
		if envs[i] != env {
			return false
		}
	}
	return true
}

func validEnvEntries(envs []string) []string {
	entries := make([]string, 0, len(envs))
	for _, env := range envs {
		if strings.Contains(env, "=") {
			entries = append(entries, env)
		}
	}
	return entries
}

func dedupeEnvByKey(envs []string) []string {
	lastIndex := make(map[string]int, len(envs))
	for i, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if ok {
			lastIndex[key] = i
		}
	}

	entries := make([]string, 0, len(lastIndex))
	for i, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if ok && lastIndex[key] == i {
			entries = append(entries, env)
		}
	}
	return entries
}

func withInheritedManagedEnv(extras, envs []string) []string {
	seen := make(map[string]struct{}, len(extras))
	for _, env := range extras {
		if key, _, ok := strings.Cut(env, "="); ok {
			seen[key] = struct{}{}
		}
	}
	for _, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if !ok || key != "RAILS_ENV" {
			continue
		}
		if _, exists := seen[key]; !exists {
			return append([]string{env}, extras...)
		}
	}
	return extras
}

func isManagedDryRunEnvEntry(env string) bool {
	key, _, ok := strings.Cut(env, "=")
	return ok && (key == EnvTestEnvNumber || key == EnvParallelTestGroups || key == "RAILS_ENV")
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
