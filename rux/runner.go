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
	SpecFile string
	Success  bool
	Output   string
	Error    error
	Duration time.Duration
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

	args := []string{"bundle", "exec", "rspec", "--format", "progress", specFile}

	var jsonFile string
	if saveJSON {
		// Create temp file for JSON output
		tmpFile, err := os.CreateTemp("", "rux-results-*.json")
		if err != nil {
			return TestResult{
				SpecFile: specFile,
				Success:  false,
				Output:   "",
				Error:    fmt.Errorf("failed to create temp file: %v", err),
				Duration: time.Since(start),
			}
		}
		jsonFile = tmpFile.Name()
		tmpFile.Close()

		// Add JSON output args
		args = append(args, "--format", "json", "--out", jsonFile)
	}

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

	// Determine success
	success := err == nil
	if exitErr, ok := err.(*exec.ExitError); ok {
		// RSpec failed tests return exit code 1, which is still a "successful" run
		success = exitErr.ExitCode() <= 1
	}

	// Clean up JSON file if not saving
	if saveJSON && jsonFile != "" {
		defer os.Remove(jsonFile)
		// TODO: Could read and process JSON here if needed
	}

	return TestResult{
		SpecFile: specFile,
		Success:  success,
		Output:   outputBuilder.String(),
		Error:    err,
		Duration: time.Since(start),
	}
}

// RunTestsInParallel executes spec files in parallel using a worker pool
func RunTestsInParallel(specFiles []string, dryRun bool, saveJSON bool, maxWorkers int) ([]TestResult, time.Duration) {
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

// PrintResults displays a summary of test results
func PrintResults(results []TestResult, wallTime time.Duration) {
	fmt.Println() // New line after progress dots

	// Calculate total CPU time
	var totalCPUTime time.Duration
	passed := 0

	for _, result := range results {
		totalCPUTime += result.Duration
		if result.Success {
			passed++
		}
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Files: %d/%d passed\n", passed, len(results))
	fmt.Printf("Wall time: %v\n", wallTime)
	fmt.Printf("Total CPU time: %v\n", totalCPUTime)

	// Show failed files if any
	for _, result := range results {
		if !result.Success {
			fmt.Printf("FAILED: %s\n", result.SpecFile)
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
		}
	}
}
