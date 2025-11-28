package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

// TestExecutor orchestrates the execution of tests
type TestExecutor struct {
	globalConfig   *config.GlobalConfig
	testFiles      []string
	testLabel      string
	runtimeTracker *RuntimeTracker
	currentJob     job.Job
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(globalConfig *config.GlobalConfig, testFiles []string, currentJob job.Job) *TestExecutor {
	var label string
	switch currentJob.Name {
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
		currentJob:     currentJob,
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
	toStdErr(e.globalConfig.DryRun, "plur version version=%s\n", GetVersionInfo())

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
		logger.Logger.Debug("Using runtime-based grouped execution", "group_count", len(groups))
	} else {
		groups = GroupSpecFilesBySize(e.testFiles, e.globalConfig.WorkerCount)
		logger.Logger.Debug("Using size-based grouped execution", "group_count", len(groups))
	}

	// Display what would be executed
	for i, group := range groups {
		// Build command using framework-specific wrappers
		var args []string
		switch e.currentJob.Name {
		case "rspec":
			args = buildRSpecCommand(e.currentJob, group.Files, e.globalConfig)
		case "minitest":
			args = buildMinitestCommand(e.currentJob, group.Files, e.globalConfig)
		default:
			args = job.BuildJobCmd(e.currentJob, group.Files)
		}
		toStdErr(e.globalConfig.DryRun, "Worker %d: %s\n", i, strings.Join(args, " "))
	}

	return nil
}

// executeTests handles the actual test execution
func (e *TestExecutor) executeTests() error {
	e.summaryMsg()

	results, wallTime := RunTestsInParallel(e.globalConfig, e.testFiles, e.runtimeTracker, e.currentJob)

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
				logger.Logger.Debug("Runtime data saved", "runtime_path", runtimePath)
			}
		}
	}

	summary := BuildTestSummary(results, wallTime, e.currentJob)
	PrintResults(summary, e.globalConfig.ColorOutput, e.currentJob)

	// Return error if tests failed
	if !summary.Success {
		return fmt.Errorf("test run failed: %d examples, %d failures", summary.TotalExamples, summary.TotalFailures)
	}

	return nil
}
