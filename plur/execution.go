package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/types"
)

// TestExecutor orchestrates the execution of tests
type TestExecutor struct {
	globalConfig   *config.GlobalConfig
	testFiles      []string
	testLabel      string
	runtimeTracker *RuntimeTracker
	currentTask    *task.Task
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(globalConfig *config.GlobalConfig, testFiles []string, currentTask *task.Task) *TestExecutor {
	var label string
	switch currentTask.Name {
	case "rspec":
		label = pluralize(len(testFiles), "spec", "specs")
	case "minitest":
		label = pluralize(len(testFiles), "test", "tests")
	default:
		label = pluralize(len(testFiles), "test", "tests")
	}
	return &TestExecutor{
		globalConfig:   globalConfig,
		testFiles:      testFiles,
		testLabel:      label,
		runtimeTracker: NewRuntimeTracker(),
		currentTask:    currentTask,
	}
}

func (e *TestExecutor) summaryMsg() {
	actualWorkers := e.globalConfig.WorkerCount
	if len(e.testFiles) < e.globalConfig.WorkerCount {
		actualWorkers = len(e.testFiles)
	}

	toStdErr(e.globalConfig.DryRun, "Running %d %s in parallel using %d workers\n",
		len(e.testFiles), e.testLabel, actualWorkers)
}

// Execute runs the test execution based on the configuration
func (e *TestExecutor) Execute() error {
	fmt.Printf("plur version version=%s\n", GetVersionInfo())

	if e.globalConfig.DryRun {
		return e.executeDryRun()
	}

	return e.executeTests()
}

// executeDryRun handles the dry-run mode
func (e *TestExecutor) executeDryRun() error {
	if e.globalConfig.Auto {
		toStdErr(true, "bundle install\n")
	}

	e.summaryMsg()

	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	var groups []FileGroup
	if len(runtimeData) > 0 {
		groups = GroupSpecFilesByRuntime(e.testFiles, e.globalConfig.WorkerCount, runtimeData)
		toStdErr(e.globalConfig.DryRun, "Using runtime-based grouped execution: %d groups\n", len(groups))
	} else {
		groups = GroupSpecFilesBySize(e.testFiles, e.globalConfig.WorkerCount)
		toStdErr(e.globalConfig.DryRun, "Using size-based grouped execution: %d groups\n", len(groups))
	}

	// Display what would be executed
	for i, group := range groups {
		args := e.currentTask.BuildCommand(group.Files, e.globalConfig, "")
		toStdErr(e.globalConfig.DryRun, "Worker %d: %s\n", i, strings.Join(args, " "))
	}

	return nil
}

// executeTests handles the actual test execution
func (e *TestExecutor) executeTests() error {
	e.summaryMsg()

	results, wallTime := RunTestsInParallel(e.globalConfig, e.testFiles, e.runtimeTracker, e.currentTask)

	// Save runtime data only if some tests actually ran
	hasValidRuntimeData := false
	for _, result := range results {
		if result.State != types.StateError && result.ExampleCount > 0 {
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
	summary := BuildTestSummary(results, wallTime, e.currentTask)
	PrintResults(summary, e.globalConfig.ColorOutput, e.currentTask)

	// Return error if tests failed
	if !summary.Success {
		return fmt.Errorf("test run failed: %d examples, %d failures",
			summary.TotalExamples, summary.TotalFailures)
	}

	return nil
}
