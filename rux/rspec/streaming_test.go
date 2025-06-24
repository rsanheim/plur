package rspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStreamingMessage_LoadSummary(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantType     string
		wantLoadTime float64
	}{
		{
			name:         "load_summary message",
			input:        `RUX_JSON:{"type":"load_summary","summary":{"count":5,"load_time":0.456}}`,
			wantType:     "load_summary",
			wantLoadTime: 0.456,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseStreamingMessage(tt.input)
			require.NoError(t, err, "ParseStreamingMessage() should not return error")
			require.NotNil(t, msg, "ParseStreamingMessage() should not return nil message")
			assert.Equal(t, tt.wantType, msg.Type, "message type should match")

			// Check load time for load_summary message
			if msg.Type == "load_summary" {
				require.NotNil(t, msg.Summary, "Summary should not be nil for load_summary message")
				assert.Equal(t, tt.wantLoadTime, msg.Summary.FileLoadTime, "FileLoadTime should match")
			}
		})
	}
}

func TestParseStreamingMessage_NonJSONLine(t *testing.T) {
	// Test that non-JSON lines return nil without error
	msg, err := ParseStreamingMessage("This is not a JSON line")
	assert.NoError(t, err, "non-JSON lines should not return error")
	assert.Nil(t, msg, "non-JSON lines should return nil message")
}

func TestParseStreamingMessage_TurboTestsCompatibility(t *testing.T) {
	// Test other turbo_tests message formats
	tests := []struct {
		name  string
		input string
		want  StreamingMessage
	}{
		{
			name:  "group_started message",
			input: `RUX_JSON:{"type":"group_started","group":{"description":"MyClass"}}`,
			want: StreamingMessage{
				Type: "group_started",
			},
		},
		{
			name:  "group_finished message",
			input: `RUX_JSON:{"type":"group_finished","group":{"description":"MyClass"}}`,
			want: StreamingMessage{
				Type: "group_finished",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseStreamingMessage(tt.input)
			require.NoError(t, err, "ParseStreamingMessage() should not return error")
			require.NotNil(t, msg, "ParseStreamingMessage() should not return nil message")
			assert.Equal(t, tt.want.Type, msg.Type, "message type should match")
		})
	}
}
