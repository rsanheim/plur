package main

import (
	"fmt"
	"time"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/types"
)

// TestFile represents a test file
type TestFile struct {
	Path     string // Full path to the file
	Filename string // Just the filename
}

// TestSummary represents the aggregated summary of all test results
type TestSummary struct {
	TotalExamples     int
	TotalFailures     int
	AllFailures       []types.TestCaseNotification
	TotalCPUTime      time.Duration
	WallTime          time.Duration
	TotalFileLoadTime time.Duration // Max file load time across all workers (since they run in parallel)
	HasFailures       bool
	Success           bool                 // True if no failures and no errors
	ErroredFiles      []WorkerResult       // Workers that had errors running tests
	Framework         config.TestFramework // The test framework used
	TotalPending      int                  // Total pending/skipped tests
	AllResults        []WorkerResult       // All worker results for accessing raw output

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []WorkerResult, wallTime time.Duration) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []WorkerResult{},
		AllResults:   results, // Store all results for raw output access
		Success:      true,    // Start assuming success
	}

	// Track if we're in single-file mode (single worker)
	singleWorkerMode := len(results) == 1

	for i, result := range results {
		summary.TotalCPUTime += result.Duration
		summary.TotalExamples += result.ExampleCount
		summary.TotalFailures += result.FailureCount
		summary.TotalPending += result.PendingCount

		// Set framework from first result (all should be the same)
		if i == 0 {
			summary.Framework = result.Framework
		}

		// Track the maximum file load time (since workers run in parallel)
		if result.FileLoadTime > summary.TotalFileLoadTime {
			summary.TotalFileLoadTime = result.FileLoadTime
		}

		// Check the result state to determine success/failure
		switch result.State {
		case types.StateFailed:
			summary.HasFailures = true
			summary.Success = false
			// Filter and append only failed test notifications
			for _, test := range result.Tests {
				if test.Event == types.TestFailed {
					summary.AllFailures = append(summary.AllFailures, test)
				}
			}
		case types.StateError:
			summary.HasFailures = true
			summary.Success = false
			summary.ErroredFiles = append(summary.ErroredFiles, result)
			// StateSuccess requires no action - summary.Success defaults to true
		}

		// Collect formatted failures (concatenate them)
		if result.FormattedFailures != "" {
			summary.FormattedFailures += result.FormattedFailures
		}
		// In single-worker mode, we can use the formatted summary directly
		if singleWorkerMode && result.FormattedSummary != "" {
			summary.FormattedSummary = result.FormattedSummary
		}
		// Note: We can't use FormattedSummary from individual workers in parallel mode
		// because each worker only knows about its own totals
	}

	return summary
}

// PrintResults displays a test summary
func PrintResults(summary TestSummary, colorOutput bool) {
	parser, err := NewTestOutputParser(summary.Framework)
	if err != nil {
		// Fallback to basic output
		fmt.Printf("%d examples, %d failures\n", summary.TotalExamples, summary.TotalFailures)
		return
	}

	// For minitest with failures, print the raw output which contains failure details
	if summary.Framework == config.FrameworkMinitest && summary.HasFailures {
		// Collect all output from failed workers
		for _, result := range summary.AllResults {
			if result.State == types.StateFailed && result.Output != "" {
				// The raw output contains the failure details
				fmt.Print(result.Output)
			}
		}
	} else if summary.HasFailures && summary.FormattedFailures != "" {
		// For RSpec, use the formatted failures
		fmt.Print(summary.FormattedFailures)
	}

	// Print summary
	summaryText := summary.FormattedSummary
	hasFormattedSummary := summaryText != ""
	if !hasFormattedSummary {
		summaryText = parser.FormatSummary(nil, summary.TotalExamples,
			summary.TotalFailures, summary.TotalPending,
			summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())
	}

	if colorOutput && !hasFormattedSummary {
		// Only colorize if we generated the summary ourselves
		summaryText = parser.ColorizeSummary(summaryText, summary.HasFailures)
	}
	fmt.Print(summaryText)
	fmt.Println()

	// Print failed examples list only if we didn't get a formatted summary
	// (RSpec's formatted summary already includes the failed examples list)
	if !hasFormattedSummary && summary.Framework != config.FrameworkMinitest {
		// Skip for minitest since we already printed the raw output
		if failedList := parser.FormatFailuresList(summary.AllFailures); failedList != "" {
			fmt.Println("\nFailed examples:")
			fmt.Print(failedList)
		}
	}

	// Print errored files
	for _, result := range summary.ErroredFiles {
		if result.State == types.StateError && result.Output != "" {
			fmt.Print(result.Output)
		}
	}
}
