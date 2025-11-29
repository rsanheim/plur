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
	var extras []string
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, EnvTestEnvNumber+"=") ||
			strings.HasPrefix(env, EnvParallelTestGroups+"=") {
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
	config  *config.GlobalConfig
	files   []string
	job     job.Job
	tracker *RuntimeTracker
}

func NewRunner(cfg *config.GlobalConfig, files []string, j job.Job) *Runner {
	return &Runner{
		config:  cfg,
		files:   files,
		job:     j,
		tracker: NewRuntimeTracker(),
	}
}

// Group files, build the commands, then either print them for dry-run or execute them
func (r *Runner) Run() ([]WorkerResult, time.Duration, error) {
	// planning...
	groups := r.groupFiles()
	commands := r.buildCommands(groups)

	r.printSummary(len(commands))

	// executing...
	if r.config.DryRun {
		for i, cmd := range commands {
			toStdErr(true, "Worker %d: %s\n", i, dryRunString(cmd))
		}
		return nil, 0, nil
	}

	results, wallTime := r.executeWorkers(commands)
	return results, wallTime, nil
}

func (r *Runner) groupFiles() []FileGroup {
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

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

func (r *Runner) buildCommands(groups []FileGroup) []*exec.Cmd {
	commands := make([]*exec.Cmd, len(groups))

	for i, group := range groups {
		var args []string
		switch r.job.Name {
		case "rspec":
			args = buildRSpecCommand(r.job, group.Files, r.config)
		case "minitest":
			args = buildMinitestCommand(r.job, group.Files, r.config)
		default:
			args = job.BuildJobCmd(r.job, group.Files)
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = r.buildEnv(i, len(groups))
		commands[i] = cmd
	}

	return commands
}

func (r *Runner) buildEnv(workerIndex, totalGroups int) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", EnvParallelTestGroups, totalGroups))

	if !r.config.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, r.config)
		env = append(env, EnvTestEnvNumber+"="+testEnvNumber)
	}

	return env
}

func (r *Runner) printSummary(workerCount int) {
	actualWorkers := workerCount
	if len(r.files) < workerCount {
		actualWorkers = len(r.files)
	}

	label := r.testLabel()
	toStdErr(r.config.DryRun, "Running %d %s in parallel using %d workers\n",
		len(r.files), label, actualWorkers)
}

func (r *Runner) testLabel() string {
	switch r.job.Name {
	case "rspec":
		return pluralize(len(r.files), "spec", "specs")
	case "minitest":
		return pluralize(len(r.files), "test", "tests")
	default:
		return pluralize(len(r.files), "test", "tests")
	}
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
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()
		outputAggregator(outputChan, r.config.ColorOutput)
	}()

	var wg sync.WaitGroup
	for i, cmd := range commands {
		wg.Add(1)
		go func(workerIdx int, c *exec.Cmd) {
			defer wg.Done()
			result := r.runCommand(ctx, workerIdx, c, outputChan)
			results <- result
		}(i, cmd)
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

	parser, err := r.job.CreateParser()
	if err != nil {
		return errorResult(err, start)
	}
	collector := NewTestCollector()
	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, outputChan, workerIdx)
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
		FailureCount:      result.FailureCount,
		PendingCount:      result.PendingCount,
		Tests:             result.Tests,
		FormattedFailures: result.FormattedFailures,
		FormattedSummary:  result.FormattedSummary,
	}
}

func (r *Runner) Tracker() *RuntimeTracker {
	return r.tracker
}

// GetWorkerCount determines the number of workers to use based on CLI, env, and defaults
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
func outputAggregator(outputChan <-chan OutputMessage, colorOutput bool) {
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
		case "stderr":
			fmt.Fprintln(os.Stderr, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
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

// insertBeforeFiles inserts additional arguments before the file arguments in a command
func insertBeforeFiles(args []string, files []string, newArgs ...string) []string {
	filesStart := -1
	for i, arg := range args {
		for _, file := range files {
			if arg == file {
				filesStart = i
				break
			}
		}
		if filesStart != -1 {
			break
		}
	}

	if filesStart == -1 {
		return append(args, newArgs...)
	}

	// Insert new args before files
	result := make([]string, 0, len(args)+len(newArgs))
	result = append(result, args[:filesStart]...)
	result = append(result, newArgs...)
	result = append(result, args[filesStart:]...)
	return result
}

// buildRSpecCommand builds an RSpec command with framework-specific flags
func buildRSpecCommand(j job.Job, files []string, globalConfig *config.GlobalConfig) []string {
	args := job.BuildJobCmd(j, files)

	// Add formatter if available
	formatterPath := globalConfig.ConfigPaths.GetJSONRowsFormatterPath()
	if formatterPath != "" {
		args = insertBeforeFiles(args, files, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")
	}

	if !globalConfig.ColorOutput {
		args = insertBeforeFiles(args, files, "--no-color")
	} else {
		args = insertBeforeFiles(args, files, "--force-color", "--tty")
	}

	return args
}

// buildMinitestCommand builds a Minitest command with framework-specific handling
func buildMinitestCommand(j job.Job, files []string, _globalConfig *config.GlobalConfig) []string {
	if len(files) > 1 {
		cmd := []string{"bundle", "exec", "ruby", "-Itest"}
		requires := make([]string, 0, len(files))
		for _, file := range files {
			// Strip the "test/" prefix if present since we're using -Itest, and strip the .rb extension
			testFile := strings.TrimPrefix(file, "test/")
			testFile = strings.TrimSuffix(testFile, ".rb")
			requires = append(requires, testFile)
		}

		// Create the require pattern
		requireList := `"` + strings.Join(requires, `", "`) + `"`
		cmd = append(cmd, "-e", `[`+requireList+`].each { |f| require f }`)
		return cmd
	}

	// For single file, use BuildJobCmd directly
	return job.BuildJobCmd(j, files)
}
