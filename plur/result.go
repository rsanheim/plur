package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/types"
)

// WorkerResult represents the accumulated results from a worker executing one or more test files
type WorkerResult struct {
	State        types.TestState
	Output       string
	Error        error
	Duration     time.Duration
	FileLoadTime time.Duration
	ExampleCount int
	FailureCount int
	PendingCount int
	Tests        []types.TestCaseNotification // All test notifications

	// Formatted output from RSpec
	FormattedFailures string
	FormattedPending  string
	FormattedSummary  string
}

// Success returns true if the test execution was successful (no failures or errors)
func (r WorkerResult) Success() bool {
	return r.State == types.StateSuccess
}

// OutputMessage is a message from workers for output aggregation
type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr", "stdout"
	Content  string
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
	TotalPending      int            // Total pending/skipped tests
	AllResults        []WorkerResult // All worker results for accessing raw output

	// Formatted output from RSpec
	FormattedFailures string
	FormattedPending  string
	FormattedSummary  string
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []WorkerResult, wallTime time.Duration, currentJob job.Job) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []WorkerResult{},
		AllResults:   results, // Store all results for raw output access
		Success:      true,    // Start assuming success
	}

	// Track if we're in single-file mode (single worker)
	singleWorkerMode := len(results) == 1

	for _, result := range results {
		summary.TotalCPUTime += result.Duration
		summary.TotalExamples += result.ExampleCount
		summary.TotalFailures += result.FailureCount
		summary.TotalPending += result.PendingCount

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

		// Collect formatted failures and pending (concatenate them)
		if result.FormattedFailures != "" {
			summary.FormattedFailures += result.FormattedFailures
		}
		if result.FormattedPending != "" {
			summary.FormattedPending += result.FormattedPending
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

// renumberFailures replaces {{FNUM}} placeholders with incrementing numbers.
// The Ruby formatter outputs {{FNUM}} instead of actual numbers so plur can
// correctly number failures after aggregating from multiple workers.
func renumberFailures(output string) string {
	count := 0
	for strings.Contains(output, "{{FNUM}}") {
		count++
		output = strings.Replace(output, "{{FNUM}}", strconv.Itoa(count), 1)
	}
	return output
}

// PrintResults displays a test summary
func PrintResults(summary TestSummary, colorOutput bool, currentJob job.Job) {
	parser, err := currentJob.CreateParser()
	if err != nil {
		// Fallback to basic output
		fmt.Printf("%d examples, %d failures\n", summary.TotalExamples, summary.TotalFailures)
		return
	}

	// Print pending section first (RSpec outputs pending before failures)
	if summary.FormattedPending != "" {
		fmt.Print(summary.FormattedPending)
	}

	// For minitest with failures, print the raw output which contains failure details
	if currentJob.IsMinitestStyle() && summary.HasFailures {
		// Collect all output from failed workers
		for _, result := range summary.AllResults {
			if result.State == types.StateFailed && result.Output != "" {
				// The raw output contains the failure details
				fmt.Print(result.Output)
			}
		}
	} else if summary.HasFailures && summary.FormattedFailures != "" {
		// Add single "Failures:" header and renumber {{FNUM}} placeholders
		fmt.Print("\nFailures:\n")
		fmt.Print(renumberFailures(summary.FormattedFailures))
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
	if !hasFormattedSummary && !currentJob.IsMinitestStyle() {
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
