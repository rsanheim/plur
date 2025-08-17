package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/rspec"
	"github.com/rsanheim/plur/types"
)

// WorkerResult represents the accumulated results from a worker executing one or more test files
type WorkerResult struct {
	File         *TestFile // Primary file (first file when multiple files are run together)
	State        types.TestState
	Output       string
	Error        error
	Duration     time.Duration
	FileLoadTime time.Duration // Time to load spec files before running tests
	JSONOutput   *rspec.JSONOutput
	ExampleCount int
	FailureCount int
	PendingCount int
	Tests        []types.TestCaseNotification // All test notifications
	Framework    config.TestFramework         // The test framework used

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// Success returns true if the test execution was successful (no failures or errors)
func (r WorkerResult) Success() bool {
	return r.State == types.StateSuccess
}

// OutputMessage represents a message to be output
type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr"
	Content  string // For error messages
	Files    string // For stderr messages - comma-separated list of files
}

// getFrameworkFromTask converts task name to TestFramework enum
func getFrameworkFromTask(currentTask *task.Task) config.TestFramework {
	switch currentTask.Name {
	case "minitest":
		return config.FrameworkMinitest
	case "rspec":
		return config.FrameworkRSpec
	default:
		return config.FrameworkRSpec // Default fallback
	}
}

// GetWorkerCount determines the number of workers to use based on CLI, env, and defaults
func GetWorkerCount(cliWorkers int) int {
	// Priority: CLI flag > ENV var > default (cores-2)
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
// Note: This should not be called in serial mode (config.IsSerial() == true)
func GetTestEnvNumber(workerIndex int, config *config.GlobalConfig) string {
	// New default behavior: all workers get explicit numbers
	if config.FirstIs1 {
		return fmt.Sprintf("%d", workerIndex+1)
	}

	// Legacy behavior: first worker gets "", others get 2,3,4...
	if workerIndex == 0 {
		return ""
	}
	return fmt.Sprintf("%d", workerIndex+1)
}

// ANSI color codes
const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// Pre-compiled output strings to avoid repeated concatenation
var (
	greenDot   = []byte(colorGreen + "." + colorReset)
	redF       = []byte(colorRed + "F" + colorReset)
	yellowStar = []byte(colorYellow + "*" + colorReset)
	plainDot   = []byte(".")
	plainF     = []byte("F")
	plainStar  = []byte("*")
)

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
			// Just output stderr content without noisy file list prefix
			fmt.Fprintln(os.Stderr, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		}
	}
}

// errorResult creates a WorkerResult for error cases
func errorResult(testFile *TestFile, err error, start time.Time, framework config.TestFramework) WorkerResult {
	// Extract error message for output
	errorOutput := ""
	if err != nil {
		errorOutput = fmt.Sprintf("Error: %v\n", err)
	}

	return WorkerResult{
		File:      testFile,
		State:     types.StateError,
		Output:    errorOutput,
		Error:     err,
		Duration:  time.Since(start),
		Framework: framework,
	}
}

// RunSpecFile executes multiple test files in a single test process
func RunSpecFile(ctx context.Context, globalConfig *config.GlobalConfig, specCmd *SpecCmd, testFiles []string, workerIndex int, outputChan chan<- OutputMessage, currentTask *task.Task) WorkerResult {
	// Dispatch to framework-specific implementation
	framework := getFrameworkFromTask(currentTask)
	switch framework {
	case config.FrameworkMinitest:
		return RunMinitestFiles(ctx, globalConfig, specCmd, testFiles, workerIndex, outputChan, currentTask)
	default:
		return RunRSpecFiles(ctx, globalConfig, specCmd, testFiles, workerIndex, outputChan, currentTask)
	}
}

