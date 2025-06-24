package main

import (
	"bufio"
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
	"github.com/rsanheim/rux/minitest"
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

// TestResult represents the result of running a single spec file
type TestResult struct {
	File         *TestFile
	State        TestState
	Output       string
	Error        error
	Duration     time.Duration
	FileLoadTime time.Duration // Time to load spec files before running tests
	JSONOutput   *rspec.JSONOutput
	Failures     []TestFailure
	ExampleCount int
	FailureCount int
	PendingCount int
	Tests        []types.TestCaseNotification // All test notifications

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// Success returns true if the test execution was successful (no failures or errors)
func (r TestResult) Success() bool {
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

// errorResult creates a TestResult for error cases
func errorResult(testFile *TestFile, err error, start time.Time) TestResult {
	// Extract error message for output
	errorOutput := ""
	if err != nil {
		errorOutput = fmt.Sprintf("Error: %v\n", err)
	}

	return TestResult{
		File:     testFile,
		State:    StateError,
		Output:   errorOutput,
		Error:    err,
		Duration: time.Since(start),
	}
}

// convertRSpecFailures converts RSpec-specific failures to generic TestFailure
func convertRSpecFailures(testFile *TestFile, rspecFailures []rspec.FailureDetail) []TestFailure {
	failures := make([]TestFailure, len(rspecFailures))
	for i, f := range rspecFailures {
		failures[i] = TestFailure{
			File:        testFile,
			Description: f.Description,
			LineNumber:  f.LineNumber,
			Message:     f.Message,
			Backtrace:   f.Backtrace,
		}
	}
	return failures
}

// RunSpecFile executes multiple test files in a single test process
func RunSpecFile(ctx context.Context, config *Config, testFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) TestResult {
	// Dispatch to framework-specific implementation
	switch config.Framework {
	case FrameworkMinitest:
		return RunMinitestFiles(ctx, config, testFiles, workerIndex, dryRun, outputChan)
	default:
		return RunRSpecFiles(ctx, config, testFiles, workerIndex, dryRun, outputChan)
	}
}

// RunRSpecFiles executes multiple spec files in a single RSpec process
func RunRSpecFiles(ctx context.Context, config *Config, specFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) TestResult {
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
		return TestResult{
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
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start)
	}

	// Start the command
	func() {
		defer tracing.StartRegionWithWorker(ctx, "process_spawn", workerIndex, fmt.Sprintf("%d files", len(specFiles)))()
		err = cmd.Start()
	}()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start)
	}

	// Create parser and collector for event-based processing
	parser := NewRSpecOutputParser()
	collector := NewTestCollector()
	var stderrBuilder strings.Builder
	var wg sync.WaitGroup

	// Stream stdout and parse using event-based architecture
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)

		firstOutput := true
		for scanner.Scan() {
			line := scanner.Text()

			if firstOutput {
				// Trace time to first output
				firstOutput = false
				tracing.LogEvent(ctx, "ruby_first_output",
					"worker_id", workerIndex,
					"spec_files", len(specFiles),
					"time_since_spawn", time.Since(start).Seconds()*1000)
			}

			// Parse line into notifications
			notifications, _ := parser.ParseLine(line)

			// Process each notification
			for _, notification := range notifications {
				collector.AddNotification(notification)

				// Send progress updates to output channel
				switch notification.GetEvent() {
				case types.TestPassed:
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "dot",
					}
				case types.TestFailed:
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "failure",
					}
				case types.TestPending:
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "pending",
					}
				case types.SuiteStarted:
					if suite, ok := notification.(types.SuiteNotification); ok && suite.LoadTime > 0 {
						tracing.LogEvent(ctx, "rspec_loaded",
							"worker_id", workerIndex,
							"spec_files", len(specFiles),
							"load_time", suite.LoadTime.Seconds(),
							"time_since_spawn", time.Since(start).Seconds()*1000)
					}
				}
			}

			// Debug output if enabled
			if os.Getenv("RUX_DEBUG") == "1" && len(notifications) > 0 {
				dump(notifications)
			}
		}
	}()

	// Stream stderr in real-time
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuilder.WriteString("STDERR: " + line + "\n")
			outputChan <- OutputMessage{
				WorkerID: workerIndex,
				Type:     "stderr",
				Content:  line,
				Files:    strings.Join(specFiles, ","),
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for output streaming to complete
	wg.Wait()

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

	// Extract failures from the accumulated test results
	var failures []TestFailure
	for _, test := range result.Tests {
		if test.Event == types.TestFailed && test.Exception != nil {
			failures = append(failures, TestFailure{
				File:        testFile,
				Description: test.FullDescription,
				LineNumber:  test.LineNumber,
				Message:     test.Exception.Message,
				Backtrace:   test.Exception.Backtrace,
			})
		}
	}

	// Determine the state based on the execution outcome
	state := StateSuccess
	output := result.Output + stderrBuilder.String()

	// Check if this is an execution error (couldn't run tests)
	if err != nil && result.ExampleCount == 0 &&
		(strings.Contains(output, "error occurred outside of examples") ||
			strings.Contains(result.FormattedSummary, "error occurred outside of examples")) {
		state = StateError
		// For execution errors, keep the full output which contains error details
	} else if !success {
		state = StateFailed
	}

	return TestResult{
		File:              testFile,
		State:             state,
		Output:            output,
		Error:             err,
		Duration:          time.Since(start),
		FileLoadTime:      result.FileLoadTime,
		JSONOutput:        jsonOutput,
		Failures:          failures,
		ExampleCount:      result.ExampleCount,
		FailureCount:      result.FailureCount,
		PendingCount:      result.PendingCount,
		FormattedFailures: result.FormattedFailures,
		FormattedSummary:  result.FormattedSummary,
	}
}

