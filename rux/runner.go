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
	JSONOutput   *RSpecJSONOutput
	Failures     []FailureDetail
	ExampleCount int
	FailureCount int
}

// JobWithWorker represents a job assigned to a specific worker
type JobWithWorker struct {
	SpecFile    string
	WorkerIndex int
}

// OutputMessage represents a message to be output
type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr"
	Content  string // For error messages
	SpecFile string // For stderr messages
}

// FindSpecFiles discovers all spec files in the spec directory
func FindSpecFiles() ([]string, error) {
	var specFiles []string

	// Check if spec directory exists
	if _, err := os.Stat("spec"); os.IsNotExist(err) {
		return specFiles, nil // Return empty list if no spec directory
	}

	// Walk the spec directory recursively
	err := filepath.WalkDir("spec", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file ends with _spec.rb
		if strings.HasSuffix(path, "_spec.rb") {
			specFiles = append(specFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking spec directory: %v", err)
	}

	return specFiles, nil
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
		cachedFormatterPath, formatterPathErr = GetFormatterPath()
	})
	return cachedFormatterPath, formatterPathErr
}

// outputAggregator handles all output from workers to avoid lock contention
func outputAggregator(outputChan <-chan OutputMessage, colorOutput bool) {
	for msg := range outputChan {
		switch msg.Type {
		case "dot":
			if colorOutput {
				fmt.Print("\033[32m.\033[0m") // Green dot
			} else {
				fmt.Print(".")
			}
		case "failure":
			if colorOutput {
				fmt.Print("\033[31mF\033[0m") // Red F
			} else {
				fmt.Print("F")
			}
		case "pending":
			if colorOutput {
				fmt.Print("\033[33m*\033[0m") // Yellow asterisk
			} else {
				fmt.Print("*")
			}
		case "stderr":
			fmt.Fprintf(os.Stderr, "[%s] %s\n", msg.SpecFile, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		}
	}
}

// RunSpecFile executes a single spec file using the streaming JSON formatter
func RunSpecFile(ctx context.Context, specFile string, workerIndex int, dryRun bool, saveJSON bool, outputChan chan<- OutputMessage) TestResult {
	start := time.Now()

	// Get the cached formatter path (computed only once)
	formatterPath, err := getCachedFormatterPath()
	if err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to get formatter path: %v", err),
			Duration: time.Since(start),
		}
	}

	// Build args with streaming JSON formatter
	args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}

	// Always use --no-color for RSpec since we'll handle colors ourselves
	args = append(args, "--no-color")
	args = append(args, specFile)

	if dryRun {
		return TestResult{
			SpecFile: specFile,
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
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stdout pipe: %v", err),
			Duration: time.Since(start),
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stderr pipe: %v", err),
			Duration: time.Since(start),
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to start command: %v", err),
			Duration: time.Since(start),
		}
	}

	var outputBuilder strings.Builder
	var wg sync.WaitGroup
	streamingResults := &StreamingResults{}

	// Stream stdout and parse JSON messages in real-time
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			msg, err := ParseJSONMessage(line)
			if msg != nil {
				// Handle different message types
				switch msg.Type {
				case "start":
					streamingResults.LoadTime = msg.LoadTime
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
				SpecFile: specFile,
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for output streaming to complete
	wg.Wait()

	// Determine success based on exit code
	success := err == nil
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		success = exitCode <= 1
	}

	// Convert streaming results to RSpec JSON format
	jsonOutput := streamingResults.ConvertToRSpecJSON()
	failures := ExtractFailures(jsonOutput.Examples)

	success = exitCode == 0

	return TestResult{
		SpecFile:     specFile,
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

// RunSpecsInParallel executes spec files in parallel using a worker pool
func RunSpecsInParallel(specFiles []string, dryRun bool, saveJSON bool, colorOutput bool, maxWorkers int) ([]TestResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()
	results := make(chan TestResult, len(specFiles))

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
	os.Setenv("PARALLEL_TEST_GROUPS", fmt.Sprintf("%d", maxWorkers))

	// Create worker pool with limited workers
	jobs := make(chan JobWithWorker, len(specFiles))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		workerIndex := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := RunSpecFile(ctx, job.SpecFile, workerIndex, dryRun, saveJSON, outputChan)
				results <- result
			}
		}()
	}

	// Send jobs to workers
	go func() {
		for i, specFile := range specFiles {
			jobs <- JobWithWorker{
				SpecFile:    specFile,
				WorkerIndex: i % maxWorkers, // Distribute files across workers
			}
		}
		close(jobs)
	}()

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Close output channel and wait for aggregator to finish
	close(outputChan)
	outputWg.Wait()

	// Collect results
	var testResults []TestResult
	for result := range results {
		testResults = append(testResults, result)
	}

	return testResults, time.Since(start)
}
