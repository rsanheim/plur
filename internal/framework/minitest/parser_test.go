package minitest

import (
	"testing"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// progressChars returns the progress characters in order, and stdoutLines the
// content of every test-written stdout notification.
func classify(notifications []types.TestNotification) (progress string, stdout []string, raw []string) {
	for _, n := range notifications {
		switch v := n.(type) {
		case types.ProgressEvent:
			progress += v.Character
		case types.OutputNotification:
			if v.Event == types.TestStdout {
				stdout = append(stdout, v.Content)
			} else {
				raw = append(raw, v.Content)
			}
		}
	}
	return progress, stdout, raw
}

func TestOutputParser_TaggedStdout(t *testing.T) {
	t.Run("splits progress prefix from test output", func(t *testing.T) {
		parser := &outputParser{}

		notifications, consumed := parser.ParseLine("..PLUR_OUT:in test_titleize")
		assert.True(t, consumed, "tagged lines are consumed so they never reach rawOutput")

		progress, stdout, raw := classify(notifications)
		assert.Equal(t, "..", progress)
		assert.Equal(t, []string{"in test_titleize"}, stdout)
		assert.Empty(t, raw)
		assert.Equal(t, 2, parser.progressCount)
	})

	t.Run("no progress prefix", func(t *testing.T) {
		parser := &outputParser{}

		notifications, _ := parser.ParseLine("PLUR_OUT:hello")

		progress, stdout, _ := classify(notifications)
		assert.Empty(t, progress)
		assert.Equal(t, []string{"hello"}, stdout)
		assert.Equal(t, 0, parser.progressCount)
	})

	t.Run("preserves empty lines and leading whitespace", func(t *testing.T) {
		parser := &outputParser{}

		_, stdout, _ := classify(first(parser.ParseLine("PLUR_OUT:")))
		assert.Equal(t, []string{""}, stdout)

		_, stdout, _ = classify(first(parser.ParseLine("PLUR_OUT:   indented  ")))
		assert.Equal(t, []string{"   indented  "}, stdout)
	})

	t.Run("text that looks like progress is not counted as progress", func(t *testing.T) {
		parser := &outputParser{}

		notifications, _ := parser.ParseLine("...PLUR_OUT:.....")

		progress, stdout, _ := classify(notifications)
		assert.Equal(t, "...", progress)
		assert.Equal(t, []string{"....."}, stdout)
		assert.Equal(t, 3, parser.progressCount)
	})

	t.Run("puts FIRST_LINE does not leak an F into the progress line", func(t *testing.T) {
		parser := &outputParser{}

		notifications, _ := parser.ParseLine(".PLUR_OUT:FIRST_LINE")

		progress, stdout, _ := classify(notifications)
		assert.Equal(t, ".", progress)
		assert.Equal(t, []string{"FIRST_LINE"}, stdout)
	})

	t.Run("a test printing the marker keeps its own text intact", func(t *testing.T) {
		parser := &outputParser{}

		// The plugin tags the line, so plur's marker is always the first one.
		notifications, _ := parser.ParseLine("..PLUR_OUT:PLUR_OUT:sneaky")

		progress, stdout, _ := classify(notifications)
		assert.Equal(t, "..", progress)
		assert.Equal(t, []string{"PLUR_OUT:sneaky"}, stdout)
	})

	t.Run("non-progress prefix is preserved instead of miscounted", func(t *testing.T) {
		parser := &outputParser{}

		// minitest-reporters writes ANSI colored progress characters
		notifications, _ := parser.ParseLine("\033[32m.\033[0mPLUR_OUT:hello")

		progress, stdout, raw := classify(notifications)
		assert.Empty(t, progress, "an unrecognized prefix must not inflate the progress count")
		assert.Equal(t, []string{"hello"}, stdout)
		assert.Equal(t, []string{"\033[32m.\033[0m"}, raw)
	})

	t.Run("bare text is never mistaken for progress", func(t *testing.T) {
		parser := &outputParser{}

		// Untagged output, e.g. a subprocess writing straight to fd 1
		notifications, consumed := parser.ParseLine("..FROM_SUBPROCESS")

		assert.False(t, consumed, "untagged text stays in rawOutput")
		assert.Empty(t, notifications)
		assert.Equal(t, 0, parser.progressCount)
	})
}

func TestOutputParser_FullIntegrationWithTaggedStdout(t *testing.T) {
	parser := &outputParser{}

	lines := []string{
		"Run options: --seed 58399",
		"",
		"# Running:",
		"",
		".PLUR_OUT:in test_titleize",
		".....PLUR_OUT:in test_addition",
		"..",
		"",
		"Finished in 0.001234s, 2430.1337 runs/s, 4860.2674 assertions/s.",
		"",
		"8 runs, 23 assertions, 0 failures, 0 errors, 0 skips",
		// a trailing `print` with no newline is flushed after the summary
		"PLUR_OUT:trailing partial",
	}

	var all []types.TestNotification
	for _, line := range lines {
		notifications, _ := parser.ParseLine(line)
		all = append(all, notifications...)
	}

	progress, stdout, _ := classify(all)
	assert.Equal(t, "........", progress, "8 tests produce exactly 8 progress characters")
	assert.Equal(t, 8, parser.progressCount)
	assert.Equal(t, []string{"in test_titleize", "in test_addition", "trailing partial"}, stdout)

	var suite types.SuiteNotification
	for _, n := range all {
		if s, ok := n.(types.SuiteNotification); ok && s.Event == types.SuiteFinished {
			suite = s
		}
	}
	assert.Equal(t, 8, suite.TestCount)
	assert.Equal(t, 23, suite.AssertionCount)
}

// A failing run still has to classify minitest's own chrome, and the tagged
// output must not disturb the failure section or the summary.
func TestOutputParser_TaggedStdoutOnFailureRun(t *testing.T) {
	parser := &outputParser{}

	lines := []string{
		"Run options: --seed 1",
		"",
		"# Running:",
		"",
		".PLUR_OUT:FIRST_LINE",
		"F.",
		"",
		"Finished in 0.000668s, 11976.0480 runs/s, 31437.1261 assertions/s.",
		"",
		"  1) Failure:",
		"CalculatorTest#test_addition [test/calculator_test.rb:29]:",
		"Expected: 999",
		"  Actual: 5",
		"",
		"4 runs, 5 assertions, 1 failures, 0 errors, 0 skips",
	}

	var all []types.TestNotification
	for _, line := range lines {
		notifications, _ := parser.ParseLine(line)
		all = append(all, notifications...)
	}

	progress, stdout, _ := classify(all)
	assert.Equal(t, ".F.", progress)
	assert.Equal(t, []string{"FIRST_LINE"}, stdout)

	require.Len(t, parser.failures, 1)
	assert.Equal(t, "CalculatorTest#test_addition", parser.failures[0].TestID)

	var suite types.SuiteNotification
	for _, n := range all {
		if s, ok := n.(types.SuiteNotification); ok && s.Event == types.SuiteFinished {
			suite = s
		}
	}
	assert.Equal(t, types.SuiteFinished, suite.Event, "SuiteFinished must still fire")
	assert.Equal(t, 4, suite.TestCount)
	assert.Equal(t, 1, suite.FailureCount)
}

// The Ruby plugin and the Go parser have to agree on the marker.
func TestPluginMarkerMatchesParser(t *testing.T) {
	assert.Contains(t, plurPluginCode, `PREFIX = "`+stdoutMarker+`"`)
}

func first(notifications []types.TestNotification, _ bool) []types.TestNotification {
	return notifications
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
