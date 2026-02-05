package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

const (
	EnvTestEnvNumber      = "TEST_ENV_NUMBER"
	EnvParallelTestGroups = "PARALLEL_TEST_GROUPS"
)

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

// Handles grouping files into worker assignments and building the commands to run.
// - Phase 1 (PLAN): Single-threaded - load runtime data, group files, build commands
// - Phase 2 (EXECUTE): The dry-run seam - either print commands or spawn workers
type Runner struct {
	config    *config.GlobalConfig
	files     []string
	job       job.Job
	framework framework.Spec
	tracker   *RuntimeTracker
	extraArgs []string
}

func NewRunner(cfg *config.GlobalConfig, files []string, j job.Job, extraArgs []string) (*Runner, error) {
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, err
	}
	tracker, err := NewRuntimeTracker(cfg.RuntimeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime tracker: %w", err)
	}
	return &Runner{
		config:    cfg,
		files:     files,
		job:       j,
		framework: spec,
		tracker:   tracker,
		extraArgs: extraArgs,
	}, nil
}

// Group files, build the commands, then either print them for dry-run or execute them
func (r *Runner) Run() ([]WorkerResult, time.Duration, error) {
	// planning...
	groups := r.groupFiles()
	commands, err := r.buildCommands(groups)
	if err != nil {
		return nil, 0, err
	}

	r.printSummary(len(commands))

	// executing...
	if r.config.DryRun {
		for i, cmd := range commands {
			printDryRunWorker(r.config.DryRun, i, cmd)
		}
		return nil, 0, nil
	}

	results, wallTime := r.executeWorkers(commands)
	return results, wallTime, nil
}

func (r *Runner) groupFiles() []FileGroup {
	runtimeData := r.tracker.LoadedData()

	var groups []FileGroup
	if len(runtimeData) > 0 {
		groups = GroupSpecFilesByRuntime(r.files, r.config.WorkerCount, runtimeData)
		logger.Logger.Debug("Using runtime-based grouped execution", "group_count", len(groups))
	} else {
		groups = GroupSpecFilesBySize(r.files, r.config.WorkerCount)
		logger.Logger.Debug("Using size-based grouping (no runtime data available)")
	}
	return groups
}

func (r *Runner) buildCommands(groups []FileGroup) ([]*exec.Cmd, error) {
	commands := make([]*exec.Cmd, len(groups))

	for i, group := range groups {
		if r.job.UsesTargets() && logger.IsDebugEnabled() {
			logger.Logger.Debug("ignoring {{target}} tokens in run mode", "job", r.job.Name)
		}

		args, err := framework.BuildRunArgs(r.job, group.Files, r.config, r.extraArgs)
		if err != nil {
			return nil, err
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = r.buildEnv(i, len(groups))
		commands[i] = cmd
	}

	return commands, nil
}

func (r *Runner) buildEnv(workerIndex, totalGroups int) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", EnvParallelTestGroups, totalGroups))

	if !r.config.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, r.config)
		env = append(env, EnvTestEnvNumber+"="+testEnvNumber)
	}

	env = append(env, r.job.Env...)

	return env
}

func (r *Runner) printSummary(workerCount int) {
	actualWorkers := workerCount
	if len(r.files) < workerCount {
		actualWorkers = len(r.files)
	}

	label := r.testLabel()
	toStdErr(r.config.DryRun, "Running %d %s [%s] in parallel using %d workers\n",
		len(r.files), label, r.framework.Name, actualWorkers)
}

func (r *Runner) testLabel() string {
	if r.framework.Name == "rspec" {
		return pluralize(len(r.files), "spec", "specs")
	}
	return pluralize(len(r.files), "test", "tests")
}

func (r *Runner) executeWorkers(commands []*exec.Cmd) ([]WorkerResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()

	results := make(chan WorkerResult, len(commands))
	outputChan := make(chan OutputMessage, len(commands)*10)

	// Set PARALLEL_TEST_GROUPS env var (also set per-command, but this ensures
	// it's available globally for any child process inspection)
	os.Setenv(EnvParallelTestGroups, fmt.Sprintf("%d", len(commands)))

	var outputWg sync.WaitGroup
	outputWg.Go(func() {
		outputAggregator(outputChan, r.config.ColorOutput, r.config.RspecTrace)
	})

	var wg sync.WaitGroup
	for i, cmd := range commands {
		workerIdx := i
		workerCmd := cmd
		wg.Go(func() {
			result := r.runCommand(ctx, workerIdx, workerCmd, outputChan)
			results <- result
		})
	}

	wg.Wait()
	close(results)

	close(outputChan)
	outputWg.Wait()

	var allResults []WorkerResult
	for result := range results {
		allResults = append(allResults, result)
		if result.State != types.StateError && len(result.Tests) > 0 {
			for _, test := range result.Tests {
				r.tracker.AddTestNotification(test)
			}
		}
	}

	fmt.Println() // newline after dots

	return allResults, time.Since(start)
}

