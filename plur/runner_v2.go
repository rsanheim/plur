package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

// RunnerV2 implements the clean execution architecture where:
// - Phase 1 (PLAN): Single-threaded - load runtime data, group files, build commands
// - Phase 2 (EXECUTE): The dry-run seam - either print commands or spawn workers
type RunnerV2 struct {
	config  *config.GlobalConfig
	files   []string
	job     job.Job
	tracker *RuntimeTracker
}

func NewRunnerV2(cfg *config.GlobalConfig, files []string, j job.Job) *RunnerV2 {
	return &RunnerV2{
		config:  cfg,
		files:   files,
		job:     j,
		tracker: NewRuntimeTracker(),
	}
}

func (r *RunnerV2) Run() ([]WorkerResult, time.Duration, error) {
	// === PHASE 1: PLAN (single-threaded) ===
	groups := r.groupFiles()
	commands := r.buildCommands(groups)

	r.printSummary(len(commands))

	// === PHASE 2: EXECUTE (the dry-run seam) ===
	if r.config.DryRun {
		for i, cmd := range commands {
			toStdErr(true, "Worker %d: %s\n", i, strings.Join(cmd.Args, " "))
		}
		return nil, 0, nil
	}

	results, wallTime := r.executeWorkers(commands)
	return results, wallTime, nil
}

func (r *RunnerV2) groupFiles() []FileGroup {
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

func (r *RunnerV2) buildCommands(groups []FileGroup) []*exec.Cmd {
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

func (r *RunnerV2) buildEnv(workerIndex, totalGroups int) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("PARALLEL_TEST_GROUPS=%d", totalGroups))

	if !r.config.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, r.config)
		env = append(env, "TEST_ENV_NUMBER="+testEnvNumber)
	}

	return env
}

func (r *RunnerV2) printSummary(workerCount int) {
	actualWorkers := workerCount
	if len(r.files) < workerCount {
		actualWorkers = len(r.files)
	}

	label := r.testLabel()
	toStdErr(r.config.DryRun, "Running %d %s in parallel using %d workers\n",
		len(r.files), label, actualWorkers)
}

func (r *RunnerV2) testLabel() string {
	switch r.job.Name {
	case "rspec":
		return pluralize(len(r.files), "spec", "specs")
	case "minitest":
		return pluralize(len(r.files), "test", "tests")
	default:
		return pluralize(len(r.files), "test", "tests")
	}
}

func (r *RunnerV2) executeWorkers(commands []*exec.Cmd) ([]WorkerResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()

	results := make(chan WorkerResult, len(commands))
	outputChan := make(chan OutputMessage, len(commands)*10)

	// Set PARALLEL_TEST_GROUPS env var (also set per-command, but this ensures
	// it's available globally for any child process inspection)
	os.Setenv("PARALLEL_TEST_GROUPS", fmt.Sprintf("%d", len(commands)))

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

func (r *RunnerV2) runCommand(ctx context.Context, workerIdx int, cmd *exec.Cmd, outputChan chan<- OutputMessage) WorkerResult {
	start := time.Now()
	testFiles := r.extractTestFiles(cmd.Args)
	var testFile *TestFile
	if len(testFiles) > 0 {
		testFile = &TestFile{
			Path:     testFiles[0],
			Filename: filepath.Base(testFiles[0]),
		}
	} else {
		testFile = &TestFile{
			Path:     "unknown",
			Filename: "unknown",
		}
	}

	commandString := strings.Join(cmd.Args, " ")
	logger.Logger.Info("running", "cmd", commandString, "worker", workerIdx)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start)
	}

	logger.Logger.Debug("starting", "worker", workerIdx, "file_count", len(testFiles), "files", testFiles)
	if err := cmd.Start(); err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start)
	}

	parser, err := r.job.CreateParser()
	if err != nil {
		return errorResult(testFile, err, start)
	}
	collector := NewTestCollector()
	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, outputChan, workerIdx, testFiles)
	err = cmd.Wait()
	result := collector.BuildResult(testFile, time.Since(start))

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
		File:              testFile,
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

func (r *RunnerV2) extractTestFiles(args []string) []string {
	var files []string

	suffix := r.job.GetTargetSuffix()
	if suffix == "" {
		switch r.job.Name {
		case "rspec":
			suffix = "_spec.rb"
		case "minitest":
			suffix = "_test.rb"
		default:
			suffix = ".rb"
		}
	}

	for _, arg := range args {
		if strings.HasSuffix(arg, suffix) {
			files = append(files, arg)
		}
	}

	return files
}

func (r *RunnerV2) Tracker() *RuntimeTracker {
	return r.tracker
}
