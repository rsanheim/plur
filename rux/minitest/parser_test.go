package minitest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected *OutputSummary
	}{
		{
			name:   "basic success output",
			output: "10 tests, 20 assertions, 0 failures, 0 errors",
			expected: &OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     0,
				Skips:      0,
			},
		},
		{
			name:   "output with failures",
			output: "15 tests, 25 assertions, 2 failures, 1 error",
			expected: &OutputSummary{
				Tests:      15,
				Assertions: 25,
				Failures:   2,
				Errors:     1,
				Skips:      0,
			},
		},
		{
			name:   "output with skips",
			output: "5 tests, 10 assertions, 0 failures, 0 errors, 3 skips",
			expected: &OutputSummary{
				Tests:      5,
				Assertions: 10,
				Failures:   0,
				Errors:     0,
				Skips:      3,
			},
		},
		{
			name:   "singular forms",
			output: "1 test, 1 assertion, 1 failure, 1 error, 1 skip",
			expected: &OutputSummary{
				Tests:      1,
				Assertions: 1,
				Failures:   1,
				Errors:     1,
				Skips:      1,
			},
		},
		{
			name:   "output with ANSI color codes",
			output: "10 tests, 20 assertions, 0 \x1b[31mfailures, 0 errors",
			expected: &OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     0,
				Skips:      0,
			},
		},
		{
			name:   "output with surrounding text",
			output: `Loaded suite test
Started
..............
Finished in 0.145069 seconds.

10 tests, 20 assertions, 0 failures, 0 errors`,
			expected: &OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     0,
				Skips:      0,
			},
		},
		{
			name:     "no summary line",
			output:   "Running tests...",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseOutput(tt.output)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		summary  OutputSummary
		expected bool
	}{
		{
			name: "all passing",
			summary: OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     0,
			},
			expected: true,
		},
		{
			name: "with failures",
			summary: OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   1,
				Errors:     0,
			},
			expected: false,
		},
		{
			name: "with errors",
			summary: OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     1,
			},
			expected: false,
		},
		{
			name: "with skips only",
			summary: OutputSummary{
				Tests:      10,
				Assertions: 20,
				Failures:   0,
				Errors:     0,
				Skips:      2,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.summary.IsSuccessful()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFailureMessages(t *testing.T) {
	output := `Run options: --seed 1234

# Running:

F.E

Finished in 0.001234s, 2430.1337 runs/s, 2430.1337 assertions/s.

  1) Failure:
UserTest#test_name_validation [test/models/user_test.rb:10]:
Expected: "John"
  Actual: "Jane"

  2) Error:
UserTest#test_email_format [test/models/user_test.rb:15]:
NoMethodError: undefined method 'email' for nil:NilClass

3 tests, 3 assertions, 1 failure, 1 error, 0 skips`

	failures := ExtractFailureMessages(output)
	
	assert.Equal(t, 2, len(failures))
	assert.Contains(t, failures[0], "1) Failure:")
	assert.Contains(t, failures[0], "UserTest#test_name_validation")
	assert.Contains(t, failures[0], "Expected: \"John\"")
	
	assert.Contains(t, failures[1], "2) Error:")
	assert.Contains(t, failures[1], "UserTest#test_email_format")
	assert.Contains(t, failures[1], "NoMethodError")
}