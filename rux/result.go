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
}

// BuildTestSummary collects and calculates summary data from test results
func BuildTestSummary(results []TestResult, wallTime time.Duration) TestSummary {
	summary := TestSummary{
		WallTime:     wallTime,
		ErroredFiles: []TestResult{},
		Success:      true, // Start assuming success
	}

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
	}

	return summary
}

// PrintResults displays a test summary
func PrintResults(summary TestSummary) {
	fmt.Println() // New line after progress dots

	// Simple case: all tests passed
	if summary.Success {
		fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
			summary.WallTime.Seconds(), summary.TotalCPUTime.Seconds())
		fmt.Printf("%d examples, 0 failures\n", summary.TotalExamples)
		return
	}

	// Print failures if any
	if len(summary.AllFailures) > 0 {
		fmt.Println("\nFailures:")

		for i, failure := range summary.AllFailures {
			fmt.Print(rspec.FormatFailure(i+1, failure))
			fmt.Println() // Extra line between failures
		}
	}

	// Print summary like RSpec does
	fmt.Printf("Finished in %.5f seconds (files took %.5f seconds to load)\n",
		summary.WallTime.Seconds(), summary.TotalCPUTime.Seconds())

	if summary.TotalFailures > 0 {
		fmt.Printf("%d examples, %s\n", summary.TotalExamples, pluralize(summary.TotalFailures, "1 failure", fmt.Sprintf("%d failures", summary.TotalFailures)))
	} else {
		fmt.Printf("%d examples, 0 failures\n", summary.TotalExamples)
	}

	// Print failed examples summary
	if len(summary.AllFailures) > 0 {
		fmt.Println("\nFailed examples:")
		fmt.Print(rspec.FormatFailedExamples(summary.AllFailures))
	}

	// Show any spec files that had errors running
	if len(summary.ErroredFiles) > 0 {
		fmt.Println()
		for _, result := range summary.ErroredFiles {
			fmt.Printf("ERROR running %s: %v\n", result.SpecFile, result.Error)
		}
	}
}
