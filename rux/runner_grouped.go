package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// RunSpecFiles executes multiple spec files in a single RSpec process
func RunSpecFiles(ctx context.Context, specFiles []string, workerIndex int, dryRun bool, saveJSON bool, outputChan chan<- OutputMessage) TestResult {
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

// RunSpecsInParallelGrouped executes spec files in parallel using intelligent grouping
func RunSpecsInParallelGrouped(specFiles []string, dryRun bool, saveJSON bool, colorOutput bool, maxWorkers int) ([]TestResult, time.Duration) {
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
				result := RunSpecFiles(ctx, files, workerIndex, dryRun, saveJSON, outputChan)
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
		}
		
		// Ensure newline after dots
		fmt.Println()
		
		return allResults, time.Since(start)
	} else {
		// Fall back to original one-file-per-process for large suites
		// or when we have more workers than files
		return RunSpecsInParallel(specFiles, dryRun, saveJSON, colorOutput, maxWorkers)
	}
}