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

// RunSpecFile executes a single spec file and returns the result
func RunSpecFile(ctx context.Context, specFile string, workerIndex int, dryRun bool, saveJSON bool, outputMutex *sync.Mutex) TestResult {
	start := time.Now()

	// Always create temp file for JSON output (we need it for failure reporting)
	// Use project's tmp directory
	tmpDir := filepath.Join(filepath.Dir(specFile), "..", "tmp")
	os.MkdirAll(tmpDir, 0755) // Ensure tmp directory exists

	tmpFile, err := os.CreateTemp(tmpDir, "rux-results-*.json")
	if err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create temp file: %v", err),
			Duration: time.Since(start),
		}
	}
	jsonFile := tmpFile.Name()
	tmpFile.Close()

	// Build args with both progress and JSON formatters
	args := []string{"bundle", "exec", "rspec", "--format", "progress", "--format", "json", "--out", jsonFile, specFile}

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
		"PARALLEL_TEST_GROUPS="+os.Getenv("PARALLEL_TEST_GROUPS"), // Will be set by caller
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

	// Stream stdout in real-time (only progress dots now)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")

			// Check if this line contains only progress indicators
			isProgressLine := len(strings.TrimSpace(line)) > 0 &&
				strings.Trim(line, ".F*") == ""

			outputMutex.Lock()
			if isProgressLine {
				// Progress dots - print without newline
				fmt.Print(line)
			}
			// Skip "Finished in..." and other RSpec output
			outputMutex.Unlock()
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

			outputMutex.Lock()
			fmt.Fprintf(os.Stderr, "[%s] %s\n", specFile, line)
			outputMutex.Unlock()
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
		// RSpec returns exit code 1 for test failures, which we still consider a "successful" run
		// (meaning the test runner itself didn't crash)
		success = exitCode <= 1
	}

	// Parse JSON output
	var jsonOutput *RSpecJSONOutput
	var failures []FailureDetail
	var exampleCount, failureCount int

	if jsonFile != "" && success {
		jsonOutput, err = ParseRSpecJSON(jsonFile)
		if err != nil {
			// Log error but don't fail the whole test run
			outputMutex.Lock()
			fmt.Fprintf(os.Stderr, "[%s] Warning: Failed to parse JSON output: %v\n", specFile, err)
			outputMutex.Unlock()
		} else {
			failures = ExtractFailures(jsonOutput.Examples)
			exampleCount = jsonOutput.Summary.ExampleCount
			failureCount = jsonOutput.Summary.FailureCount
		}
	}

	// Clean up JSON file unless explicitly saving
	if !saveJSON && jsonFile != "" {
		os.Remove(jsonFile)
	}

	success = exitCode == 0

	return TestResult{
		SpecFile:     specFile,
		Success:      success,
		Output:       outputBuilder.String(),
		Error:        err,
		Duration:     time.Since(start),
		JSONOutput:   jsonOutput,
		Failures:     failures,
		ExampleCount: exampleCount,
		FailureCount: failureCount,
	}
}

// RunSpecsInParallel executes spec files in parallel using a worker pool
func RunSpecsInParallel(specFiles []string, dryRun bool, saveJSON bool, maxWorkers int) ([]TestResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()
	results := make(chan TestResult, len(specFiles))

	// Mutex to synchronize output from multiple processes
	var outputMutex sync.Mutex

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
				result := RunSpecFile(ctx, job.SpecFile, workerIndex, dryRun, saveJSON, &outputMutex)
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

	// Collect results
	var testResults []TestResult
	for result := range results {
		testResults = append(testResults, result)
	}

	return testResults, time.Since(start)
}

// RunDatabaseTask executes a Rails database task in parallel across test databases
func RunDatabaseTask(task string, workerCount int, dryRun bool) error {
	if dryRun {
		fmt.Printf("[dry-run] Would run database task '%s' with %d workers\n", task, workerCount)
		for i := 0; i < workerCount; i++ {
			testEnvNumber := GetTestEnvNumber(i)
			envStr := ""
			if testEnvNumber != "" {
				envStr = fmt.Sprintf("TEST_ENV_NUMBER=%s ", testEnvNumber)
			}
			fmt.Printf("[dry-run] Worker %d: %sRAILS_ENV=test bundle exec rake %s\n", i, envStr, task)
		}
		return nil
	}

	fmt.Printf("Running database task '%s' with %d workers...\n", task, workerCount)

	// Set up parallel execution
	ctx := context.Background()
	results := make(chan error, workerCount)
	var wg sync.WaitGroup

	// Run the task in parallel for each test database
	for i := 0; i < workerCount; i++ {
		workerIndex := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			testEnvNumber := GetTestEnvNumber(workerIndex)
			cmd := exec.CommandContext(ctx, "bundle", "exec", "rake", task)

			// Set environment variables
			cmd.Env = append(os.Environ(),
				"TEST_ENV_NUMBER="+testEnvNumber,
				"RAILS_ENV=test",
				"PARALLEL_TEST_GROUPS="+fmt.Sprintf("%d", workerCount),
			)

			if err := cmd.Run(); err != nil {
				results <- fmt.Errorf("worker %d failed: %v", workerIndex, err)
			} else {
				results <- nil
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Check for errors
	var errors []string
	for err := range results {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("database task failed:\n%s", strings.Join(errors, "\n"))
	}

	fmt.Printf("Database task '%s' completed successfully\n", task)
	return nil
}

// TestSummary represents the aggregated summary of all test results
type TestSummary struct {
	TotalExamples int
	TotalFailures int
	AllFailures   []FailureDetail
	TotalCPUTime  time.Duration
	WallTime      time.Duration
	HasFailures   bool
	ErroredFiles  []TestResult // Files that had errors running
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []TestResult, wallTime time.Duration) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []TestResult{},
	}

	for _, result := range results {
		summary.TotalCPUTime += result.Duration
		summary.TotalExamples += result.ExampleCount
		summary.TotalFailures += result.FailureCount

		if len(result.Failures) > 0 {
			summary.AllFailures = append(summary.AllFailures, result.Failures...)
			summary.HasFailures = true
		}

		if !result.Success {
			summary.HasFailures = true
			if result.Error != nil {
				summary.ErroredFiles = append(summary.ErroredFiles, result)
			}
		}
	}

	return summary
}

// PrintResults displays a test summary
func PrintResults(summary TestSummary) {
	fmt.Println() // New line after progress dots

	// Print failures if any
	if len(summary.AllFailures) > 0 {
		fmt.Println("\nFailures:")

		for i, failure := range summary.AllFailures {
			fmt.Print(FormatFailure(i+1, failure))
			fmt.Println() // Extra line between failures
		}
	}

	// Print summary like RSpec does
	fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
		summary.WallTime.Seconds(), summary.TotalCPUTime.Seconds())

	if summary.TotalFailures > 0 {
		fmt.Printf("%d examples, %d failures\n", summary.TotalExamples, summary.TotalFailures)
	} else {
		fmt.Printf("%d examples, 0 failures\n", summary.TotalExamples)
	}

	// Print failed examples summary
	if len(summary.AllFailures) > 0 {
		fmt.Println("\nFailed examples:")
		fmt.Print(FormatFailedExamples(summary.AllFailures))
	}

	// Show any spec files that had errors running
	if len(summary.ErroredFiles) > 0 {
		fmt.Println()
		for _, result := range summary.ErroredFiles {
			fmt.Printf("ERROR running %s: %v\n", result.SpecFile, result.Error)
		}
	}
}
