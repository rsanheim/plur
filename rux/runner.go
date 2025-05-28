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
func RunSpecFile(ctx context.Context, specFile string, workerIndex int, dryRun bool, saveJSON bool, colorOutput bool, outputMutex *sync.Mutex) TestResult {
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
	args := []string{"bundle", "exec", "rspec", "--format", "progress", "--format", "json", "--out", jsonFile}

	// Always use --no-color for RSpec since we'll add our own colors
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
				// Progress dots - print with our own colors (inlined for performance)
				if colorOutput {
					// Use strings.Builder for efficient string building
					var result strings.Builder
					result.Grow(len(line) * 2) // Pre-allocate for worst case (every char gets color codes)

					for _, char := range line {
						switch char {
						case '.':
							result.WriteString("\033[32m.\033[0m") // Green dot
						case 'F':
							result.WriteString("\033[31mF\033[0m") // Red F
						case '*':
							result.WriteString("\033[33m*\033[0m") // Yellow asterisk for pending
						default:
							result.WriteRune(char)
						}
					}
					fmt.Print(result.String())
				} else {
					fmt.Print(line)
				}
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

	// Parse JSON output - try regardless of exit code to get detailed error info
	var jsonOutput *RSpecJSONOutput
	var failures []FailureDetail
	var exampleCount, failureCount int

	if jsonFile != "" {
		jsonOutput, err = ParseRSpecJSON(jsonFile)
		if err != nil {
			// Provide different error messages based on exit code
			outputMutex.Lock()
			if exitCode > 1 {
				// Command failed before RSpec could run (likely bundle exec failure)
				if strings.Contains(outputBuilder.String(), "bundler:") || strings.Contains(outputBuilder.String(), "Bundler::") {
					fmt.Fprintf(os.Stderr, "[%s] Bundle exec failed - try running 'bundle install' first\n", specFile)
				} else {
					fmt.Fprintf(os.Stderr, "[%s] Command failed with exit code %d: %v\n", specFile, exitCode, err)
				}
			} else {
				// RSpec ran but JSON parsing failed
				fmt.Fprintf(os.Stderr, "[%s] Warning: Failed to parse JSON output: %v\n", specFile, err)
			}
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
func RunSpecsInParallel(specFiles []string, dryRun bool, saveJSON bool, colorOutput bool, maxWorkers int) ([]TestResult, time.Duration) {
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
				result := RunSpecFile(ctx, job.SpecFile, workerIndex, dryRun, saveJSON, colorOutput, &outputMutex)
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
