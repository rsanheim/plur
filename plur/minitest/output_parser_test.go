package minitest

import (
	"testing"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
)

func TestOutputParser_BasicFlow(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

	// Suite starts
	notifications, _ := parser.ParseLine("# Running:")
	assert.Len(notifications, 1)
	assert.Equal(types.SuiteStarted, notifications[0].GetEvent())

	// Progress indicators
	notifications, _ = parser.ParseLine("...")
	assert.Len(notifications, 3)

	// Ignore "Finished in" line
	notifications, _ = parser.ParseLine("Finished in 0.001234s, 2430.1337 runs/s")
	assert.Empty(notifications)

	// Summary line
	notifications, _ = parser.ParseLine("3 runs, 3 assertions, 0 failures, 0 errors, 0 skips")
	assert.Len(notifications, 1) // Just suite finished (no failures)
	assert.Equal(types.SuiteFinished, notifications[0].GetEvent())
	suite := notifications[0].(types.SuiteNotification)
	assert.Equal(3, suite.TestCount)
	assert.Equal(3, suite.AssertionCount)
	assert.Equal(0, suite.ErrorCount)
}

func TestOutputParser_SummaryIncludesErrors(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

	notifications, _ := parser.ParseLine("3 runs, 4 assertions, 1 failure, 2 errors, 0 skips")
	assert.Len(notifications, 1)
	suite := notifications[0].(types.SuiteNotification)
	assert.Equal(types.SuiteFinished, suite.Event)
	assert.Equal(3, suite.TestCount)
	assert.Equal(4, suite.AssertionCount)
	assert.Equal(1, suite.FailureCount)
	assert.Equal(2, suite.ErrorCount)
	assert.Equal(0, suite.PendingCount)
}

func TestOutputParser_SummaryWithErrorsOnly(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

	notifications, _ := parser.ParseLine("3 runs, 4 assertions, 0 failures, 2 errors, 0 skips")
	assert.Len(notifications, 1)
	suite := notifications[0].(types.SuiteNotification)
	assert.Equal(types.SuiteFinished, suite.Event)
	assert.Equal(3, suite.TestCount)
	assert.Equal(4, suite.AssertionCount)
	assert.Equal(0, suite.FailureCount)
	assert.Equal(2, suite.ErrorCount)
	assert.Equal(0, suite.PendingCount)
}

func TestOutputParser_ProgressParsing(t *testing.T) {
	t.Run("all passing", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}
		notifications, _ := parser.ParseLine("...")

		// Should create 3 progress events
		assert.Len(notifications, 3)
		for i, n := range notifications {
			assert.Equal(types.Progress, n.GetEvent())
			pe := n.(types.ProgressEvent)
			assert.Equal(".", pe.Character)
			assert.Equal(i, pe.Index)
		}

		// Progress count should be updated
		assert.Equal(3, parser.progressCount)
	})

	t.Run("mixed results", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}
		notifications, _ := parser.ParseLine("..F.F")

		// Should create 5 progress events
		assert.Len(notifications, 5)

		// Check each progress event
		assert.Equal(".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal(".", notifications[1].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[2].(types.ProgressEvent).Character)
		assert.Equal(".", notifications[3].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[4].(types.ProgressEvent).Character)

		// Progress count
		assert.Equal(5, parser.progressCount)
	})

	t.Run("with errors and skips", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}
		notifications, _ := parser.ParseLine(".FES")

		assert.Len(notifications, 4)
		assert.Equal(".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal("F", notifications[1].(types.ProgressEvent).Character)
		assert.Equal("E", notifications[2].(types.ProgressEvent).Character)
		assert.Equal("S", notifications[3].(types.ProgressEvent).Character)

		// Check indices
		assert.Equal(0, notifications[0].(types.ProgressEvent).Index)
		assert.Equal(1, notifications[1].(types.ProgressEvent).Index)
		assert.Equal(2, notifications[2].(types.ProgressEvent).Index)
		assert.Equal(3, notifications[3].(types.ProgressEvent).Index)
	})
}

