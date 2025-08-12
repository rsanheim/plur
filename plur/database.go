package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// RunDatabaseTask executes a Rails database task in parallel across test databases
func RunDatabaseTask(task string, config *GlobalConfig) error {
	if config.DryRun {
		for i := 0; i < config.WorkerCount; i++ {
			envStr := ""
			if !config.IsSerial() {
				testEnvNumber := GetTestEnvNumber(i, config)
				envStr = fmt.Sprintf("TEST_ENV_NUMBER=%s ", testEnvNumber)
			}
			fmt.Printf("[dry-run] Worker %d: %sRAILS_ENV=test bundle exec rake %s\n", i, envStr, task)
		}
		return nil
	}

	fmt.Printf("Running database task '%s' with %d workers...\n", task, config.WorkerCount)

	// Set up parallel execution
	ctx := context.Background()
	results := make(chan error, config.WorkerCount)
	var wg sync.WaitGroup

	for i := 0; i < config.WorkerCount; i++ {
		workerIndex := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			cmd := exec.CommandContext(ctx, "bundle", "exec", "rake", task)

			// Set environment variables
			env := os.Environ()
			env = append(env, "RAILS_ENV=test")
			env = append(env, "PARALLEL_TEST_GROUPS="+fmt.Sprintf("%d", config.WorkerCount))

			// Only set TEST_ENV_NUMBER if not in serial mode
			if !config.IsSerial() {
				testEnvNumber := GetTestEnvNumber(workerIndex, config)
				env = append(env, "TEST_ENV_NUMBER="+testEnvNumber)
			}

			cmd.Env = env

			output, err := cmd.CombinedOutput()
			if err != nil {
				// Include the actual output in the error message
				results <- fmt.Errorf("worker %d failed: %v\nOutput:\n%s", workerIndex, err, string(output))
			} else {
				results <- nil
			}
		}()
	}

	wg.Wait()
	close(results)

	// Collect and deduplicate errors
	var errors []error
	errorOutputs := make(map[string][]int) // Map output to worker indices that had this error

	for err := range results {
		if err != nil {
			errors = append(errors, err)
			// Extract just the output part for deduplication
			errStr := err.Error()
			if idx := strings.Index(errStr, "\nOutput:\n"); idx != -1 {
				output := errStr[idx+9:] // Skip past "\nOutput:\n"
				workerMatch := regexp.MustCompile(`^worker (\d+) failed:`).FindStringSubmatch(errStr)
				if len(workerMatch) > 1 {
					if workerIdx, parseErr := strconv.Atoi(workerMatch[1]); parseErr == nil {
						errorOutputs[output] = append(errorOutputs[output], workerIdx)
					}
				}
			}
		}
	}

	if len(errors) > 0 {
		// Check if all errors have the same output
		if len(errorOutputs) == 1 && len(errors) == config.WorkerCount {
			// All workers failed with the same error
			for output, workers := range errorOutputs {
				return fmt.Errorf("database task failed:\nAll %d workers failed with the same error:\n%s",
					len(workers), output)
			}
		}

		// Different errors or not all workers failed - show each unique error
		var uniqueErrors []string
		for output, workers := range errorOutputs {
			if len(workers) == 1 {
				uniqueErrors = append(uniqueErrors, fmt.Sprintf("worker %d failed:\n%s", workers[0], output))
			} else {
				workerList := make([]string, len(workers))
				for i, w := range workers {
					workerList[i] = fmt.Sprintf("%d", w)
				}
				uniqueErrors = append(uniqueErrors, fmt.Sprintf("workers [%s] failed with:\n%s",
					strings.Join(workerList, ", "), output))
			}
		}
		return fmt.Errorf("database task failed:\n%s", strings.Join(uniqueErrors, "\n---\n"))
	}

	fmt.Printf("Database task '%s' completed successfully\n", task)
	return nil
}
