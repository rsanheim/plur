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

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/rspec"
	"github.com/rsanheim/rux/tracing"
	"github.com/rsanheim/rux/types"
)

// TestState represents the state of a test execution
type TestState string

const (
	StateSuccess TestState = "success" // All tests passed
	StateFailed  TestState = "failed"  // Some tests failed/exceptions
	StateError   TestState = "error"   // Fatal error, couldn't run tests
)

// WorkerResult represents the accumulated results from a worker executing one or more test files
type WorkerResult struct {
	File         *TestFile // Primary file (first file when multiple files are run together)
	State        TestState
	Output       string
	Error        error
	Duration     time.Duration
	FileLoadTime time.Duration // Time to load spec files before running tests
	JSONOutput   *rspec.JSONOutput
	ExampleCount int
	FailureCount int
	PendingCount int
	Tests        []types.TestCaseNotification // All test notifications
	Framework    TestFramework                // The test framework used

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// Success returns true if the test execution was successful (no failures or errors)
func (r WorkerResult) Success() bool {
	return r.State == StateSuccess
}

// OutputMessage represents a message to be output
type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr"
	Content  string // For error messages
	Files    string // For stderr messages - comma-separated list of files
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
// Following parallel_tests convention: worker 0 gets "", worker 1 gets "2", etc.
func GetTestEnvNumber(workerIndex int) string {
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
			fmt.Fprintf(os.Stderr, "[%s] %s\n", msg.Files, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		}
	}
}

// errorResult creates a WorkerResult for error cases
func errorResult(testFile *TestFile, err error, start time.Time, framework TestFramework) WorkerResult {
	// Extract error message for output
	errorOutput := ""
	if err != nil {
		errorOutput = fmt.Sprintf("Error: %v\n", err)
	}

	return WorkerResult{
		File:      testFile,
		State:     StateError,
		Output:    errorOutput,
		Error:     err,
		Duration:  time.Since(start),
		Framework: framework,
	}
}

// RunSpecFile executes multiple test files in a single test process
func RunSpecFile(ctx context.Context, config *Config, testFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) WorkerResult {
	// Dispatch to framework-specific implementation
	switch config.Framework {
	case FrameworkMinitest:
		return RunMinitestFiles(ctx, config, testFiles, workerIndex, dryRun, outputChan)
	default:
		return RunRSpecFiles(ctx, config, testFiles, workerIndex, dryRun, outputChan)
	}
}

// RunRSpecFiles executes multiple spec files in a single RSpec process
func RunRSpecFiles(ctx context.Context, config *Config, specFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) WorkerResult {
	defer tracing.StartRegionWithWorker(ctx, "run_spec_files", workerIndex, strings.Join(specFiles, ","))()
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

	// Build command using the appropriate builder
	builder := NewCommandBuilder(config.Framework)
	args := builder.BuildCommand(specFiles, config)

	// Log the command in debug mode
	logger.Logger.Debug("executing command", "worker", workerIndex, "command", strings.Join(args, " "))

	if dryRun {
		return WorkerResult{
			File:     testFile,
			State:    StateSuccess,
			Output:   fmt.Sprintf("[dry-run] %s", strings.Join(args, " ")),
			Error:    nil,
			Duration: time.Since(start),
		}
	}

	// Create command with context for timeout handling
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Set up environment variables for parallel testing
	testEnvNumber := GetTestEnvNumber(workerIndex)
	cmd.Env = append(os.Environ(),
		"TEST_ENV_NUMBER="+testEnvNumber,
		"PARALLEL_TEST_GROUPS="+os.Getenv("PARALLEL_TEST_GROUPS"),
		"RUX_FORMATTER_SEPARATOR=RUX_JSON:",
	)

	// Set up stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start, config.Framework)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start, config.Framework)
	}

	// Start the command
	func() {
		defer tracing.StartRegionWithWorker(ctx, "process_spawn", workerIndex, fmt.Sprintf("%d files", len(specFiles)))()
		err = cmd.Start()
	}()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start, config.Framework)
	}

	// Create parser and collector for event-based processing
	parser, err := NewTestOutputParser(config.Framework)
	if err != nil {
		return errorResult(testFile, err, start, config.Framework)
	}
	collector := NewTestCollector()

	// Stream output through parser and collector
	stderrOutput := streamTestOutput(ctx, stdout, stderr, parser, collector, outputChan, workerIndex, specFiles, config.Framework, start)

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
	state := StateSuccess
	output := result.Output + stderrOutput

	// Check if this is an execution error (couldn't run tests)
	if err != nil && result.ExampleCount == 0 &&
		(strings.Contains(output, "error occurred outside of examples") ||
			strings.Contains(result.FormattedSummary, "error occurred outside of examples")) {
		state = StateError
		// For execution errors, keep the full output which contains error details
	} else if !success {
		state = StateFailed
	}

	return WorkerResult{
		File:              testFile,
		State:             state,
		Output:            output,
		Error:             err,
		Duration:          time.Since(start),
		FileLoadTime:      result.FileLoadTime,
		JSONOutput:        jsonOutput,
		ExampleCount:      result.ExampleCount,
		FailureCount:      result.FailureCount,
		PendingCount:      result.PendingCount,
		Tests:             result.Tests,
		FormattedFailures: result.FormattedFailures,
		FormattedSummary:  result.FormattedSummary,
		Framework:         config.Framework,
	}
}

