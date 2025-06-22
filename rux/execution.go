package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// TestExecutor orchestrates the execution of tests
type TestExecutor struct {
	config         *Config
	specFiles      []string
	runtimeTracker *RuntimeTracker
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(config *Config, specFiles []string) *TestExecutor {
	return &TestExecutor{
		config:         config,
		specFiles:      specFiles,
		runtimeTracker: NewRuntimeTracker(),
	}
}

// Execute runs the test execution based on the configuration
func (e *TestExecutor) Execute() error {
	fmt.Printf("rux version %s\n", GetVersionInfo())

	if e.config.DryRun {
		return e.executeDryRun()
	}

	return e.executeTests()
}

// executeDryRun handles the dry-run mode
func (e *TestExecutor) executeDryRun() error {
	if e.config.Auto {
		fmt.Fprintln(os.Stderr, "[dry-run] bundle install")
	}

	fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(e.specFiles))

	// Load runtime data if available
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	// Group files for execution
	var groups []FileGroup
	if len(runtimeData) > 0 {
		groups = GroupSpecFilesByRuntime(e.specFiles, e.config.WorkerCount, runtimeData)
		fmt.Fprintf(os.Stderr, "[dry-run] Using runtime-based grouped execution: %d groups\n", len(groups))
	} else {
		groups = GroupSpecFilesBySize(e.specFiles, e.config.WorkerCount)
		fmt.Fprintf(os.Stderr, "[dry-run] Using size-based grouped execution: %d groups\n", len(groups))
	}

	// Display what would be executed
	for i, group := range groups {
		args := e.buildTestCommand(group.Files)
		fmt.Fprintf(os.Stderr, "[dry-run] Worker %d: %s\n", i, strings.Join(args, " "))
	}

	return nil
}

// executeTests handles the actual test execution
func (e *TestExecutor) executeTests() error {
	actualWorkers := e.config.WorkerCount
	if len(e.specFiles) < e.config.WorkerCount {
		actualWorkers = len(e.specFiles)
	}

	fmt.Printf("Running %d spec files in parallel using %d workers (%d cores available)...\n",
		len(e.specFiles), actualWorkers, runtime.NumCPU())

	results, wallTime := RunSpecsInParallel(e.config, e.specFiles, e.runtimeTracker)

	// Save runtime data only if some tests actually ran
	hasValidRuntimeData := false
	for _, result := range results {
		if result.State != StateError && result.ExampleCount > 0 {
			hasValidRuntimeData = true
			break
		}
	}

	if hasValidRuntimeData {
		if err := e.runtimeTracker.SaveToFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save runtime data: %v\n", err)
		} else {
			if runtimePath, err := GetRuntimeFilePath(); err == nil {
				fmt.Fprintf(os.Stderr, "Runtime data saved to: %s\n", runtimePath)
			}
		}
	}

	// Build summary and print results
	summary := BuildTestSummary(results, wallTime)
	PrintResults(summary, e.config.ColorOutput)

	// Return error if tests failed
	if !summary.Success {
		return fmt.Errorf("test run failed: %d examples, %d failures",
			summary.TotalExamples, summary.TotalFailures)
	}

	return nil
}

// buildTestCommand constructs the test command arguments based on the framework
func (e *TestExecutor) buildTestCommand(files []string) []string {
	builder := NewCommandBuilder(e.config.Framework)
	return builder.BuildCommand(files, e.config)
}
