package minitest

import (
	"log/slog"
	"testing"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/types"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize logger for tests
	logger.Logger = slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors during tests
	}))
}

func TestOutputParser_StateTransitions(t *testing.T) {
	assert := assert.New(t)
	parser := &OutputParser{}

	// Initial state
	assert.Equal(Started, parser.state)

	// Transition to TestsRunning
	notifications, _ := parser.ParseLine("# Running:")
	assert.Len(notifications, 1)
	assert.Equal(types.SuiteStarted, notifications[0].GetEvent())
	assert.Equal(TestsRunning, parser.state)

	// Transition to TestsComplete
	notifications, _ = parser.ParseLine("Finished in 0.001234s, 2430.1337 runs/s")
	assert.Empty(notifications)
	assert.Equal(TestsComplete, parser.state)

	// Transition to SummaryComplete
	notifications, _ = parser.ParseLine("3 runs, 3 assertions, 0 failures, 0 errors, 0 skips")
	assert.Len(notifications, 1) // Just suite finished
	// Check notification is suite finished
	assert.Equal(types.SuiteFinished, notifications[0].GetEvent())
	assert.Equal(SummaryComplete, parser.state)
}

func TestOutputParser_ProgressParsing(t *testing.T) {
	t.Run("all passing", func(t *testing.T) {
		assert := assert.New(t)
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine("...")

		// Should create 3 progress events
		assert.Len(notifications, 3)
		for i, n := range notifications {
			assert.Equal(types.Progress, n.GetEvent())
			pe := n.(types.ProgressEvent)
			assert.Equal(".", pe.Character)
			assert.Equal(i, pe.Index)
		}

		// Progress counts should be updated
		assert.Equal(3, parser.progress.examples)
		assert.Equal(3, parser.progress.passed)
		assert.Equal(0, parser.progress.failed)
	})

	t.Run("mixed results", func(t *testing.T) {
		assert := assert.New(t)
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine("..F.F")

		// Should create 5 progress events
		assert.Len(notifications, 5)

		// Check each progress event
		assert.Equal(".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal(".", notifications[1].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[2].(types.ProgressEvent).Character)
		assert.Equal(".", notifications[3].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[4].(types.ProgressEvent).Character)

		// Progress counts
		assert.Equal(5, parser.progress.examples)
		assert.Equal(3, parser.progress.passed)
		assert.Equal(2, parser.progress.failed)
	})

	t.Run("with errors and skips", func(t *testing.T) {
		assert := assert.New(t)
		parser := &OutputParser{state: TestsRunning}
		notifications, _ := parser.ParseLine(".FES")

		assert.Len(notifications, 4)
		assert.Equal(".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[1].(types.ProgressEvent).Character)
		assert.Equal("E", notifications[2].(types.ProgressEvent).Character)
		assert.Equal("S", notifications[3].(types.ProgressEvent).Character)

		// We no longer track indices
	})
}

func TestOutputParser_FailureDetailMatching(t *testing.T) {
	assert := assert.New(t)
	// Setup: parser that has already processed progress line
	parser := &OutputParser{
		state:    TestsComplete,
		progress: ProgressCounts{examples: 5, passed: 3, failed: 2},
	}

	// Parse first failure header
	notifications, _ := parser.ParseLine("  1) Failure:")
	assert.Empty(notifications) // No notification yet
	assert.Equal(SummaryStarted, parser.state)

	// Parse failure details
	notifications, _ = parser.ParseLine("MixedResultsTest#test_email_validation [test/mixed_results_test.rb:54]:")
	assert.Empty(notifications) // Still accumulating

	// Parse failure message
	notifications, _ = parser.ParseLine("Expected false to be truthy.")
	assert.Empty(notifications) // Still accumulating

	// Empty line triggers notification creation
	notifications, _ = parser.ParseLine("")
	assert.Len(notifications, 1) // Should create a new failure notification

	// Check the created notification
	failure := notifications[0].(types.TestCaseNotification)
	assert.Equal(types.TestFailed, failure.Event)
	assert.Equal("MixedResultsTest#test_email_validation", failure.TestID) // Uses actual test name
	assert.Equal("MixedResultsTest#test_email_validation", failure.Description)
	assert.Equal("test/mixed_results_test.rb:54", failure.Location)
	assert.Equal("Expected false to be truthy.", failure.Exception.Message)
}

func TestOutputParser_FullIntegration(t *testing.T) {
	assert := assert.New(t)
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
	// 1 formatted failures notification
	// 1 suite finish
	// Total = 14 notifications
	assert.Len(allNotifications, 14)

	// Check progress events
	assert.Len(progressEvents, 7)
	// Progress line was "FFF..F."
	assert.Equal("F", progressEvents[0].Character)
	assert.Equal("F", progressEvents[1].Character)
	assert.Equal("F", progressEvents[2].Character)
	assert.Equal(".", progressEvents[3].Character)
	assert.Equal(".", progressEvents[4].Character)
	assert.Equal("F", progressEvents[5].Character)
	assert.Equal(".", progressEvents[6].Character)

	// Check test case notifications - only failures now
	assert.Len(testCases, 4) // 4 failures only

	// Count failures
	failureCount := 0
	for _, tc := range testCases {
		if tc.Event == types.TestFailed {
			failureCount++
			// Verify failure has details
			assert.NotEmpty(tc.Description)
			assert.NotEmpty(tc.Location)
			assert.NotNil(tc.Exception)
			assert.NotEmpty(tc.Exception.Message)
		}
	}
	assert.Equal(4, failureCount)

	// Check formatted failures notification
	var formattedFailures types.FormattedFailuresNotification
	foundFormattedFailures := false
	for _, n := range allNotifications {
		if ff, ok := n.(types.FormattedFailuresNotification); ok {
			formattedFailures = ff
			foundFormattedFailures = true
			break
		}
	}
	assert.True(foundFormattedFailures, "Should have a FormattedFailuresNotification")
	assert.Contains(formattedFailures.Content, "Failures:")
	assert.Contains(formattedFailures.Content, "MixedResultsTest#test_display_name_failure")

	// Check suite summary
	var suite types.SuiteNotification
	for _, n := range allNotifications {
		if s, ok := n.(types.SuiteNotification); ok && s.Event == types.SuiteFinished {
			suite = s
			break
		}
	}
	assert.Equal(7, suite.TestCount)
	assert.Equal(4, suite.FailureCount)
	assert.Equal(0, suite.PendingCount)
}
