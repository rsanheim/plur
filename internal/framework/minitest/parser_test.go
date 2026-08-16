package minitest

import (
	"testing"
	"time"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLine_SuiteStarted(t *testing.T) {
	parser := NewOutputParser()

	notifications, consumed := parser.ParseLine(`PLUR_JSON:{"type":"suite_started"}`)

	assert.True(t, consumed)
	require.Len(t, notifications, 1)
	suite := notifications[0].(types.SuiteNotification)
	assert.Equal(t, types.SuiteStarted, suite.Event)
	assert.Greater(t, suite.LoadTime, time.Duration(0))
}

func TestParseLine_TestResults(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		event    types.TestEvent
		status   string
		progress string
	}{
		{
			name:     "passing test",
			line:     `PLUR_JSON:{"type":"test_result","status":"passed","id":"FooTest#test_pass","file_path":"test/foo_test.rb","line_number":5,"run_time":0.25}`,
			event:    types.TestPassed,
			status:   "passed",
			progress: "dot",
		},
		{
			name:     "failing test",
			line:     `PLUR_JSON:{"type":"test_result","status":"failed","id":"FooTest#test_fail","file_path":"test/foo_test.rb","line_number":9,"run_time":0.25}`,
			event:    types.TestFailed,
			status:   "failed",
			progress: "failure",
		},
		{
			name:     "erroring test",
			line:     `PLUR_JSON:{"type":"test_result","status":"error","id":"FooTest#test_boom","file_path":"test/foo_test.rb","line_number":13,"run_time":0.25}`,
			event:    types.TestFailed,
			status:   "error",
			progress: "error_progress",
		},
		{
			name:     "skipped test",
			line:     `PLUR_JSON:{"type":"test_result","status":"skipped","id":"FooTest#test_skip","file_path":"test/foo_test.rb","line_number":17,"run_time":0.25}`,
			event:    types.TestPending,
			status:   "skipped",
			progress: "pending",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewOutputParser()

			notifications, consumed := parser.ParseLine(tc.line)

			assert.True(t, consumed)
			require.Len(t, notifications, 1)
			test := notifications[0].(types.TestCaseNotification)
			assert.Equal(t, tc.event, test.Event)
			assert.Equal(t, tc.status, test.Status)
			assert.Equal(t, "test/foo_test.rb", test.FilePath)
			assert.Equal(t, 250*time.Millisecond, test.Duration)

			progress, isProgress := parser.NotificationToProgress(test)
			assert.True(t, isProgress)
			assert.Equal(t, tc.progress, progress)
		})
	}
}

func TestParseLine_TestResultCarriesIdentityAndLocation(t *testing.T) {
	parser := NewOutputParser()

	line := `PLUR_JSON:{"type":"test_result","status":"passed","id":"FooTest#test_pass","file_path":"test/foo_test.rb","line_number":5,"run_time":0.01}`
	notifications, _ := parser.ParseLine(line)

	require.Len(t, notifications, 1)
	test := notifications[0].(types.TestCaseNotification)
	assert.Equal(t, "FooTest#test_pass", test.TestID)
	assert.Equal(t, "FooTest#test_pass", test.FullDescription)
	assert.Equal(t, "test/foo_test.rb:5", test.Location)
	assert.Equal(t, 5, test.LineNumber)
}

func TestParseLine_TestResultWithoutSourceLocation(t *testing.T) {
	// Dynamically defined tests can lack a resolvable source_location; the
	// reporter then omits file_path and the tracker skips the test.
	parser := NewOutputParser()

	line := `PLUR_JSON:{"type":"test_result","status":"passed","id":"EvalTest#test_dynamic","file_path":null,"line_number":null,"run_time":0.01}`
	notifications, consumed := parser.ParseLine(line)

	assert.True(t, consumed)
	require.Len(t, notifications, 1)
	test := notifications[0].(types.TestCaseNotification)
	assert.Equal(t, "", test.FilePath)
	assert.Equal(t, "", test.Location)
}

