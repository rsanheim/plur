package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

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
