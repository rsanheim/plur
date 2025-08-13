package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration float64
		expected string
	}{
		// Sub-second durations (5 decimal precision, trailing zeros removed)
		{
			name:     "very small duration",
			duration: 0.001,
			expected: "0.001 seconds",
		},
		{
			name:     "sub-second with trailing zeros",
			duration: 0.10000,
			expected: "0.1 seconds",
		},
		{
			name:     "sub-second with multiple decimals",
			duration: 0.123456,
			expected: "0.12346 seconds",
		},
		{
			name:     "near one second",
			duration: 0.99999,
			expected: "0.99999 seconds",
		},

		// 1-119 seconds (2 decimal precision, trailing zeros removed)
		{
			name:     "exactly one second",
			duration: 1.0,
			expected: "1 second",
		},
		{
			name:     "one second singular",
			duration: 1.00001,
			expected: "1 second",
		},
		{
			name:     "multiple seconds",
			duration: 2.5,
			expected: "2.5 seconds",
		},
		{
			name:     "issue example duration",
			duration: 6.84375,
			expected: "6.84 seconds",
		},
		{
			name:     "round up case",
			duration: 59.999,
			expected: "60 seconds",
		},
		{
			name:     "exactly 60 seconds",
			duration: 60.0,
			expected: "1 minute 0 seconds",
		},

		// Over 60 seconds (minute format)
		{
			name:     "just over one minute",
			duration: 60.001,
			expected: "1 minute 0 seconds",
		},
		{
			name:     "one minute with seconds",
			duration: 75.5,
			expected: "1 minute 15.5 seconds",
		},
		{
			name:     "near two minutes",
			duration: 119.999,
			expected: "2 minutes 0 seconds",
		},

		// 120-299 seconds (1 decimal precision)
		{
			name:     "exactly two minutes",
			duration: 120.0,
			expected: "2 minutes 0 seconds",
		},
		{
			name:     "three minutes with decimal",
			duration: 180.567,
			expected: "3 minutes 0.6 seconds",
		},
		{
			name:     "four minutes",
			duration: 240.123,
			expected: "4 minutes 0.1 seconds",
		},
		{
			name:     "near five minutes",
			duration: 299.999,
			expected: "5 minutes 0 seconds",
		},

		// >= 300 seconds (no decimal places)
		{
			name:     "exactly five minutes",
			duration: 300.0,
			expected: "5 minutes 0 seconds",
		},
		{
			name:     "five minutes with decimals ignored",
			duration: 300.789,
			expected: "5 minutes 1 second",
		},
		{
			name:     "one hour",
			duration: 3600.123,
			expected: "60 minutes 0 seconds",
		},
		{
			name:     "two hours",
			duration: 7200.999,
			expected: "120 minutes 1 second",
		},

		// Edge cases
		{
			name:     "negative duration",
			duration: -1.0,
			expected: "0 seconds",
		},
		{
			name:     "zero duration",
			duration: 0.0,
			expected: "0 seconds",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDuration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStripTrailingZeros(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"1.00", "1"},
		{"1.50", "1.5"},
		{"1.0001", "1.0001"},
		{"10", "10"},
		{"0.10000", "0.1"},
		{"123.456000", "123.456"},
		{"0.0", "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := stripTrailingZeros(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPluralize(t *testing.T) {
	testCases := []struct {
		count    interface{}
		word     string
		expected string
	}{
		{1, "second", "1 second"},
		{2, "second", "2 seconds"},
		{0, "second", "0 seconds"},
		{"1", "minute", "1 minute"},
		{"2", "minute", "2 minutes"},
		{"1.0", "second", "1.0 second"},
		{"1.5", "second", "1.5 seconds"},
		{1, "process", "1 process"},
		{2, "process", "2 processes"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := pluralize(tc.count, tc.word)
			assert.Equal(t, tc.expected, result)
		})
	}
}