// RunSpecsInParallel executes spec files in parallel using intelligent grouping
func RunSpecsInParallel(config *Config, specFiles []string, runtimeTracker *RuntimeTracker) ([]WorkerResult, time.Duration) {
	defer tracing.StartRegion(context.Background(), "run_specs_parallel_grouped")()
	start := time.Now()
	ctx := context.Background()

	// Load runtime data if available
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	maxWorkers := config.WorkerCount
	colorOutput := config.ColorOutput
	dryRun := config.DryRun

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
			result := RunSpecFile(ctx, config, files, workerIndex, dryRun, outputChan)
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
		if runtimeTracker != nil && result.State != StateError && len(result.Tests) > 0 {
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
func RunMinitestFiles(ctx context.Context, config *Config, testFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) WorkerResult {
	defer tracing.StartRegionWithWorker(ctx, "run_minitest_files", workerIndex, strings.Join(testFiles, ","))()
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

	// Build command using the appropriate builder
	builder := NewCommandBuilder(config.Framework)
	args := builder.BuildCommand(testFiles, config)

	// Log the command in debug mode
	logger.Logger.Debug("executing minitest command", "worker", workerIndex, "command", strings.Join(args, " "))

	if dryRun {
		return WorkerResult{
			File:     testFile,
			State:    StateSuccess,
			Output:   fmt.Sprintf("[dry-run] %s", strings.Join(args, " ")),
			Error:    nil,
			Duration: time.Since(start),
		}
	}

	// Create command with context for timeout handling
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Set up environment variables for parallel testing
	testEnvNumber := GetTestEnvNumber(workerIndex)
	cmd.Env = append(os.Environ(),
		"TEST_ENV_NUMBER="+testEnvNumber,
		"PARALLEL_TEST_GROUPS="+os.Getenv("PARALLEL_TEST_GROUPS"),
	)

	// Set up stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start, config.Framework)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start, config.Framework)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start, config.Framework)
	}

	// Create parser and collector for event-based processing
	parser, err := NewTestOutputParser(config.Framework)
	if err != nil {
		return errorResult(testFile, err, start, config.Framework)
	}
	collector := NewTestCollector()

	// Stream output through parser and collector
	stderrOutput := streamTestOutput(ctx, stdout, stderr, parser, collector, outputChan, workerIndex, testFiles, config.Framework, start)

	// Wait for command to complete
	err = cmd.Wait()

	// Determine success based on exit code
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0

	// Build the final result from the collector
	result := collector.BuildResult(testFile, time.Since(start))

	// Determine the state based on the execution outcome
	state := StateSuccess
	output := result.Output + stderrOutput

	if err != nil && result.ExampleCount == 0 {
		// Couldn't run tests at all
		state = StateError
	} else if !success {
		state = StateFailed
	}

	// Update result with final state and output
	result.State = state
	result.Output = output
	result.Error = err
	result.Framework = config.Framework

	return result
}
