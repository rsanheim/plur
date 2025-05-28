package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rsanheim/rux/rspec"
)

// Global cached formatter path
var (
	cachedFormatterPath string
	formatterPathOnce   sync.Once
	formatterPathErr    error
)

// TestResult represents the result of running a single spec file
type TestResult struct {
	SpecFile     string
	Success      bool
	Output       string
	Error        error
	Duration     time.Duration
	JSONOutput   *rspec.JSONOutput
	Failures     []rspec.FailureDetail
	ExampleCount int
	FailureCount int
}

// OutputMessage represents a message to be output
type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr"
	Content  string // For error messages
	SpecFile string // For stderr messages
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

// getCachedFormatterPath returns the formatter path, computing it only once
func getCachedFormatterPath() (string, error) {
	formatterPathOnce.Do(func() {
		cacheDir, err := getRuxCacheDir()
		if err != nil {
			formatterPathErr = err
			return
		}
		cachedFormatterPath, formatterPathErr = rspec.GetFormatterPath(cacheDir)
	})
	return cachedFormatterPath, formatterPathErr
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
			fmt.Fprintf(os.Stderr, "[%s] %s\n", msg.SpecFile, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		}
	}
}

// errorResult creates a TestResult for error cases
func errorResult(specFiles []string, err error, start time.Time) TestResult {
	return TestResult{
		SpecFile: strings.Join(specFiles, ","),
		Success:  false,
		Output:   "",
		Error:    err,
		Duration: time.Since(start),
	}
}

// RunSpecFile executes multiple spec files in a single RSpec process
func RunSpecFile(ctx context.Context, specFiles []string, workerIndex int, dryRun bool, outputChan chan<- OutputMessage) TestResult {
	defer TraceFuncWithWorker("run_spec_files", workerIndex, strings.Join(specFiles, ","))()
	start := time.Now()

	// Get the cached formatter path (computed only once)
	var formatterPath string
	var err error
	func() {
		defer TraceFuncWithWorker("get_formatter_path", workerIndex, "grouped")()
		formatterPath, err = getCachedFormatterPath()
	}()
	if err != nil {
		return errorResult(specFiles, fmt.Errorf("failed to get formatter path: %v", err), start)
	}

	// Build args with streaming JSON formatter
	args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}

	// Always use --no-color for RSpec since we'll handle colors ourselves
	args = append(args, "--no-color")
	// Add all spec files
	args = append(args, specFiles...)

	if dryRun {
		return TestResult{
			SpecFile: strings.Join(specFiles, " "),
			Success:  true,
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
		return errorResult(specFiles, fmt.Errorf("failed to create stdout pipe: %v", err), start)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorResult(specFiles, fmt.Errorf("failed to create stderr pipe: %v", err), start)
	}

	// Start the command
	func() {
		defer TraceFuncWithWorker("process_spawn", workerIndex, fmt.Sprintf("%d files", len(specFiles)))()
		err = cmd.Start()
	}()
	if err != nil {
		return errorResult(specFiles, fmt.Errorf("failed to start command: %v", err), start)
	}

	var outputBuilder strings.Builder
	var wg sync.WaitGroup
	streamingResults := &rspec.StreamingResults{}

	// Stream stdout and parse JSON messages in real-time
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
				TraceFuncWithMetadata("ruby_first_output", map[string]interface{}{
					"worker_id":        workerIndex,
					"spec_files":       len(specFiles),
					"time_since_spawn": time.Since(start).Seconds() * 1000,
				})()
			}

			msg, err := rspec.ParseStreamingMessage(line)
			if msg != nil {
				// Handle different message types
				switch msg.Type {
				case "start":
					streamingResults.LoadTime = msg.LoadTime
					TraceFuncWithMetadata("rspec_loaded", map[string]interface{}{
						"worker_id":        workerIndex,
						"spec_files":       len(specFiles),
						"load_time":        msg.LoadTime,
						"time_since_spawn": time.Since(start).Seconds() * 1000,
					})()
				case "example_passed":
					streamingResults.AddExample(*msg)
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "dot",
					}
				case "example_failed":
					streamingResults.AddExample(*msg)
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "failure",
					}
				case "example_pending":
					streamingResults.AddExample(*msg)
					outputChan <- OutputMessage{
						WorkerID: workerIndex,
						Type:     "pending",
					}
				case "close":
					// End of test run
				}
			} else if err != nil {
				// JSON parsing error - log it
				outputBuilder.WriteString(fmt.Sprintf("JSON parse error: %v for line: %s\n", err, line))
			} else {
				// Non-JSON output (warnings, errors, etc.)
				outputBuilder.WriteString(line + "\n")
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
			outputBuilder.WriteString("STDERR: " + line + "\n")
			outputChan <- OutputMessage{
				WorkerID: workerIndex,
				Type:     "stderr",
				Content:  line,
				SpecFile: strings.Join(specFiles, ","),
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for output streaming to complete
	wg.Wait()

	// Determine success based on exit code
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	success := exitCode == 0

	// Convert streaming results to RSpec JSON format
	jsonOutput := streamingResults.ConvertToJSONOutput()
	failures := rspec.ExtractFailures(jsonOutput.Examples)

	return TestResult{
		SpecFile:     strings.Join(specFiles, " "),
		Success:      success,
		Output:       outputBuilder.String(),
		Error:        err,
		Duration:     time.Since(start),
		JSONOutput:   jsonOutput,
		Failures:     failures,
		ExampleCount: streamingResults.ExampleCount,
		FailureCount: streamingResults.FailureCount,
	}
}

// RunSpecsInParallel executes spec files in parallel using intelligent grouping
func RunSpecsInParallel(specFiles []string, dryRun bool, colorOutput bool, maxWorkers int, runtimeTracker *RuntimeTracker) ([]TestResult, time.Duration) {
	defer TraceFunc("run_specs_parallel_grouped")()
	start := time.Now()
	ctx := context.Background()

	// Load runtime data if available
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	// Group files using runtime data if available, otherwise by size
	var groups []FileGroup
	if len(runtimeData) > 0 {
		fmt.Fprintf(os.Stderr, "Using runtime-based grouped execution: %d %s across %d workers\n", len(specFiles), pluralize(len(specFiles), "file", "files"), maxWorkers)
		groups = GroupSpecFilesByRuntime(specFiles, maxWorkers, runtimeData)
	} else {
		fmt.Fprintf(os.Stderr, "Using size-based grouped execution: %d %s across %d workers\n", len(specFiles), pluralize(len(specFiles), "file", "files"), maxWorkers)
		groups = GroupSpecFilesBySize(specFiles, maxWorkers)
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
			result := RunSpecFile(ctx, files, workerIndex, dryRun, outputChan)
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
		// Track runtime data if tracker is available
		if runtimeTracker != nil && result.JSONOutput != nil {
			for _, example := range result.JSONOutput.Examples {
				runtimeTracker.AddExample(example)
			}
		}
	}

	// Ensure newline after dots
	fmt.Println()

	return allResults, time.Since(start)
}
