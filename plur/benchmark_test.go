package main

import (
	"strings"
	"testing"

	"github.com/rsanheim/plur/types"
)

// BenchmarkTestCollectorRawOutput tests memory allocations in TestCollector's rawOutput string builder
func BenchmarkTestCollectorRawOutput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()

		// Simulate typical test output - 100 lines of output
		for j := 0; j < 100; j++ {
			collector.AddNotification(types.OutputNotification{
				Event:   types.RawOutput,
				Content: "This is a typical test output line with some content that might appear during test execution",
			})
		}

		// Force the string to be built
		_ = collector.rawOutput.String()
	}
}

// BenchmarkTestCollectorWithTests simulates a more realistic scenario with various test notifications
func BenchmarkTestCollectorWithTests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()

		// Simulate 50 test cases with mixed results
		for j := 0; j < 50; j++ {
			// Add some output
			collector.AddNotification(types.OutputNotification{
				Event:   types.RawOutput,
				Content: "Running test case...",
			})

			// Add test result (mix of passed, failed, pending)
			switch j % 3 {
			case 0:
				collector.AddNotification(types.TestCaseNotification{
					Event:           types.TestPassed,
					Description:     "test passes",
					FullDescription: "Test case passes correctly",
					FilePath:        "/path/to/test_spec.rb",
					LineNumber:      j + 1,
				})
			case 1:
				collector.AddNotification(types.TestCaseNotification{
					Event:           types.TestFailed,
					Description:     "test fails",
					FullDescription: "Test case fails with error",
					FilePath:        "/path/to/test_spec.rb",
					LineNumber:      j + 1,
				})
			default:
				collector.AddNotification(types.TestCaseNotification{
					Event:           types.TestPending,
					Description:     "test pending",
					FullDescription: "Test case is pending",
					FilePath:        "/path/to/test_spec.rb",
					LineNumber:      j + 1,
				})
			}
		}

		// Build the final result
		_ = collector.rawOutput.String()
	}
}

// BenchmarkStreamHelperStderr simulates stderr string building in stream_helper.go
func BenchmarkStreamHelperStderr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stderrBuilder strings.Builder

		// Simulate 50 stderr lines
		for j := 0; j < 50; j++ {
			line := "STDERR: Warning: This is a typical stderr output line that might appear during test execution\n"
			stderrBuilder.WriteString("STDERR: " + line + "\n")
		}

		// Force the string to be built
		_ = stderrBuilder.String()
	}
}

// BenchmarkStreamHelperStderrLarge tests with larger output volumes
func BenchmarkStreamHelperStderrLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var stderrBuilder strings.Builder

		// Simulate 500 stderr lines (large test suite with warnings)
		for j := 0; j < 500; j++ {
			line := "STDERR: Warning: Deprecation warning or other verbose output that Rails apps tend to produce during test runs\n"
			stderrBuilder.WriteString("STDERR: " + line + "\n")
		}

		// Force the string to be built
		_ = stderrBuilder.String()
	}
}

// BenchmarkTestCollectorSlices tests the slice allocations in TestCollector
func BenchmarkTestCollectorSlices(b *testing.B) {
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()

		// Add 100 test notifications to trigger slice growth
		for j := 0; j < 100; j++ {
			collector.AddNotification(types.TestCaseNotification{
				Event:           types.TestPassed,
				Description:     "test passes",
				FullDescription: "Test case passes correctly",
				FilePath:        "/path/to/test_spec.rb",
				LineNumber:      j + 1,
			})
		}
	}
}