func TestOutputParser_FailureDetailMatching(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

	// Parse first failure header - should start collecting
	notifications, _ := parser.ParseLine("  1) Failure:")
	assert.Empty(notifications)
	assert.True(parser.collectingFailures)

	// Parse failure details
	notifications, _ = parser.ParseLine("MixedResultsTest#test_email_validation [test/mixed_results_test.rb:54]:")
	assert.Empty(notifications) // Still accumulating

	// Parse failure message
	notifications, _ = parser.ParseLine("Expected false to be truthy.")
	assert.Empty(notifications) // Still accumulating

	// Empty line
	notifications, _ = parser.ParseLine("")
	assert.Empty(notifications) // Still accumulating

	// Summary line triggers extraction
	notifications, _ = parser.ParseLine("5 runs, 5 assertions, 2 failures, 0 errors, 0 skips")
	assert.Len(notifications, 2) // 1 failure TestCaseNotification + 1 SuiteNotification

	// Check that failures were extracted
	assert.Len(parser.failures, 1)
	failure := parser.failures[0]
	assert.Equal(types.TestFailed, failure.Event)
	assert.Equal("MixedResultsTest#test_email_validation", failure.TestID)
	assert.Equal("MixedResultsTest#test_email_validation", failure.Description)
	assert.Equal("test/mixed_results_test.rb:54", failure.Location)
	assert.Equal("Expected false to be truthy.", failure.Exception.Message)
}

func TestOutputParser_BothSummaryFormats(t *testing.T) {
	t.Run("runs format", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}

		// Standard minitest output uses "runs"
		notifications, _ := parser.ParseLine("5 runs, 13 assertions, 0 failures, 0 errors, 0 skips")
		assert.Len(notifications, 1)
		suite := notifications[0].(types.SuiteNotification)
		assert.Equal(types.SuiteFinished, suite.Event)
		assert.Equal(5, suite.TestCount)
		assert.Equal(13, suite.AssertionCount)
		assert.Equal(0, suite.FailureCount)
		assert.Equal(0, suite.ErrorCount)
		assert.Equal(0, suite.PendingCount)
	})

	t.Run("tests format", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}

		// Minitest::Reporters output uses "tests"
		notifications, _ := parser.ParseLine("2 tests, 2 assertions, 0 failures, 0 errors, 0 skips")
		assert.Len(notifications, 1)
		suite := notifications[0].(types.SuiteNotification)
		assert.Equal(types.SuiteFinished, suite.Event)
		assert.Equal(2, suite.TestCount)
		assert.Equal(2, suite.AssertionCount)
		assert.Equal(0, suite.FailureCount)
		assert.Equal(0, suite.ErrorCount)
		assert.Equal(0, suite.PendingCount)
	})

	t.Run("singular forms", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}

		// Test singular "run"
		notifications, _ := parser.ParseLine("1 run, 1 assertion, 0 failures, 0 errors, 0 skips")
		assert.Len(notifications, 1)

		// Test singular "test"
		notifications, _ = parser.ParseLine("1 test, 1 assertion, 1 failure, 0 errors, 0 skips")
		assert.Len(notifications, 1)
		suite := notifications[0].(types.SuiteNotification)
		assert.Equal(1, suite.TestCount)
		assert.Equal(1, suite.AssertionCount)
		assert.Equal(1, suite.FailureCount)
		assert.Equal(0, suite.ErrorCount)
	})
}

func TestOutputParser_FullIntegration(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

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
	// 4 failure TestCaseNotifications
	// 1 suite finish
	// Total = 13 notifications
	assert.Len(allNotifications, 13)

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

	// Check test case notifications - we now emit individual failure notifications
	assert.Len(testCases, 4) // 4 failure notifications

	// Check that failures were extracted and stored in parser
	if len(parser.failures) != 4 {
		t.Logf("Expected 4 failures, got %d", len(parser.failures))
		for i, f := range parser.failures {
			t.Logf("Failure %d: %s", i, f.TestID)
		}
	}
	assert.Len(parser.failures, 4) // 4 failures extracted by ExtractFailures

	// Check suite summary
	var suite types.SuiteNotification
	for _, n := range allNotifications {
		if s, ok := n.(types.SuiteNotification); ok && s.Event == types.SuiteFinished {
			suite = s
			break
		}
	}
	assert.Equal(7, suite.TestCount)
	assert.Equal(11, suite.AssertionCount)
	assert.Equal(4, suite.FailureCount)
	assert.Equal(0, suite.ErrorCount)
	assert.Equal(0, suite.PendingCount)
}