// RunRSpecFiles executes multiple spec files in a single RSpec process
func RunRSpecFiles(ctx context.Context, globalConfig *config.GlobalConfig, specCmd *SpecCmd, specFiles []string, workerIndex int, outputChan chan<- OutputMessage, currentTask *task.Task) WorkerResult {
	start := time.Now()

	// Create TestFile for the primary file (or combined representation)
	var testFile *TestFile
	if len(specFiles) > 0 {
		testFile = &TestFile{
			Path:     specFiles[0], // Use first file as primary
			Filename: filepath.Base(specFiles[0]),
		}
	} else {
		testFile = &TestFile{
			Path:     "unknown",
			Filename: "unknown",
		}
	}

	// Build command using the task
	args := currentTask.BuildCommand(specFiles, globalConfig, specCmd.Command)

	// Log the command in debug mode
	logger.Logger.Debug("executing command", "worker", workerIndex, "command", strings.Join(args, " "))

	if globalConfig.DryRun {
		return WorkerResult{
			File:     testFile,
			State:    types.StateSuccess,
			Output:   fmt.Sprintf("[dry-run] %s", strings.Join(args, " ")),
			Error:    nil,
			Duration: time.Since(start),
		}
	}

	// Create command with context for timeout handling
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Set up environment variables for parallel testing
	env := os.Environ()
	env = append(env, "PARALLEL_TEST_GROUPS="+os.Getenv("PARALLEL_TEST_GROUPS"))
	env = append(env, "PLUR_FORMATTER_SEPARATOR=PLUR_JSON:")

	// Only set TEST_ENV_NUMBER if not in serial mode
	if !globalConfig.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, globalConfig)
		env = append(env, "TEST_ENV_NUMBER="+testEnvNumber)
	}

	cmd.Env = env

	// Set up stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start, getFrameworkFromTask(currentTask))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start, getFrameworkFromTask(currentTask))
	}

	// Start the command
	func() {
		err = cmd.Start()
	}()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start, getFrameworkFromTask(currentTask))
	}

	// Create parser and collector for event-based processing
	parser, err := NewTestOutputParser(getFrameworkFromTask(currentTask))
	if err != nil {
		return errorResult(testFile, err, start, getFrameworkFromTask(currentTask))
	}

	// Use fixed default allocation hints
	collector := NewTestCollector()

	// Stream output through parser and collector
	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, outputChan, workerIndex, specFiles, getFrameworkFromTask(currentTask))

	// Wait for command to complete
	err = cmd.Wait()

	// Build the final result from the collector
	result := collector.BuildResult(testFile, time.Since(start))

	// Determine success based on exit code
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0

	// Convert accumulated results to RSpec JSON format for compatibility
	jsonOutput := result.JSONOutput
	if jsonOutput == nil {
		jsonOutput = &rspec.JSONOutput{
			Version:  "3.13.0",
			Examples: []rspec.Example{},
			Summary: rspec.Summary{
				Duration:     result.Duration.Seconds(),
				ExampleCount: result.ExampleCount,
				FailureCount: result.FailureCount,
				PendingCount: result.PendingCount,
			},
		}
	}

	// Determine the state based on the execution outcome
	state := types.StateSuccess
	output := result.Output + stderrOutput

	// Check if this is an execution error (couldn't run tests)
	if err != nil && result.ExampleCount == 0 &&
		(strings.Contains(output, "error occurred outside of examples") ||
			strings.Contains(result.FormattedSummary, "error occurred outside of examples")) {
		state = types.StateError
		// For execution errors, keep the full output which contains error details
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
		JSONOutput:        jsonOutput,
		ExampleCount:      result.ExampleCount,
		FailureCount:      result.FailureCount,
		PendingCount:      result.PendingCount,
		Tests:             result.Tests,
		FormattedFailures: result.FormattedFailures,
		FormattedSummary:  result.FormattedSummary,
		Framework:         getFrameworkFromTask(currentTask),
	}
}

