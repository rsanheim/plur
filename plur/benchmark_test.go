package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rsanheim/plur/rspec"
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

// =============================================================================
// Test Data Generators for Scale Benchmarks
// =============================================================================

// generateSpecFiles creates realistic spec file paths
func generateSpecFiles(count int) []string {
	dirs := []string{"models", "controllers", "services", "lib", "helpers", "jobs", "mailers"}
	files := make([]string, count)
	for i := 0; i < count; i++ {
		dir := dirs[i%len(dirs)]
		files[i] = fmt.Sprintf("spec/%s/file_%04d_spec.rb", dir, i)
	}
	return files
}

// generateRuntimeData creates runtime data with Pareto distribution (80/20 rule)
// Some specs are 10-100x slower than median
func generateRuntimeData(files []string) map[string]float64 {
	data := make(map[string]float64, len(files))
	for i, file := range files {
		// 10% of files are slow (5-15 seconds), rest are fast (0.1-1.0 seconds)
		if i%10 == 0 {
			data[file] = 5.0 + float64(i%100)/10.0 // Slow: 5-15 seconds
		} else {
			data[file] = 0.1 + float64(i%10)/10.0 // Fast: 0.1-1.0 seconds
		}
	}
	return data
}

// generateRSpecJSONLines creates realistic RSpec JSON output lines
func generateRSpecJSONLines(testCount int, failureRate float64) []string {
	lines := make([]string, 0, testCount+10)
	failureThreshold := int(float64(testCount) * failureRate)

	// Load summary
	lines = append(lines, fmt.Sprintf(
		`PLUR_JSON:{"type":"load_summary","summary":{"count":%d,"load_time":1.5}}`,
		testCount,
	))

	// Test results
	for i := 0; i < testCount; i++ {
		status := "passed"
		msgType := "example_passed"
		if i < failureThreshold {
			status = "failed"
			msgType = "example_failed"
		}

		lines = append(lines, fmt.Sprintf(
			`PLUR_JSON:{"type":"%s","example":{"description":"test %d","full_description":"should work %d","file_path":"spec/model_spec.rb","line_number":%d,"status":"%s","run_time":0.05}}`,
			msgType, i, i, i+10, status,
		))
	}

	// Dump summary
	lines = append(lines, fmt.Sprintf(
		`PLUR_JSON:{"type":"dump_summary","example_count":%d,"failure_count":%d,"pending_count":0,"duration":5.5}`,
		testCount, failureThreshold,
	))

	return lines
}

// generateTestNotifications creates test case notifications
func generateTestNotifications(count int, failureRate float64) []types.TestCaseNotification {
	notifications := make([]types.TestCaseNotification, 0, count)
	failureThreshold := int(float64(count) * failureRate)

	for i := 0; i < count; i++ {
		event := types.TestPassed
		var exception *types.TestException

		if i < failureThreshold {
			event = types.TestFailed
			exception = &types.TestException{
				Class:     "RSpec::Expectations::ExpectationNotMetError",
				Message:   fmt.Sprintf("Expected %d to eq %d", i, i+1),
				Backtrace: []string{fmt.Sprintf("spec/model_spec.rb:%d", i+10)},
			}
		}

		notifications = append(notifications, types.TestCaseNotification{
			Event:           event,
			TestID:          fmt.Sprintf("test_%d", i),
			Description:     fmt.Sprintf("test case %d", i),
			FullDescription: fmt.Sprintf("Model test case %d should pass", i),
			FilePath:        fmt.Sprintf("spec/models/model_%d_spec.rb", i/10),
			LineNumber:      i + 10,
			Duration:        time.Duration(50+i%200) * time.Millisecond,
			Exception:       exception,
		})
	}
	return notifications
}

// =============================================================================
// Grouper Scale Benchmarks
// =============================================================================

func BenchmarkGroupSpecFilesBySize_1000Files(b *testing.B) {
	files := generateSpecFiles(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesBySize(files, 8)
	}
}

