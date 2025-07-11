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

// ExtractFailures parses minitest output and extracts failure details as test notifications
func ExtractFailures(output string) []types.TestCaseNotification {
	var notifications []types.TestCaseNotification

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

		notification := types.TestCaseNotification{
			Event:           types.TestFailed,
			TestID:          testMatches[1],
			Description:     testMatches[1],
			FullDescription: testMatches[1],
			Status:          "failed",
		}

		// Set location fields if they exist
		if len(testMatches) > 3 && testMatches[2] != "" {
			notification.Location = testMatches[2] + ":" + testMatches[3]
			notification.FilePath = testMatches[2]
			notification.LineNumber = parseInt(testMatches[3])
		}

		// Process remaining lines
		if len(lines) > 2 {
			var message string
			var backtrace []string

			if failureType == "error" {
				// First line after header is the error message
				message = lines[2]

				// Extract error class if present (e.g., "ArgumentError: comparison of Integer with nil failed")
				// Handle cases like "ActiveRecord::StatementInvalid: PG::ConnectionBad: connection is closed"
				errorClass := "StandardError"
				parts := strings.SplitN(message, ": ", 2)
				if len(parts) >= 2 && parts[0] != "" {
					errorClass = strings.TrimSpace(parts[0])
				}

				// Rest are backtrace
				if len(lines) > 3 {
					for _, line := range lines[3:] {
						backtrace = append(backtrace, strings.TrimSpace(line))
					}

					// Extract location from last test file in backtrace
					location, filePath, lineNumber := extractLocationFromBacktrace(backtrace)
					if location != "" {
						notification.Location = location
						notification.FilePath = filePath
						notification.LineNumber = lineNumber
					}
				}

				notification.Exception = &types.TestException{
					Class:     errorClass,
					Message:   message,
					Backtrace: backtrace,
				}
			} else {
				// For failures, all remaining lines are the message
				message = strings.Join(lines[2:], "\n")
				notification.Exception = &types.TestException{
					Class:     "Minitest::Assertion",
					Message:   message,
					Backtrace: []string{},
				}
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications
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