func TestParseLine_DumpFailures(t *testing.T) {
	parser := NewOutputParser()

	line := `PLUR_JSON:{"type":"dump_failures","formatted_output":"\n  ‽) Failure:\nFooTest#test_fail [test/foo_test.rb:10]:\nExpected: 1\n  Actual: 2\n"}`
	notifications, consumed := parser.ParseLine(line)

	assert.True(t, consumed)
	require.Len(t, notifications, 1)
	failures := notifications[0].(types.FormattedFailuresNotification)
	assert.Contains(t, failures.Content, "‽) Failure:")
	assert.Contains(t, failures.Content, "FooTest#test_fail [test/foo_test.rb:10]:")
}

func TestParseLine_Summary(t *testing.T) {
	parser := NewOutputParser()

	line := `PLUR_JSON:{"type":"suite_finished","test_count":9,"assertion_count":8,"failure_count":2,"error_count":1,"pending_count":1}`
	notifications, consumed := parser.ParseLine(line)

	assert.True(t, consumed)
	require.Len(t, notifications, 1)
	suite := notifications[0].(types.SuiteNotification)
	assert.Equal(t, types.SuiteFinished, suite.Event)
	assert.Equal(t, 9, suite.TestCount)
	assert.Equal(t, 8, suite.AssertionCount)
	assert.Equal(t, 2, suite.FailureCount)
	assert.Equal(t, 1, suite.ErrorCount)
	assert.Equal(t, 1, suite.PendingCount)
}

func TestParseLine_BareLinesAreTestOutput(t *testing.T) {
	// With minitest's reporters replaced, anything without the row prefix is
	// test-written stdout - including lines that look like progress or
	// minitest's own prose.
	parser := NewOutputParser()

	for _, line := range []string{
		"OUT_MID_RUN",
		"...",
		"..in test_foo",
		"# Running:",
		"3 runs, 3 assertions, 0 failures, 0 errors, 0 skips",
		"  1) Failure:",
	} {
		notifications, consumed := parser.ParseLine(line)
		assert.False(t, consumed, "line %q must not be consumed", line)
		assert.Empty(t, notifications, "line %q must produce no notifications", line)
	}
}

func TestParseLine_MalformedRowFallsBackToRawOutput(t *testing.T) {
	// A test printing the prefix itself must not crash the parser or forge
	// notifications; the line falls through as ordinary test output.
	parser := NewOutputParser()

	notifications, consumed := parser.ParseLine("PLUR_JSON:not json at all")

	assert.False(t, consumed)
	assert.Empty(t, notifications)
}

func TestParseLine_UnknownRowTypeIsConsumed(t *testing.T) {
	// Forward compatibility: rows from a newer reporter are plur-internal,
	// never test output.
	parser := NewOutputParser()

	notifications, consumed := parser.ParseLine(`PLUR_JSON:{"type":"telemetry"}`)

	assert.True(t, consumed)
	assert.Empty(t, notifications)
}

func TestFormatSummary(t *testing.T) {
	parser := NewOutputParser()

	t.Run("pluralizes counts", func(t *testing.T) {
		suite := &types.SuiteNotification{AssertionCount: 11, ErrorCount: 1}
		summary := parser.FormatSummary(suite, 12, 2, 1, 1.5, 0.2)

		assert.Contains(t, summary, "12 runs, 11 assertions, 2 failures, 1 error, 1 skip")
		assert.Contains(t, summary, "Finished in 1.5 seconds")
	})

	t.Run("singular counts", func(t *testing.T) {
		suite := &types.SuiteNotification{AssertionCount: 1}
		summary := parser.FormatSummary(suite, 1, 1, 0, 0.5, 0.1)

		assert.Contains(t, summary, "1 run, 1 assertion, 1 failure, 0 errors, 0 skips")
	})
}
