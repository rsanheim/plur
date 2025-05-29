package main

import (
	"fmt"
	"time"

	"github.com/rsanheim/rux/rspec"
)

// TestSummary represents the aggregated summary of all test results
type TestSummary struct {
	TotalExamples int
	TotalFailures int
	AllFailures   []rspec.FailureDetail
	TotalCPUTime  time.Duration
	WallTime      time.Duration
	HasFailures   bool
	Success       bool         // True if no failures and no errors
	ErroredFiles  []TestResult // Files that had errors running

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []TestResult, wallTime time.Duration) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []TestResult{},
		Success:      true, // Start assuming success
	}

	// Track if we're in single-file mode (single worker)
	singleWorkerMode := len(results) == 1

	for _, result := range results {
		summary.TotalCPUTime += result.Duration
		summary.TotalExamples += result.ExampleCount
		summary.TotalFailures += result.FailureCount

		if len(result.Failures) > 0 {
			summary.AllFailures = append(summary.AllFailures, result.Failures...)
			summary.HasFailures = true
			summary.Success = false
		}

		if !result.Success {
			summary.HasFailures = true
			summary.Success = false
			if result.Error != nil {
				summary.ErroredFiles = append(summary.ErroredFiles, result)
			}
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
	// fmt.Println() // New line after progress dots

	// Simple case: all tests passed
	if summary.Success {
		// Use formatted summary if available, otherwise fall back to manual formatting
		if summary.FormattedSummary != "" {
			fmt.Print(summary.FormattedSummary)
		} else {
			fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
				summary.WallTime.Seconds(), summary.TotalCPUTime.Seconds())
			fmt.Printf("%s, 0 failures\n", pluralize(summary.TotalExamples, "1 example", fmt.Sprintf("%d examples", summary.TotalExamples)))
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
			fmt.Print(rspec.FormatFailure(i+1, failure))
			fmt.Println() // Extra line between failures
		}
	}

	// Print summary
	if summary.FormattedSummary != "" {
		// Use RSpec's formatted summary (includes timing, totals, and failed examples list)
		fmt.Print(summary.FormattedSummary)
	} else {
		// Fall back to manual formatting for parallel mode
		fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
			summary.WallTime.Seconds(), summary.TotalCPUTime.Seconds())

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

		// Print failed examples summary
		if len(summary.AllFailures) > 0 {
			fmt.Println("\nFailed examples:")
			fmt.Print(rspec.FormatFailedExamples(summary.AllFailures))
		}
	}
	fmt.Println()

	// RJS 2025-05-28 - this doesn't seem right - we are printing 'errors' when the spec is a legit failure
	// we probalby need proper error handling (i.e. an exception)
	//
	// Show any spec files that had errors running
	// if len(summary.ErroredFiles) > 0 {
	// 	fmt.Println()
	// 	for _, result := range summary.ErroredFiles {
	// 		fmt.Printf("ERROR running %s: %v\n", result.SpecFile, result.Error)
	// 	}
	// }
}
