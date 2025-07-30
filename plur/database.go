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
func RunDatabaseTask(task string, config *GlobalConfig) error {
	if config.DryRun {
		fmt.Printf("[dry-run] Would run database task '%s' with %d workers\n", task, config.WorkerCount)
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

	// Run the task in parallel for each test database
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
