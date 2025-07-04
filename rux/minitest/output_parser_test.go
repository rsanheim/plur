package minitest

import (
	"io"
	"log/slog"
	"testing"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	logger.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors during tests
	}))
}

func TestOutputParser_StateTransitions(t *testing.T) {
	parser := &OutputParser{}

	// Initial state
	assert.Equal(t, Started, parser.state)

	// Transition to TestsRunning
	notifications, _ := parser.ParseLine("# Running:")
	require.Len(t, notifications, 1)
	assert.Equal(t, types.SuiteStarted, notifications[0].GetEvent())
	assert.Equal(t, TestsRunning, parser.state)

	// Transition to TestsComplete
	notifications, _ = parser.ParseLine("Finished in 0.001234s, 2430.1337 runs/s")
	assert.Empty(t, notifications)
	assert.Equal(t, TestsComplete, parser.state)

	// Transition to SummaryComplete
	notifications, _ = parser.ParseLine("3 runs, 3 assertions, 0 failures, 0 errors, 0 skips")
	require.Len(t, notifications, 1) // Just suite finished
	// Check notification is suite finished
	assert.Equal(t, types.SuiteFinished, notifications[0].GetEvent())
	assert.Equal(t, SummaryComplete, parser.state)
}

func TestOutputParser_ProgressParsing(t *testing.T) {
	t.Run("all passing", func(t *testing.T) {
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine("...")

		// Should create 3 progress events
		assert.Len(t, notifications, 3)
		for i, n := range notifications {
			assert.Equal(t, types.Progress, n.GetEvent())
			pe := n.(types.ProgressEvent)
			assert.Equal(t, ".", pe.Character)
			assert.Equal(t, i, pe.Index)
		}

		// Progress counts should be updated
		assert.Equal(t, 3, parser.progress.examples)
		assert.Equal(t, 3, parser.progress.passed)
		assert.Equal(t, 0, parser.progress.failed)
	})

	t.Run("mixed results", func(t *testing.T) {
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine("..F.F")

		// Should create 5 progress events
		assert.Len(t, notifications, 5)

		// Check each progress event
		assert.Equal(t, ".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal(t, ".", notifications[1].(types.ProgressEvent).Character)
		assert.Equal(t, "F", notifications[2].(types.ProgressEvent).Character)
		assert.Equal(t, ".", notifications[3].(types.ProgressEvent).Character)
		assert.Equal(t, "F", notifications[4].(types.ProgressEvent).Character)

		// Progress counts
		assert.Equal(t, 5, parser.progress.examples)
		assert.Equal(t, 3, parser.progress.passed)
		assert.Equal(t, 2, parser.progress.failed)

		// We no longer track failure indices
	})

	t.Run("with errors and skips", func(t *testing.T) {
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine(".FES")

		assert.Len(t, notifications, 4)
		assert.Equal(t, ".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal(t, "F", notifications[1].(types.ProgressEvent).Character)
		assert.Equal(t, "E", notifications[2].(types.ProgressEvent).Character)
		assert.Equal(t, "S", notifications[3].(types.ProgressEvent).Character)

		// We no longer track indices
	})
}

func TestOutputParser_FailureDetailMatching(t *testing.T) {
	// Setup: parser that has already processed progress line
	parser := &OutputParser{
		state:    TestsComplete,
		progress: ProgressCounts{examples: 5, passed: 3, failed: 2},
	}

	// Parse first failure header
	notifications, _ := parser.ParseLine("  1) Failure:")
	assert.Empty(t, notifications) // No notification yet
	assert.Equal(t, SummaryStarted, parser.state)

	// Parse failure details
	notifications, _ = parser.ParseLine("MixedResultsTest#test_email_validation [test/mixed_results_test.rb:54]:")
	assert.Empty(t, notifications) // Still accumulating

	// Parse failure message
	notifications, _ = parser.ParseLine("Expected false to be truthy.")
	assert.Empty(t, notifications) // Still accumulating

	// Empty line triggers notification creation
	notifications, _ = parser.ParseLine("")
	require.Len(t, notifications, 1) // Should create a new failure notification

	// Check the created notification
	failure := notifications[0].(types.TestCaseNotification)
	assert.Equal(t, types.TestFailed, failure.Event)
	assert.Equal(t, "MixedResultsTest#test_email_validation", failure.TestID) // Uses actual test name
	assert.Equal(t, "MixedResultsTest#test_email_validation", failure.Description)
	assert.Equal(t, "test/mixed_results_test.rb:54", failure.Location)
	assert.Equal(t, "Expected false to be truthy.", failure.Exception.Message)
}

func TestOutputParser_FullIntegration(t *testing.T) {
	parser := &OutputParser{}

	lines := []string{
		"Run options: --seed 58399",
		"",
		"# Running:",
		"",
		"FFF..F.",
		"",
		"Finished in 0.000586s, 11945.3917 runs/s, 18771.3298 assertions/s.",
		"",
		"  1) Failure:",
		"MixedResultsTest#test_display_name_failure [test/mixed_results_test.rb:46]:",
		`Expected: "john doe"`,
		`  Actual: "JOHN DOE"`,
		"",
		"  2) Failure:",
		"MixedResultsTest#test_type_error_will_fail [test/mixed_results_test.rb:70]:",
		`Expected: "25"`,
		`  Actual: 25`,
		"",
		"  3) Failure:",
		"MixedResultsTest#test_email_validation_mixed [test/mixed_results_test.rb:54]:",
		"Expected false to be truthy.",
		"",
		"  4) Failure:",
		"MixedResultsTest#test_nil_handling_error [test/mixed_results_test.rb:60]:",
		`Expected: ""`,
		`  Actual: nil`,
		"",
		"7 runs, 11 assertions, 4 failures, 0 errors, 0 skips",
	}

	var allNotifications []types.TestNotification
	var progressEvents []types.ProgressEvent
	var testCases []types.TestCaseNotification

	for _, line := range lines {
		notifications, _ := parser.ParseLine(line)
		for _, n := range notifications {
			allNotifications = append(allNotifications, n)
			if pe, ok := n.(types.ProgressEvent); ok {
				progressEvents = append(progressEvents, pe)
			} else if tc, ok := n.(types.TestCaseNotification); ok {
				testCases = append(testCases, tc)
			}
		}
	}

	// Should have:
	// 1 suite start
	// 7 progress events (from "FFF..F.")
	// 4 failure notifications (from failure details)
	// 1 suite finish
	// Total = 13 notifications (no passed test notifications)
	assert.Len(t, allNotifications, 13)

	// Check progress events
	assert.Len(t, progressEvents, 7)
	// Progress line was "FFF..F."
	assert.Equal(t, "F", progressEvents[0].Character)
	assert.Equal(t, "F", progressEvents[1].Character)
	assert.Equal(t, "F", progressEvents[2].Character)
	assert.Equal(t, ".", progressEvents[3].Character)
	assert.Equal(t, ".", progressEvents[4].Character)
	assert.Equal(t, "F", progressEvents[5].Character)
	assert.Equal(t, ".", progressEvents[6].Character)

	// Check test case notifications - only failures now
	assert.Len(t, testCases, 4) // 4 failures only

	// Count failures
	failureCount := 0
	for _, tc := range testCases {
		if tc.Event == types.TestFailed {
			failureCount++
			// Verify failure has details
			assert.NotEmpty(t, tc.Description)
			assert.NotEmpty(t, tc.Location)
			assert.NotNil(t, tc.Exception)
			assert.NotEmpty(t, tc.Exception.Message)
		}
	}
	assert.Equal(t, 4, failureCount)

	// Check suite summary
	var suite types.SuiteNotification
	for _, n := range allNotifications {
		if s, ok := n.(types.SuiteNotification); ok && s.Event == types.SuiteFinished {
			suite = s
			break
		}
	}
	assert.Equal(t, 7, suite.TestCount)
	assert.Equal(t, 4, suite.FailureCount)
	assert.Equal(t, 0, suite.PendingCount)
}
