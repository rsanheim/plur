package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/logger"
)

// pluralize returns the singular or plural form of a word based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func logDiscoverySummary(cfg *config.GlobalConfig, msg string, attrs ...any) {
	if cfg.DryRun {
		msg = "[dry-run] " + msg
	}
	logger.Logger.Debug(msg, attrs...)
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
	cmdStr := strings.Join(cmd.Args, " ")
	if len(extras) > 0 {
		return strings.Join(extras, " ") + " " + cmdStr
	}
	return cmdStr
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