// RunSpecsInParallel executes spec files in parallel using intelligent grouping
func RunSpecsInParallel(config *Config, specFiles []string, runtimeTracker *RuntimeTracker) ([]TestResult, time.Duration) {
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

	results := make(chan TestResult, len(groups))

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
	var allResults []TestResult
	for result := range results {
		allResults = append(allResults, result)
		// Track runtime data if tracker is available and tests actually ran
		if runtimeTracker != nil && result.State != StateError && result.JSONOutput != nil {
			for _, example := range result.JSONOutput.Examples {
				runtimeTracker.AddExample(example)
			}
		}
	}

	// Ensure newline after dots
	fmt.Println()

	return allResults, time.Since(start)
}

// RunMinitestFiles executes multiple test files in a single Minitest process
func RunMinitestFiles(ctx context.Context, config *Config, testFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) TestResult {
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
		return TestResult{
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
		return errorResult(testFile, fmt.Errorf("failed to create stdout pipe: %v", err), start)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to create stderr pipe: %v", err), start)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		return errorResult(testFile, fmt.Errorf("failed to start command: %v", err), start)
	}

	var outputBuilder strings.Builder
	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")

			// Send progress indicators to output channel
			// Minitest outputs progress indicators on lines containing only dots, F, E, S characters
			if outputChan != nil && isProgressLine(line) {
				for _, char := range line {
					switch char {
					case '.':
						outputChan <- OutputMessage{
							WorkerID: workerIndex,
							Type:     "dot",
						}
					case 'F', 'E':
						outputChan <- OutputMessage{
							WorkerID: workerIndex,
							Type:     "failure",
						}
					case 'S':
						outputChan <- OutputMessage{
							WorkerID: workerIndex,
							Type:     "pending",
						}
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Logger.Error("error reading stdout", "error", err, "worker", workerIndex)
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")
		}
		if err := scanner.Err(); err != nil {
			logger.Logger.Error("error reading stderr", "error", err, "worker", workerIndex)
		}
	}()

	// Wait for all output to be captured
	wg.Wait()

	// Wait for the command to complete
	err = cmd.Wait()

	// Determine success based on exit code
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0

	// Parse minitest output
	outputStr := outputBuilder.String()
	summary, failures := parseMinitestOutput(outputStr)

	// Determine the state based on the execution outcome
	state := StateSuccess
	if err != nil && summary.Tests == 0 {
		// Couldn't run tests at all
		state = StateError
	} else if !success {
		state = StateFailed
	}

	// Convert minitest failures to generic failures
	genericFailures := make([]TestFailure, len(failures))
	for i, f := range failures {
		genericFailures[i] = TestFailure{
			File:        testFile,
			Description: f.Description,
			LineNumber:  f.LineNumber,
			Message:     f.Message,
			Backtrace:   f.Backtrace,
		}
	}

	return TestResult{
		File:         testFile,
		State:        state,
		Output:       outputStr,
		Error:        err,
		Duration:     time.Since(start),
		Failures:     genericFailures,
		ExampleCount: summary.Tests,
		FailureCount: summary.Failures + summary.Errors,
		// For minitest, we'll store the summary in FormattedSummary
		FormattedSummary: fmt.Sprintf("%d runs, %d assertions, %d failures, %d errors, %d skips",
			summary.Tests, summary.Assertions, summary.Failures, summary.Errors, summary.Skips),
	}
}

// parseMinitestOutput parses minitest output and extracts summary and failures
func parseMinitestOutput(output string) (*minitest.OutputSummary, []minitest.FailureDetail) {
	// Parse the output to get summary
	summary, err := minitest.ParseOutput(output)
	if err != nil || summary == nil {
		// Return empty summary if parsing fails
		summary = &minitest.OutputSummary{}
	}

	// Extract failures from output
	failures := minitest.ExtractFailures(output)

	return summary, failures
}

// isProgressLine checks if a line contains only minitest progress indicators
func isProgressLine(line string) bool {
	// Remove any whitespace
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Check if all characters are progress indicators
	for _, char := range trimmed {
		if char != '.' && char != 'F' && char != 'E' && char != 'S' {
			return false
		}
	}

	return true
}
