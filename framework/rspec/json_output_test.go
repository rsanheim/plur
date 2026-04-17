package rspec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFailure(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		failure  FailureDetail
		expected []string // Parts that should be in the output
	}{
		{
			name:  "formats RSpec expectation failure",
			index: 1,
			failure: FailureDetail{
				Description: "Calculator#add returns the sum of two numbers",
				FilePath:    "spec/calculator_spec.rb",
				LineNumber:  10,
				ErrorClass:  "RSpec::Expectations::ExpectationNotMetError",
				Message:     "expected: 5\n     got: 4\n\n(compared using ==)",
				Backtrace:   []string{"./spec/calculator_spec.rb:12:in `block (3 levels) in <top (required)>'"},
			},
			expected: []string{
				"  1) Calculator#add returns the sum of two numbers",
				"     Failure/Error:",
				"       expected: 5",
				"            got: 4",
				"       (compared using ==)",
				"     # ./spec/calculator_spec.rb:12:in `block (3 levels) in <top (required)>'",
			},
		},
		{
			name:  "formats standard error",
			index: 2,
			failure: FailureDetail{
				Description: "User validation requires email",
				FilePath:    "spec/models/user_spec.rb",
				LineNumber:  25,
				ErrorClass:  "NoMethodError",
				Message:     "undefined method `email' for nil:NilClass",
				Backtrace:   []string{"./spec/models/user_spec.rb:27:in `block (3 levels) in <top (required)>'"},
			},
			expected: []string{
				"  2) User validation requires email",
				"     Failure/Error:",
				"     NoMethodError:",
				"       undefined method `email' for nil:NilClass",
				"     # ./spec/models/user_spec.rb:27:in `block (3 levels) in <top (required)>'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFailure(tt.index, tt.failure)

			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "FormatFailure() output missing expected string")
			}

			// Check specific indentation patterns
			lines := strings.Split(result, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}

				// Check failure header (e.g., "  1) Description")
				if strings.Contains(line, ") ") && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "     ") {
					// Should have exactly 2 spaces
					assert.False(t, strings.HasPrefix(line, "   "), "Failure header has incorrect indentation: %q", line)
				} else if strings.Contains(line, "Failure/Error:") {
					// Should have exactly 5 spaces
					assert.True(t, strings.HasPrefix(line, "     "), "Failure/Error line should start with 5 spaces: %q", line)
					assert.False(t, strings.HasPrefix(line, "      "), "Failure/Error line should not have 6+ spaces: %q", line)
				} else if strings.HasPrefix(line, "       ") && (strings.Contains(line, "expected:") || strings.Contains(line, "got:")) {
					// These are correctly indented with 7 spaces
					continue
				}
			}
		})
	}
}

func TestFormatFailedExamples(t *testing.T) {
	failures := []FailureDetail{
		{
			Description: "Calculator#add returns the sum",
			FilePath:    "spec/calculator_spec.rb",
			LineNumber:  10,
		},
		{
			Description: "Calculator#subtract returns the difference",
			FilePath:    "spec/calculator_spec.rb",
			LineNumber:  20,
		},
	}

	expected := `rspec spec/calculator_spec.rb:10 # Calculator#add returns the sum
rspec spec/calculator_spec.rb:20 # Calculator#subtract returns the difference
`

	result := FormatFailedExamples(failures)
	assert.Equal(t, expected, result, "FormatFailedExamples()")
}

func TestExtractFailures(t *testing.T) {
	examples := []Example{
		{
			ID:              "1",
			Description:     "returns the sum",
			FullDescription: "Calculator#add returns the sum",
			Status:          "passed",
			FilePath:        "spec/calculator_spec.rb",
			LineNumber:      10,
		},
		{
			ID:              "2",
			Description:     "returns the difference",
			FullDescription: "Calculator#subtract returns the difference",
			Status:          "failed",
			FilePath:        "spec/calculator_spec.rb",
			LineNumber:      20,
			Exception: &Exception{
				Class:     "RSpec::Expectations::ExpectationNotMetError",
				Message:   "expected: 5\n     got: 3",
				Backtrace: []string{"./spec/calculator_spec.rb:22:in `block'"},
			},
		},
		{
			ID:              "3",
			Description:     "pending example",
			FullDescription: "Calculator#multiply pending example",
			Status:          "pending",
			FilePath:        "spec/calculator_spec.rb",
			LineNumber:      30,
		},
	}

	failures := ExtractFailures(examples)

	assert.Len(t, failures, 1, "ExtractFailures() should return 1 failure")

	failure := failures[0]
	assert.Equal(t, "Calculator#subtract returns the difference", failure.Description, "Wrong description")
	assert.Equal(t, "spec/calculator_spec.rb", failure.FilePath, "Wrong file path")
	assert.Equal(t, 20, failure.LineNumber, "Wrong line number")
	assert.Equal(t, "RSpec::Expectations::ExpectationNotMetError", failure.ErrorClass, "Wrong error class")
}
