package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/types"
)

// WorkerResult represents the accumulated results from a worker executing one or more test files
type WorkerResult struct {
	State          types.TestState
	Output         string
	Error          error
	Duration       time.Duration
	FileLoadTime   time.Duration
	ExampleCount   int
	AssertionCount int
	FailureCount   int
	ErrorCount     int
	PendingCount   int
	Tests          []types.TestCaseNotification // All test notifications

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
	Type        string // "dot", "failure", "pending", "error_progress", "error", "stderr", "stdout"
	Content     string
	CurrentFile string // Source file path (for rspec-trace mode, may be empty)
}

// TestSummary represents the aggregated summary of all test results
type TestSummary struct {
	TotalExamples     int
	TotalAssertions   int
	TotalFailures     int
	TotalErrors       int
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
func BuildTestSummary(results []WorkerResult, wallTime time.Duration, currentJob framework.Job) TestSummary {
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
		summary.TotalAssertions += result.AssertionCount
		summary.TotalFailures += result.FailureCount
		summary.TotalErrors += result.ErrorCount
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

const placeholder string = "‽"

// renumberSummaryOutput replaces the ‽ placeholders emitted by the Ruby
// formatter with real, sequential failure numbers. Each worker emits ‽ because
// it cannot know the global failure count, so plur assigns the final numbers
// here after aggregating output from every worker.
//
// Two marker shapes appear in RSpec output:
//
//	top-level:            "‽)"    -> next incrementing number, e.g. "3)"
//	aggregate sub-failure: "‽.1)" -> the parent failure's number, e.g. "3.1)"
//
// RSpec derives aggregate sub-indices from the number we pass to
// fully_formatted, so both shapes share the same ‽ placeholder; the sub-markers
// must inherit their parent's number rather than consume a new one.
func renumberSummaryOutput(output string) string {
	var b strings.Builder
	b.Grow(len(output))

	count := 0 // most recently assigned top-level failure number
	for i := 0; i < len(output); {
		rest, isMarker := strings.CutPrefix(output[i:], placeholder)
		if !isMarker {
			b.WriteByte(output[i])
			i++
			continue
		}

		switch {
		case strings.HasPrefix(rest, ")"):
			// top-level marker "‽)"
			count++
			b.WriteString(strconv.Itoa(count))
		case count > 0 && len(rest) >= 2 && rest[0] == '.' && rest[1] >= '0' && rest[1] <= '9':
			// aggregate sub-marker "‽.N)" inherits the parent's number
			b.WriteString(strconv.Itoa(count))
		default:
			// stray placeholder that is not a marker; leave it untouched
			b.WriteString(placeholder)
		}
		i += len(placeholder)
	}
	return b.String()
}

// PrintResults displays a test summary
func PrintResults(summary TestSummary, colorOutput bool, currentJob framework.Job) {
	parser := currentJob.Framework.Parser()

	// Print pending section first (RSpec outputs pending before failures)
	if summary.FormattedPending != "" {
		fmt.Print("\nPending: (Failures listed here are expected and do not affect your suite's status)\n")
		fmt.Print(renumberSummaryOutput(summary.FormattedPending))
	}

	// For minitest with failures, print the raw output which contains failure details
	if currentJob.Framework.Name == "minitest" && summary.HasFailures {
		// Collect all output from failed workers
		for _, result := range summary.AllResults {
			if result.State == types.StateFailed && result.Output != "" {
				// The raw output contains the failure details
				fmt.Print(result.Output)
			}
		}
	} else if summary.HasFailures && summary.FormattedFailures != "" {
		fmt.Print("\nFailures:\n")
		fmt.Print(renumberSummaryOutput(summary.FormattedFailures))
	}

	// Print summary
	summaryText := summary.FormattedSummary
	hasFormattedSummary := summaryText != ""
	if !hasFormattedSummary {
		suite := &types.SuiteNotification{
			TestCount:      summary.TotalExamples,
			AssertionCount: summary.TotalAssertions,
			FailureCount:   summary.TotalFailures,
			ErrorCount:     summary.TotalErrors,
			PendingCount:   summary.TotalPending,
		}
		summaryText = parser.FormatSummary(suite, summary.TotalExamples,
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
	if !hasFormattedSummary && currentJob.Framework.Name != "minitest" {
		// Skip for minitest since we already printed the raw output
		if failedList := parser.FormatFailuresList(summary.AllFailures); failedList != "" {
			fmt.Println("\nFailed examples:")
			fmt.Print(failedList)
		}
	}

	// Print errored files
	for _, result := range summary.ErroredFiles {
		if result.State != types.StateError {
			continue
		}
		if result.Output != "" {
			fmt.Print(result.Output)
			continue
		}
		if result.Error == nil {
			continue
		}
		if _, isExit := processExitCode(result.Error); !isExit {
			fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
		}
	}
}