func BenchmarkGroupSpecFilesBySize_5000Files(b *testing.B) {
	files := generateSpecFiles(5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesBySize(files, 8)
	}
}

func BenchmarkGroupSpecFilesBySize_10000Files(b *testing.B) {
	files := generateSpecFiles(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesBySize(files, 16)
	}
}

func BenchmarkGroupSpecFilesByRuntime_1000Files(b *testing.B) {
	files := generateSpecFiles(1000)
	runtimeData := generateRuntimeData(files)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesByRuntime(files, 8, runtimeData)
	}
}

func BenchmarkGroupSpecFilesByRuntime_5000Files(b *testing.B) {
	files := generateSpecFiles(5000)
	runtimeData := generateRuntimeData(files)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesByRuntime(files, 8, runtimeData)
	}
}

func BenchmarkGroupSpecFilesByRuntime_10000Files(b *testing.B) {
	files := generateSpecFiles(10000)
	runtimeData := generateRuntimeData(files)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GroupSpecFilesByRuntime(files, 16, runtimeData)
	}
}

// =============================================================================
// RSpec Parser Scale Benchmarks
// =============================================================================

func BenchmarkRSpecParseLine_JSONEvent(b *testing.B) {
	parser := rspec.NewOutputParser()
	line := `PLUR_JSON:{"type":"example_passed","example":{"description":"test 1","full_description":"should work 1","file_path":"spec/model_spec.rb","line_number":10,"status":"passed","run_time":0.05}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseLine(line)
	}
}

func BenchmarkRSpecParseLine_RawOutput(b *testing.B) {
	parser := rspec.NewOutputParser()
	line := "Running tests... some typical test output that is not JSON"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.ParseLine(line)
	}
}

func BenchmarkRSpecParser_1000Tests(b *testing.B) {
	lines := generateRSpecJSONLines(1000, 0.02)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := rspec.NewOutputParser()
		for _, line := range lines {
			parser.ParseLine(line)
		}
	}
}

func BenchmarkRSpecParser_5000Tests(b *testing.B) {
	lines := generateRSpecJSONLines(5000, 0.02)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := rspec.NewOutputParser()
		for _, line := range lines {
			parser.ParseLine(line)
		}
	}
}

func BenchmarkRSpecParser_10000Tests(b *testing.B) {
	lines := generateRSpecJSONLines(10000, 0.02)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := rspec.NewOutputParser()
		for _, line := range lines {
			parser.ParseLine(line)
		}
	}
}

func BenchmarkRSpecParser_30000Tests(b *testing.B) {
	lines := generateRSpecJSONLines(30000, 0.02)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := rspec.NewOutputParser()
		for _, line := range lines {
			parser.ParseLine(line)
		}
	}
}

// =============================================================================
// TestCollector Scale Benchmarks
// =============================================================================

func BenchmarkTestCollector_1000Tests(b *testing.B) {
	notifications := generateTestNotifications(1000, 0.05)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()
		for _, n := range notifications {
			collector.AddNotification(n)
		}
		collector.BuildResult(5 * time.Second)
	}
}

func BenchmarkTestCollector_5000Tests(b *testing.B) {
	notifications := generateTestNotifications(5000, 0.05)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()
		for _, n := range notifications {
			collector.AddNotification(n)
		}
		collector.BuildResult(5 * time.Second)
	}
}

func BenchmarkTestCollector_10000Tests(b *testing.B) {
	notifications := generateTestNotifications(10000, 0.05)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()
		for _, n := range notifications {
			collector.AddNotification(n)
		}
		collector.BuildResult(5 * time.Second)
	}
}

func BenchmarkTestCollector_30000Tests(b *testing.B) {
	notifications := generateTestNotifications(30000, 0.05)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewTestCollector()
		for _, n := range notifications {
			collector.AddNotification(n)
		}
		collector.BuildResult(5 * time.Second)
	}
}
