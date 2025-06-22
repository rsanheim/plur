package minitest

import (
	"fmt"
	"regexp"
	"strings"
)

// FailureDetail represents a single test failure
type FailureDetail struct {
	Description string
	LineNumber  int
	Message     string
	Backtrace   []string
}

// ExtractFailures parses minitest output and extracts failure details
func ExtractFailures(output string) []FailureDetail {
	var failures []FailureDetail
	lines := strings.Split(output, "\n")
	
	// Look for failure patterns in minitest output
	// Minitest failures typically look like:
	//   1) Failure:
	// TestClassName#test_method_name [file_path:line_number]:
	// Expected: expected_value
	// Actual: actual_value
	
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