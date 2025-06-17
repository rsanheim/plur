package rspec

import (
	"testing"
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
			if err != nil {
				t.Fatalf("ParseStreamingMessage() error = %v", err)
			}
			if msg == nil {
				t.Fatal("ParseStreamingMessage() returned nil message")
			}
			if msg.Type != tt.wantType {
				t.Errorf("ParseStreamingMessage() Type = %v, want %v", msg.Type, tt.wantType)
			}

			// Check load time for load_summary message
			if msg.Type == "load_summary" {
				if msg.Summary == nil {
					t.Fatal("ParseStreamingMessage() Summary is nil for load_summary message")
				}
				if msg.Summary.FileLoadTime != tt.wantLoadTime {
					t.Errorf("ParseStreamingMessage() FileLoadTime = %v, want %v", msg.Summary.FileLoadTime, tt.wantLoadTime)
				}
			}
		})
	}
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
			if err != nil {
				t.Fatalf("ParseStreamingMessage() error = %v", err)
			}
			if msg == nil {
				t.Fatal("ParseStreamingMessage() returned nil message")
			}
			if msg.Type != tt.want.Type {
				t.Errorf("ParseStreamingMessage() Type = %v, want %v", msg.Type, tt.want.Type)
			}
		})
	}
}
