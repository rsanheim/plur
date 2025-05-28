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

// ExpandGlobPatterns takes a list of file paths/patterns and expands any glob patterns
// Supports ** for recursive directory matching like Ruby's Dir.glob
func ExpandGlobPatterns(patterns []string) ([]string, error) {
	var allFiles []string
	seenFiles := make(map[string]bool)

	for _, pattern := range patterns {
		// Check if pattern contains glob characters
		if strings.ContainsAny(pattern, "*?[") {
			// Handle ** for recursive matching
			if strings.Contains(pattern, "**") {
				matches, err := expandDoubleStarGlob(pattern)
				if err != nil {
					return nil, fmt.Errorf("error expanding glob pattern %q: %v", pattern, err)
				}

				for _, match := range matches {
					if !seenFiles[match] {
						allFiles = append(allFiles, match)
						seenFiles[match] = true
					}
				}
			} else {
				// Use standard glob for patterns without **
				matches, err := filepath.Glob(pattern)
				if err != nil {
					return nil, fmt.Errorf("error expanding glob pattern %q: %v", pattern, err)
				}

				// Filter to only include _spec.rb files
				for _, match := range matches {
					if strings.HasSuffix(match, "_spec.rb") && !seenFiles[match] {
						allFiles = append(allFiles, match)
						seenFiles[match] = true
					}
				}
			}
		} else {
			// Not a glob pattern, check if it's a valid spec file
			if _, err := os.Stat(pattern); err == nil {
				if strings.HasSuffix(pattern, "_spec.rb") && !seenFiles[pattern] {
					allFiles = append(allFiles, pattern)
					seenFiles[pattern] = true
				} else if !strings.HasSuffix(pattern, "_spec.rb") {
					// Warn about non-spec files
					fmt.Fprintf(os.Stderr, "Warning: %s does not end with _spec.rb\n", pattern)
				}
			} else {
				return nil, fmt.Errorf("file not found: %s", pattern)
			}
		}
	}

	return allFiles, nil
}

// expandDoubleStarGlob handles ** glob patterns for recursive directory matching
func expandDoubleStarGlob(pattern string) ([]string, error) {
	// Split pattern into parts
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Multiple ** not supported
		return nil, fmt.Errorf("multiple ** in pattern not supported: %s", pattern)
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// If prefix is empty, start from current directory
	if prefix == "" {
		prefix = "."
	}

	var matches []string

	// Walk the directory tree starting from prefix
	err := filepath.WalkDir(prefix, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories unless suffix is empty
		if d.IsDir() && suffix != "" {
			return nil
		}

		// Check if the path matches the suffix pattern
		if suffix != "" {
			// Get the relative path from the prefix
			relPath, err := filepath.Rel(prefix, path)
			if err != nil {
				return nil
			}

			// Check if the relative path matches the suffix pattern
			_, err = filepath.Match(suffix, relPath)
			if err != nil {
				return nil
			}

			// Also check if any parent directory + suffix matches
			// This handles cases like spec/**/models/*_spec.rb
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for i := range pathParts {
				subPath := filepath.Join(pathParts[i:]...)
				if matched, _ := filepath.Match(suffix, subPath); matched {
					if strings.HasSuffix(path, "_spec.rb") {
						matches = append(matches, path)
						return nil
					}
				}
			}
		} else if strings.HasSuffix(path, "_spec.rb") {
			// No suffix, just match all _spec.rb files
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
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

// RunSpecFile executes multiple spec files in a single RSpec process
func RunSpecFile(ctx context.Context, specFiles []string, workerIndex int, dryRun bool, saveJSON bool, outputChan chan<- OutputMessage) TestResult {
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
		return TestResult{
			SpecFile: strings.Join(specFiles, ","),
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
		return TestResult{
			SpecFile: strings.Join(specFiles, ","),
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stdout pipe: %v", err),
			Duration: time.Since(start),
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return TestResult{
			SpecFile: strings.Join(specFiles, ","),
			Success:  false,
			Output:   "",
			Error:    fmt.Errorf("failed to create stderr pipe: %v", err),
			Duration: time.Since(start),
		}
	}

	// Start the command
	func() {
		defer TraceFuncWithWorker("process_spawn", workerIndex, fmt.Sprintf("%d files", len(specFiles)))()
		err = cmd.Start()
	}()
	if err != nil {
		return TestResult{
			SpecFile: strings.Join(specFiles, ","),
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

			msg, err := ParseJSONMessage(line)
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
func RunSpecsInParallel(specFiles []string, dryRun bool, saveJSON bool, colorOutput bool, maxWorkers int, runtimeTracker *RuntimeTracker) ([]TestResult, time.Duration) {
	defer TraceFunc("run_specs_parallel_grouped")()
	start := time.Now()
	ctx := context.Background()

	// Decide whether to use grouping
	useGrouping := ShouldUseGrouping(len(specFiles), maxWorkers)

	if useGrouping {
		fmt.Fprintf(os.Stderr, "Using grouped execution: %d files across %d workers\n", len(specFiles), maxWorkers)
		// Group files by size
		groups := GroupSpecFilesBySize(specFiles, maxWorkers)

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
				result := RunSpecFile(ctx, files, workerIndex, dryRun, saveJSON, outputChan)
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
	} else {
		// Fall back to one-file-per-process for large suites
		// or when we have more workers than files
		return runSpecsInParallelSingle(specFiles, dryRun, saveJSON, colorOutput, maxWorkers, runtimeTracker, ctx, start)
	}
}

// runSpecsInParallelSingle runs specs with one file per process (original mode)
func runSpecsInParallelSingle(specFiles []string, dryRun bool, saveJSON bool, colorOutput bool, maxWorkers int, runtimeTracker *RuntimeTracker, ctx context.Context, start time.Time) ([]TestResult, time.Duration) {
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
	func() {
		defer TraceFuncWithMetadata("worker_pool_init", map[string]interface{}{
			"worker_count": maxWorkers,
			"spec_count":   len(specFiles),
		})()

		for i := 0; i < maxWorkers; i++ {
			workerIndex := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobs {
					result := RunSpecFile(ctx, []string{job.SpecFile}, workerIndex, dryRun, saveJSON, outputChan)
					results <- result
				}
			}()
		}
	}()

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
		// Track runtime data if tracker is available
		if runtimeTracker != nil && result.JSONOutput != nil {
			for _, example := range result.JSONOutput.Examples {
				runtimeTracker.AddExample(example)
			}
		}
	}

	return testResults, time.Since(start)
}