func TestNotificationToProgress(t *testing.T) {
	parser := &outputParser{}

	tests := []struct {
		char     string
		wantType string
	}{
		{".", "dot"},
		{"F", "failure"},
		{"E", "error_progress"},
		{"S", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.char, func(t *testing.T) {
			event := types.ProgressEvent{Event: types.Progress, Character: tt.char}
			gotType, ok := parser.NotificationToProgress(event)
			assert.True(t, ok)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

func TestCountLeadingProgressChars(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"..in test_titleize", 2},
		{"...in test_addition", 3},
		{"....", 4},
		{"in test_titleize", 0},
		{"", 0},
		{"F.text", 2},
		{".FES", 4},
		{"...", 3},
		{".hello", 1},
		{"  ..dots after spaces", 0},
		{"Finished in 0.001s", 1},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assert.Equal(t, tt.expected, countLeadingProgressChars(tt.line))
		})
	}
}

func TestOutputParser_MixedProgressLines(t *testing.T) {
	t.Run("leading dots with puts output", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}

		// Set up running phase
		parser.ParseLine("# Running:")

		notifications, consumed := parser.ParseLine("..in test_titleize")
		assert.False(consumed, "line should not be consumed so it goes to rawOutput")
		assert.Len(notifications, 2)
		assert.Equal(".", notifications[0].(types.ProgressEvent).Character)
		assert.Equal(0, notifications[0].(types.ProgressEvent).Index)
		assert.Equal(".", notifications[1].(types.ProgressEvent).Character)
		assert.Equal(1, notifications[1].(types.ProgressEvent).Index)
		assert.Equal(2, parser.progressCount)
	})

	t.Run("mixed line then pure dots", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{inRunningPhase: true}

		parser.ParseLine("..in test_addition")
		notifications, _ := parser.ParseLine("....")

		assert.Len(notifications, 4)
		assert.Equal(6, parser.progressCount) // 2 from mixed + 4 from pure
	})

	t.Run("not extracted outside running phase", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{} // inRunningPhase is false

		notifications, consumed := parser.ParseLine("..in test_titleize")
		assert.False(consumed)
		assert.Empty(notifications)
	})

	t.Run("no leading progress chars", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{inRunningPhase: true}

		notifications, _ := parser.ParseLine("in test_titleize")
		assert.Empty(notifications)
	})

	t.Run("Finished in line does not extract F as progress", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{inRunningPhase: true}

		notifications, _ := parser.ParseLine("Finished in 0.001234s, 2430.1337 runs/s")
		assert.Empty(notifications)
		assert.False(parser.inRunningPhase)
	})

	t.Run("inRunningPhase set by # Running: and cleared by failure header", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{}

		assert.False(parser.inRunningPhase)

		parser.ParseLine("# Running:")
		assert.True(parser.inRunningPhase)

		parser.ParseLine("  1) Failure:")
		assert.False(parser.inRunningPhase)
	})

	t.Run("inRunningPhase cleared by summary line", func(t *testing.T) {
		assert := assert.New(t)
		parser := &outputParser{inRunningPhase: true}

		parser.ParseLine("3 runs, 3 assertions, 0 failures, 0 errors, 0 skips")
		assert.False(parser.inRunningPhase)
	})
}

func TestOutputParser_FullIntegrationWithMixedProgress(t *testing.T) {
	assert := assert.New(t)
	parser := &outputParser{}

	lines := []string{
		"Run options: --seed 58399",
		"",
		"# Running:",
		"",
		"..in test_titleize",
		"..in test_addition",
		"....",
		"",
		"Finished in 0.001234s, 2430.1337 runs/s, 4860.2674 assertions/s.",
		"",
		"8 runs, 23 assertions, 0 failures, 0 errors, 0 skips",
	}

	var progressEvents []types.ProgressEvent

	for _, line := range lines {
		notifications, _ := parser.ParseLine(line)
		for _, n := range notifications {
			if pe, ok := n.(types.ProgressEvent); ok {
				progressEvents = append(progressEvents, pe)
			}
		}
	}

	// Should have 8 progress events total: 2 + 2 + 4
	assert.Len(progressEvents, 8)
	assert.Equal(8, parser.progressCount)

	// All should be dots
	for i, pe := range progressEvents {
		assert.Equal(".", pe.Character)
		assert.Equal(i, pe.Index, "progress index should be sequential")
	}
}

func TestOutputParser_FormatSummaryUsesAssertionAndErrorCounts(t *testing.T) {
	parser := &outputParser{}

	suite := &types.SuiteNotification{
		AssertionCount: 23,
		ErrorCount:     2,
	}
	summary := parser.FormatSummary(suite, 8, 1, 0, 1.2345, 0)
	assert.Contains(t, summary, "8 runs, 23 assertions, 1 failure, 2 errors, 0 skips")
}
