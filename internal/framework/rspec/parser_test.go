package rspec

import (
	"testing"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputParser_ParseLine_DumpSummaryIncludesErrorCount(t *testing.T) {
	parser := &outputParser{}
	line := `PLUR_JSON:{"type":"dump_summary","example_count":2,"failure_count":0,"pending_count":0,"errors_outside_of_examples_count":1,"duration":1.0}`

	notifications, consumed := parser.ParseLine(line)

	require.True(t, consumed)
	require.Len(t, notifications, 1)

	suite, ok := notifications[0].(types.SuiteNotification)
	require.True(t, ok)
	assert.Equal(t, types.SuiteFinished, suite.Event)
	assert.Equal(t, 2, suite.TestCount)
	assert.Equal(t, 0, suite.FailureCount)
	assert.Equal(t, 0, suite.PendingCount)
	assert.Equal(t, 1, suite.ErrorCount)
}

func TestOutputParser_FormatSummaryIncludesErrorCount(t *testing.T) {
	parser := &outputParser{}

	t.Run("singular error", func(t *testing.T) {
		summary := parser.FormatSummary(&types.SuiteNotification{ErrorCount: 1}, 2, 0, 0, 1.2, 0.1)
		assert.Contains(t, summary, "2 examples, 0 failures")
		assert.Contains(t, summary, "1 error occurred outside of examples")
	})

	t.Run("plural errors", func(t *testing.T) {
		summary := parser.FormatSummary(&types.SuiteNotification{ErrorCount: 2}, 3, 1, 0, 1.2, 0.1)
		assert.Contains(t, summary, "3 examples, 1 failure")
		assert.Contains(t, summary, "2 errors occurred outside of examples")
	})
}

func TestOutputParser_ParseLine_GroupStartedTracksCurrentFileWithoutNotification(t *testing.T) {
	parser := &outputParser{}
	line := `PLUR_JSON:{"type":"group_started","group":{"description":"Calculator","file_path":"./spec/calculator_spec.rb","line_number":3}}`

	notifications, consumed := parser.ParseLine(line)

	require.True(t, consumed)
	assert.Empty(t, notifications)
	assert.Equal(t, "spec/calculator_spec.rb", parser.CurrentFile())
}

func TestOutputParser_ParseLine_ExamplePassedCarriesIdentityFields(t *testing.T) {
	parser := &outputParser{}
	line := `PLUR_JSON:{"type":"example_passed","example":{` +
		`"id":"./spec/calc_spec.rb[1:1]",` +
		`"description":"adds","full_description":"Calculator adds",` +
		`"location":"./spec/calc_spec.rb:10","file_path":"./spec/calc_spec.rb",` +
		`"absolute_file_path":"/repo/spec/calc_spec.rb","line_number":10,` +
		`"location_rerun_argument":"./spec/calc_spec.rb:10","scoped_id":"1:1",` +
		`"status":"passed","run_time":0.25}}`

	notifications, consumed := parser.ParseLine(line)

	require.True(t, consumed)
	require.Len(t, notifications, 1)

	tc, ok := notifications[0].(types.TestCaseNotification)
	require.True(t, ok)
	assert.Equal(t, "./spec/calc_spec.rb[1:1]", tc.TestID, "TestID should be RSpec example.id when present")
	assert.Equal(t, "spec/calc_spec.rb", tc.FilePath, "leading ./ must be stripped")
	assert.Equal(t, "/repo/spec/calc_spec.rb", tc.AbsoluteFilePath)
	assert.Equal(t, "./spec/calc_spec.rb:10", tc.LocationRerunArgument)
	assert.Equal(t, "1:1", tc.ScopedID)
	assert.Equal(t, 10, tc.LineNumber)
	assert.Equal(t, types.TestPassed, tc.Event)
}

func TestOutputParser_ParseLine_ExamplePassedFallsBackToLocationWhenNoID(t *testing.T) {
	parser := &outputParser{}
	line := `PLUR_JSON:{"type":"example_passed","example":{` +
		`"description":"legacy","full_description":"legacy",` +
		`"location":"./spec/legacy_spec.rb:7","file_path":"./spec/legacy_spec.rb",` +
		`"line_number":7,"status":"passed","run_time":0.1}}`

	notifications, consumed := parser.ParseLine(line)

	require.True(t, consumed)
	require.Len(t, notifications, 1)
	tc, ok := notifications[0].(types.TestCaseNotification)
	require.True(t, ok)
	assert.Equal(t, "./spec/legacy_spec.rb:7", tc.TestID, "falls back to location when id is empty")
	assert.Equal(t, "spec/legacy_spec.rb", tc.FilePath)
}

func TestOutputParser_FormatFailuresList(t *testing.T) {
	parser := &outputParser{}
	failures := []types.TestCaseNotification{
		{
			FullDescription: "Calculator#add returns the sum",
			FilePath:        "spec/calculator_spec.rb",
			LineNumber:      10,
		},
		{
			FullDescription: "Calculator#subtract returns the difference",
			FilePath:        "spec/calculator_spec.rb",
			LineNumber:      20,
		},
	}

	expected := `rspec spec/calculator_spec.rb:10 # Calculator#add returns the sum
rspec spec/calculator_spec.rb:20 # Calculator#subtract returns the difference
`

	assert.Equal(t, expected, parser.FormatFailuresList(failures))
}