// RunSpecsInParallel executes spec files in parallel using intelligent grouping
func RunSpecsInParallel(globalConfig *config.GlobalConfig, specCmd *SpecCmd, specFiles []string, runtimeTracker *RuntimeTracker, currentTask *task.Task) ([]WorkerResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()

	// Load runtime data if available
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	maxWorkers := globalConfig.WorkerCount
	colorOutput := globalConfig.ColorOutput

	// Group files using runtime data if available, otherwise by size
	var groups []FileGroup
	if len(runtimeData) > 0 {
		fmt.Fprintf(os.Stderr, "Using runtime-based grouped execution: %d %s across %d workers\n", len(specFiles), pluralize(len(specFiles), "file", "files"), maxWorkers)
		groups = GroupSpecFilesByRuntime(specFiles, maxWorkers, runtimeData)
		logger.LogVerbose("Using runtime-based grouping", "runtime_entries", len(runtimeData))
	} else {
		fmt.Fprintf(os.Stderr, "Using size-based grouped execution: %d %s across %d workers\n", len(specFiles), pluralize(len(specFiles), "file", "files"), maxWorkers)
		groups = GroupSpecFilesBySize(specFiles, maxWorkers)
		logger.LogVerbose("Using size-based grouping (no runtime data available)")
	}

	// Log group assignments in verbose mode
	if logger.VerboseMode {
		for i, group := range groups {
			// TotalSize represents milliseconds when using runtime data, bytes when using file size
			runtimeInfo := "by file size"
			if len(runtimeData) > 0 {
				runtimeInfo = fmt.Sprintf("%.2fs", float64(group.TotalSize)/1000.0)
			}
			logger.LogVerbose("Worker assignment",
				"worker", i,
				"files", group.Files,
				"estimated_time", runtimeInfo)
		}
	}

	results := make(chan WorkerResult, len(groups))

	// Create buffered channel for output messages
	outputChan := make(chan OutputMessage, maxWorkers*10)

	// Start output aggregator goroutine
	var outputWg sync.WaitGroup
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()
		outputAggregator(outputChan, colorOutput)
	}()

	// Set PARALLEL_TEST_GROUPS environment variable
	os.Setenv("PARALLEL_TEST_GROUPS", fmt.Sprintf("%d", len(groups)))

	// Run each group in parallel
	var wg sync.WaitGroup
	for i, group := range groups {
		wg.Add(1)
		go func(workerIndex int, files []string) {
			defer wg.Done()
			logger.LogVerbose("Worker starting", "worker", workerIndex, "file_count", len(files))
			result := RunSpecFile(ctx, globalConfig, specCmd, files, workerIndex, outputChan, currentTask)
			logger.LogVerbose("Worker finished", "worker", workerIndex, "status", result.Success())
			results <- result
		}(i, group.Files)
	}

	// Wait for all groups to complete
	wg.Wait()
	close(results)

	// Close output channel and wait for aggregator to finish
	close(outputChan)
	outputWg.Wait()

	// Collect results
	var allResults []WorkerResult
	for result := range results {
		allResults = append(allResults, result)
		// Track runtime data if tracker is available and tests actually ran
		if runtimeTracker != nil && result.State != types.StateError && len(result.Tests) > 0 {
			for _, test := range result.Tests {
				runtimeTracker.AddTestNotification(test)
			}
		}
	}

	// Ensure newline after dots
	fmt.Println()

	return allResults, time.Since(start)
}

// RunMinitestFiles executes multiple test files in a single Minitest process
func RunMinitestFiles(ctx context.Context, globalConfig *config.GlobalConfig, specCmd *SpecCmd, testFiles []string, workerIndex int, outputChan chan<- OutputMessage, currentTask *task.Task) WorkerResult {
	start := time.Now()

	// Create TestFile for the primary file (or combined representation)
	var testFile *TestFile
	if len(testFiles) > 0 {
		testFile = &TestFile{
			Path:     testFiles[0], // Use first file as primary
			Filename: filepath.Base(testFiles[0]),
		}
	} else {
		testFile = &TestFile{
			Path:     "unknown",
			Filename: "unknown",
		}
	}

	// Build command using the task
	args := currentTask.BuildCommand(testFiles, globalConfig, specCmd.Command)

	// Log the command in debug mode
	logger.Logger.Debug("executing minitest command", "worker", workerIndex, "command", strings.Join(args, " "))

	if globalConfig.DryRun {
		return WorkerResult{
			File:     testFile,
			State:    types.StateSuccess,
			Output:   fmt.Sprintf("[dry-run] %s", strings.Join(args, " ")),
			Error:    nil,
			Duration: time.Since(start),
		}
	}

	// Create command with context for timeout handling
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Set up environment variables for parallel testing
	env := os.Environ()
	env = append(env, "PARALLEL_TEST_GROUPS="+os.Getenv("PARALLEL_TEST_GROUPS"))

	// Only set TEST_ENV_NUMBER if not in serial mode
	if !globalConfig.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, globalConfig)
		env = append(env, "TEST_ENV_NUMBER="+testEnvNumber)
	}

	cmd.Env = env

	// Set up stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start, getFrameworkFromTask(currentTask))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start, getFrameworkFromTask(currentTask))
	}

	err = cmd.Start()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start, getFrameworkFromTask(currentTask))
	}

	parser, err := NewTestOutputParser(getFrameworkFromTask(currentTask))
	if err != nil {
		return errorResult(testFile, err, start, getFrameworkFromTask(currentTask))
	}

	collector := NewTestCollector()

	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, outputChan, workerIndex, testFiles, getFrameworkFromTask(currentTask))

	err = cmd.Wait()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0

	result := collector.BuildResult(testFile, time.Since(start))

	state := types.StateSuccess
	output := result.Output + stderrOutput

	if err != nil && result.ExampleCount == 0 {
		// Couldn't run tests at all
		state = types.StateError
	} else if !success {
		state = types.StateFailed
	}

	result.State = state
	result.Output = output
	result.Error = err
	result.Framework = getFrameworkFromTask(currentTask)

	return result
}