func (r *Runner) runCommand(ctx context.Context, workerIdx int, cmd *exec.Cmd, outputChan chan<- OutputMessage) WorkerResult {
	start := time.Now()

	logger.Logger.Info("running", "cmd", dryRunString(cmd), "worker", workerIdx)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(fmt.Errorf("failed to create stdout pipe: %v", err), start)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(fmt.Errorf("failed to create stderr pipe: %v", err), start)
	}
	if err := cmd.Start(); err != nil {
		return errorResult(fmt.Errorf("failed to start command: %v", err), start)
	}

	parser := r.framework.Parser()
	collector := NewTestCollector()
	// Only stream unconsumed stdout for RSpec - Minitest returns consumed=false for everything
	streamStdout := !framework.IsMinitest(r.framework.Name)
	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, outputChan, workerIdx, streamStdout)
	err = cmd.Wait()
	result := collector.BuildResult(time.Since(start))

	logger.Logger.Debug("finished", "worker", workerIdx, "success", err == nil)

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0
	state := types.StateSuccess
	output := result.Output + stderrOutput

	if err != nil && result.ExampleCount == 0 {
		state = types.StateError
	} else if !success {
		state = types.StateFailed
	}

	return WorkerResult{
		State:             state,
		Output:            output,
		Error:             err,
		Duration:          result.Duration,
		FileLoadTime:      result.FileLoadTime,
		ExampleCount:      result.ExampleCount,
		AssertionCount:    result.AssertionCount,
		FailureCount:      result.FailureCount,
		ErrorCount:        result.ErrorCount,
		PendingCount:      result.PendingCount,
		Tests:             result.Tests,
		FormattedFailures: result.FormattedFailures,
		FormattedPending:  result.FormattedPending,
		FormattedSummary:  result.FormattedSummary,
	}
}

func (r *Runner) Tracker() *RuntimeTracker {
	return r.tracker
}

// GetWorkerCount determines number of workers to use; precedence is CLI > env > defaults
func GetWorkerCount(cliWorkers int) int {
	if cliWorkers > 0 {
		return cliWorkers
	}

	if envVar := os.Getenv("PARALLEL_TEST_PROCESSORS"); envVar != "" {
		if count, err := strconv.Atoi(envVar); err == nil && count > 0 {
			return count
		}
	}

	// Default: cores minus 2, minimum 1
	workers := runtime.NumCPU() - 2
	if workers < 1 {
		workers = 1
	}
	return workers
}

// GetTestEnvNumber returns the TEST_ENV_NUMBER for a given worker index
func GetTestEnvNumber(workerIndex int, config *config.GlobalConfig) string {
	if config.FirstIs1 {
		return fmt.Sprintf("%d", workerIndex+1)
	}

	// if first-is-1 is false, the first worker gets "", others get 2,3,4...
	if workerIndex == 0 {
		return ""
	}
	return fmt.Sprintf("%d", workerIndex+1)
}

// outputAggregator handles all output from workers to avoid lock contention
func outputAggregator(outputChan <-chan OutputMessage, colorOutput bool, traceOutput bool) {
	for msg := range outputChan {
		switch msg.Type {
		case "dot":
			if colorOutput {
				os.Stdout.Write(greenDot)
			} else {
				os.Stdout.Write(plainDot)
			}
		case "failure":
			if colorOutput {
				os.Stdout.Write(redF)
			} else {
				os.Stdout.Write(plainF)
			}
		case "pending":
			if colorOutput {
				os.Stdout.Write(yellowStar)
			} else {
				os.Stdout.Write(plainStar)
			}
		case "error_progress":
			if colorOutput {
				os.Stdout.Write(redE)
			} else {
				os.Stdout.Write(plainE)
			}
		case "stderr":
			fmt.Fprintln(os.Stderr, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		case "stdout":
			// Raw stdout from tests (puts/pp output)
			if traceOutput && msg.CurrentFile != "" {
				fmt.Fprintf(os.Stdout, "\n[%s]: %s", msg.CurrentFile, msg.Content)
			} else {
				fmt.Fprintln(os.Stdout, msg.Content)
			}
		}
	}
}

func errorResult(err error, start time.Time) WorkerResult {
	errorOutput := ""
	if err != nil {
		errorOutput = fmt.Sprintf("Error: %v\n", err)
	}

	return WorkerResult{
		State:    types.StateError,
		Output:   errorOutput,
		Error:    err,
		Duration: time.Since(start),
	}
}
