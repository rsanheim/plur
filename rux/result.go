package main

import (
	"fmt"
	"time"

	"github.com/rsanheim/rux/types"
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
	Success           bool           // True if no failures and no errors
	ErroredFiles      []WorkerResult // Workers that had errors running tests
	Framework         TestFramework  // The test framework used
	TotalPending      int            // Total pending/skipped tests

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []WorkerResult, wallTime time.Duration) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []WorkerResult{},
		Success:      true, // Start assuming success
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

	// Print failures if any
	if summary.HasFailures && summary.FormattedFailures != "" {
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
	if !hasFormattedSummary {
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
