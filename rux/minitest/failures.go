package minitest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rsanheim/rux/types"
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

/*
ExtractFailures parses minitest output and extracts failure details
For example, given output seen in ./failures-example.txt, we should get three FailureDetails, two 'failures' and one 'error'
*/
func ExtractFailures(output string) []FailureDetail {
	var failures []FailureDetail
	lines := strings.Split(output, "\n")

	failureHeaderRegex := regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):$`)
	testInfoRegex := regexp.MustCompile(`^(.+?)\s+\[(.+?):(\d+)\]:$`)

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check if this is a failure header
		if failureHeaderRegex.MatchString(line) {
			i++
			if i >= len(lines) {
				break
			}

			// Next line should have test info
			testLine := lines[i]
			matches := testInfoRegex.FindStringSubmatch(testLine)

			if len(matches) > 3 {
				failure := FailureDetail{
					Description: matches[1],
					LineNumber:  parseInt(matches[3]),
				}

				// Collect the error message and backtrace
				var messageLines []string
				i++

				// Read until we hit another failure or end of output
				for i < len(lines) {
					if lines[i] == "" || failureHeaderRegex.MatchString(lines[i]) {
						break
					}
					messageLines = append(messageLines, lines[i])
					i++
				}

				if len(messageLines) > 0 {
					failure.Message = strings.Join(messageLines, "\n")
				}

				failures = append(failures, failure)
				continue
			}
		}
		i++
	}

	return failures
}

// parseInt safely parses an integer, returning 0 on error
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
