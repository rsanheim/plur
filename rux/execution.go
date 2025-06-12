package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"strings"

	"github.com/rsanheim/rux/rspec"
	"github.com/urfave/cli/v2"
)

// ExecutionConfig holds all configuration for a test execution
type ExecutionConfig struct {
	SpecFiles    []string
	DryRun       bool
	Auto         bool
	WorkerCount  int
	ColorOutput  bool
	RuntimeDir   string
	TraceEnabled bool
}

// TestExecutor orchestrates the execution of tests
type TestExecutor struct {
	config         *ExecutionConfig
	runtimeTracker *RuntimeTracker
}

// NewTestExecutor creates a new test executor
func NewTestExecutor(config *ExecutionConfig) *TestExecutor {
	return &TestExecutor{
		config:         config,
		runtimeTracker: NewRuntimeTracker(),
	}
}

// Execute runs the test execution based on the configuration
func (e *TestExecutor) Execute() error {
	// Print version as first line
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

	fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(e.config.SpecFiles))

	// Get formatter path for dry-run display
	cacheDir, err := getRuxCacheDir()
	var formatterPath string
	if err != nil {
		formatterPath = "~/.cache/rux/formatters/json_rows_formatter.rb"
	} else {
		formatterPath, err = rspec.GetFormatterPath(cacheDir)
		if err != nil {
			formatterPath = "~/.cache/rux/formatters/json_rows_formatter.rb"
		}
	}

	// Load runtime data if available
	runtimeData, err := LoadRuntimeData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
		runtimeData = make(map[string]float64)
	}

	// Group files for execution
	var groups []FileGroup
	if len(runtimeData) > 0 {
		groups = GroupSpecFilesByRuntime(e.config.SpecFiles, e.config.WorkerCount, runtimeData)
		fmt.Fprintf(os.Stderr, "[dry-run] Using runtime-based grouped execution: %d groups\n", len(groups))
	} else {
		groups = GroupSpecFilesBySize(e.config.SpecFiles, e.config.WorkerCount)
		fmt.Fprintf(os.Stderr, "[dry-run] Using size-based grouped execution: %d groups\n", len(groups))
	}

	// Display what would be executed
	for i, group := range groups {
		args := e.buildRSpecArgs(formatterPath, group.Files)
		fmt.Fprintf(os.Stderr, "[dry-run] Worker %d: %s\n", i, strings.Join(args, " "))
	}

	return nil
}

// executeTests handles the actual test execution
func (e *TestExecutor) executeTests() error {
	actualWorkers := e.config.WorkerCount
	if len(e.config.SpecFiles) < e.config.WorkerCount {
		actualWorkers = len(e.config.SpecFiles)
	}

	fmt.Printf("Running %d spec files in parallel using %d workers (%d cores available)...\n",
		len(e.config.SpecFiles), actualWorkers, runtime.NumCPU())

	// Run specs in parallel with intelligent grouping
	results, wallTime := RunSpecsInParallel(
		e.config.SpecFiles,
		false, // not dry-run
		e.config.ColorOutput,
		e.config.WorkerCount,
		e.runtimeTracker,
	)

	// Save runtime data
	if err := e.runtimeTracker.SaveToFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save runtime data: %v\n", err)
	} else {
		if runtimePath, err := GetRuntimeFilePath(); err == nil {
			fmt.Fprintf(os.Stderr, "Runtime data saved to: %s\n", runtimePath)
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

// buildRSpecArgs constructs the RSpec command arguments
func (e *TestExecutor) buildRSpecArgs(formatterPath string, files []string) []string {
	args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
	
	// Add color flags based on preference
	if !e.config.ColorOutput {
		args = append(args, "--no-color")
	} else {
		args = append(args, "--force-color", "--tty")
	}
	
	args = append(args, files...)
	return args
}

// BuildExecutionConfig creates ExecutionConfig from CLI context
func BuildExecutionConfig(ctx *cli.Context) (*ExecutionConfig, error) {
	// Discover spec files
	specFiles, err := discoverSpecFiles(ctx)
	if err != nil {
		return nil, err
	}

	// Set custom runtime directory if provided
	if runtimeDir := ctx.String("runtime-dir"); runtimeDir != "" {
		customRuntimeDir = runtimeDir
	}

	return &ExecutionConfig{
		SpecFiles:    specFiles,
		DryRun:       ctx.Bool("dry-run"),
		Auto:         ctx.Bool("auto"),
		WorkerCount:  GetWorkerCount(ctx.Int("n")),
		ColorOutput:  shouldUseColor(ctx),
		RuntimeDir:   ctx.String("runtime-dir"),
		TraceEnabled: ctx.Bool("trace"),
	}, nil
}

// discoverSpecFiles determines which spec files to run based on CLI context
func discoverSpecFiles(ctx *cli.Context) ([]string, error) {
	var specFiles []string
	var err error

	trace.WithRegion(context.Background(), "file_discovery", func() {
		if ctx.NArg() > 0 {
			// Expand glob patterns from provided arguments
			specFiles, err = ExpandGlobPatterns(ctx.Args().Slice())
			if err != nil {
				return
			}
			if len(specFiles) == 0 {
				err = fmt.Errorf("no spec files found matching provided patterns")
				return
			}
		} else {
			// Auto-discover spec files
			specFiles, err = FindSpecFiles()
			if err != nil {
				return
			}
			if len(specFiles) == 0 {
				err = fmt.Errorf("no spec files found")
				return
			}
		}
	})

	return specFiles, err
}