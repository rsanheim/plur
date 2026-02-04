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
