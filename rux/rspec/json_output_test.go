package rspec

import (
	"strings"
	"testing"
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
				if !strings.Contains(result, expected) {
					t.Errorf("FormatFailure() output missing expected string:\n%q\nGot:\n%s", expected, result)
				}
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
					if strings.HasPrefix(line, "   ") {
						t.Errorf("Failure header has incorrect indentation: %q", line)
					}
				} else if strings.Contains(line, "Failure/Error:") {
					// Should have exactly 5 spaces
					if !strings.HasPrefix(line, "     ") || strings.HasPrefix(line, "      ") {
						t.Errorf("Failure/Error line has incorrect indentation: %q", line)
					}
				} else if strings.HasPrefix(line, "       ") && (strings.Contains(line, "expected:") || strings.Contains(line, "got:")) {
					// These are correctly indented with 7 spaces
					continue
				} else if strings.HasPrefix(line, "     #") {
					// These are correctly indented with 5 spaces
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
	if result != expected {
		t.Errorf("FormatFailedExamples() = %q, want %q", result, expected)
	}
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

	if len(failures) != 1 {
		t.Fatalf("ExtractFailures() returned %d failures, want 1", len(failures))
	}

	failure := failures[0]
	if failure.Description != "Calculator#subtract returns the difference" {
		t.Errorf("Wrong description: got %q", failure.Description)
	}
	if failure.FilePath != "spec/calculator_spec.rb" {
		t.Errorf("Wrong file path: got %q", failure.FilePath)
	}
	if failure.LineNumber != 20 {
		t.Errorf("Wrong line number: got %d", failure.LineNumber)
	}
	if failure.ErrorClass != "RSpec::Expectations::ExpectationNotMetError" {
		t.Errorf("Wrong error class: got %q", failure.ErrorClass)
	}
}
