package minitest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rsanheim/rux/types"
)

var (
	// Matches: "  1) Failure:" or "  2) Error:"
	failureHeaderRegex = regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):$`)
	// Matches: "TestClass#test_method [file.rb:42]:" or "TestClass#test_method:"
	testInfoRegex = regexp.MustCompile(`^(.+?)(?:\s+\[(.+?):(\d+)\])?:$`)
	// Matches: "lib/file.rb:15:in 'method_name'"
	backtraceRegex = regexp.MustCompile(`^(.+?):(\d+):in`)
	// Matches: "array_operations_test.rb" or "test_database.rb"
	testFileRegex = regexp.MustCompile(`(_test\.rb|test_.*\.rb)`)
)

// FailureDetail represents a single test failure from minitest output
type FailureDetail struct {
	Description string          // ArrayOperationsTest#test_average_precision_failure
	Location    string          // test/array_operations_test.rb:47
	FilePath    string          // test/array_operations_test.rb
	LineNumber  int             // 47
	Message     string          // Expected: 2.3333333333333335\n  Actual: 2.3333333333333335
	Backtrace   []string        // optional: backtrace for errors (not failures)
	State       types.TestState // either "failure" or "error"
}

// ExtractFailures parses minitest output and extracts failure details
func ExtractFailures(output string) []FailureDetail {
	var failures []FailureDetail

	// Split on blank lines to get individual failure blocks
	blocks := strings.Split(output, "\n\n")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) < 2 {
			continue
		}

		// Check if first line is a failure header
		headerMatches := failureHeaderRegex.FindStringSubmatch(lines[0])
		if headerMatches == nil {
			continue
		}

		failureType := strings.ToLower(headerMatches[1])

		// Parse test info from second line
		testMatches := testInfoRegex.FindStringSubmatch(lines[1])
		if len(testMatches) < 2 {
			continue
		}

		failure := FailureDetail{
			Description: testMatches[1],
			State:       types.TestState(failureType),
		}

		// Set location fields if they exist
		if len(testMatches) > 3 && testMatches[2] != "" {
			failure.Location = testMatches[2] + ":" + testMatches[3]
			failure.FilePath = testMatches[2]
			failure.LineNumber = parseInt(testMatches[3])
		}

		// Process remaining lines
		if len(lines) > 2 {
			if failureType == "error" {
				// First line after header is the error message
				failure.Message = lines[2]

				// Rest are backtrace
				if len(lines) > 3 {
					for _, line := range lines[3:] {
						failure.Backtrace = append(failure.Backtrace, strings.TrimSpace(line))
					}

					// Extract location from last test file in backtrace
					failure.Location, failure.FilePath, failure.LineNumber = extractLocationFromBacktrace(failure.Backtrace)
				}
			} else {
				// For failures, all remaining lines are the message
				failure.Message = strings.Join(lines[2:], "\n")
			}
		}

		failures = append(failures, failure)
	}

	return failures
}

// parseInt safely parses an integer, returning 0 on error
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// extractLocationFromBacktrace finds the last test file in the backtrace and extracts location info
func extractLocationFromBacktrace(backtrace []string) (location string, filePath string, lineNumber int) {
	// Iterate from the end to find the last test file
	for i := len(backtrace) - 1; i >= 0; i-- {
		line := backtrace[i]
		// Check for both _test.rb and test_*.rb patterns
		if testFileRegex.MatchString(line) {
			matches := backtraceRegex.FindStringSubmatch(line)
			if len(matches) > 2 {
				filePath = matches[1]
				lineNumber = parseInt(matches[2])
				location = filePath + ":" + matches[2]
				return
			}
		}
	}

	return "", "", 0
}
