package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/rsanheim/rux/rspec"
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
		case StateFailed:
			summary.HasFailures = true
			summary.Success = false
			// Filter and append only failed test notifications
			for _, test := range result.Tests {
				if test.Event == types.TestFailed {
					summary.AllFailures = append(summary.AllFailures, test)
				}
			}
		case StateError:
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
	// Simple case: all tests passed
	if summary.Success {
		if summary.FormattedSummary != "" {
			fmt.Print(summary.FormattedSummary)
		} else {
			parser, err := NewTestOutputParser(summary.Framework)
			if err == nil {
				formattedSummary := parser.FormatSummary(nil, summary.TotalExamples, summary.TotalFailures, summary.TotalPending,
					summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())
				fmt.Print(formattedSummary)
			} else {
				// Fallback to generic formatting
				fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
					summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())
				fmt.Printf("%s, 0 failures\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)))
			}
			fmt.Println()
		}
		return
	}

	// Print failures if any
	if summary.FormattedFailures != "" {
		// Use RSpec's formatted failures (includes colors)
		fmt.Print(summary.FormattedFailures)
	} else if len(summary.AllFailures) > 0 {
		// Fall back to manual formatting
		fmt.Println("\nFailures:")

		for i, failure := range summary.AllFailures {
			fmt.Print(FormatTestFailure(i+1, failure))
			fmt.Println() // Extra line between failures
		}
	}

	// Print summary
	if summary.FormattedSummary != "" {
		// Use RSpec's formatted summary (includes timing, totals, and failed examples list)
		fmt.Print(summary.FormattedSummary)
	} else {
		// Use the parser to format the summary
		parser, err := NewTestOutputParser(summary.Framework)
		if err == nil {
			formattedSummary := parser.FormatSummary(nil, summary.TotalExamples, summary.TotalFailures, summary.TotalPending,
				summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())
			// Add color if needed for failures
			if summary.TotalFailures > 0 && colorOutput {
				fmt.Printf("\033[31m%s\033[0m\n", formattedSummary)
			} else if summary.TotalFailures == 0 && colorOutput {
				fmt.Printf("\033[32m%s\033[0m\n", formattedSummary)
			} else {
				fmt.Println(formattedSummary)
			}
		} else {
			// Fallback to generic formatting
			fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
				summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())

			if summary.TotalFailures > 0 {
				// Check if terminal supports color and format accordingly
				if colorOutput {
					fmt.Printf("\033[31m%s, %s\033[0m\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)), pluralize(summary.TotalFailures, "1 failure", fmt.Sprintf("%d failures", summary.TotalFailures)))
				} else {
					fmt.Printf("%s, %s\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)), pluralize(summary.TotalFailures, "1 failure", fmt.Sprintf("%d failures", summary.TotalFailures)))
				}
			} else {
				if colorOutput {
					fmt.Printf("\033[32m%s, 0 failures\033[0m\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)))
				} else {
					fmt.Printf("%s, 0 failures\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)))
				}
			}
		}

		// Print failed examples summary (only for RSpec)
		if len(summary.AllFailures) > 0 && summary.Framework == FrameworkRSpec {
			fmt.Println("\nFailed examples:")
			fmt.Print(FormatFailedExamples(summary.AllFailures))
		}
	}
	fmt.Println()

	// Show any spec files that had execution errors (not test failures)
	if len(summary.ErroredFiles) > 0 {
		hasExecutionErrors := false
		for _, result := range summary.ErroredFiles {
			if result.State == StateError {
				hasExecutionErrors = true
				break
			}
		}

		if hasExecutionErrors {
			fmt.Println()
			for _, result := range summary.ErroredFiles {
				if result.State == StateError {
					// Display the error output which contains the actual error details
					if result.Output != "" {
						// Output contains the full error details from RSpec
						fmt.Print(result.Output)
					}
					// Always show the Go error for debugging
					if result.Error != nil {
						fmt.Printf("ERROR running %s: %v\n", result.File.Path, result.Error)
					}
				}
			}
		}
	}
}

// FormatTestFailure formats a single test failure for display
func FormatTestFailure(index int, failure types.TestCaseNotification) string {
	var sb strings.Builder

	// Header line with failure number and description
	sb.WriteString(fmt.Sprintf("  %d) %s\n", index, failure.FullDescription))

	// Error/Failure line
	sb.WriteString("     Failure/Error: ")

	// Try to extract the failing line from the source file
	failingLine := rspec.ExtractFailingLine(failure.FilePath, failure.LineNumber)
	if failingLine != "" {
		sb.WriteString(failingLine)
		sb.WriteString("\n")
	} else {
		// If we can't read the file, just continue without the line
		sb.WriteString("\n")
	}

	// Error message - check if Exception exists
	if failure.Exception != nil {
		// For expectation failures, the message is already formatted with proper indentation
		lines := strings.Split(strings.TrimSpace(failure.Exception.Message), "\n")
		for _, line := range lines {
			if line != "" {
				sb.WriteString("       " + line + "\n")
			}
		}

		// Backtrace (first line only, like RSpec does)
		if len(failure.Exception.Backtrace) > 0 {
			sb.WriteString(fmt.Sprintf("     # %s", failure.Exception.Backtrace[0]))
		}
	}

	return sb.String()
}

// FormatFailedExamples formats the list of failed examples
func FormatFailedExamples(failures []types.TestCaseNotification) string {
	var sb strings.Builder

	for _, failure := range failures {
		sb.WriteString(fmt.Sprintf("rspec %s:%d # %s\n",
			failure.FilePath,
			failure.LineNumber,
			failure.FullDescription))
	}

	return sb.String()
}
