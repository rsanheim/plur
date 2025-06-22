package minitest

import (
	"regexp"
	"strconv"
	"strings"
)

// OutputSummary represents the parsed minitest output summary
type OutputSummary struct {
	Tests      int
	Assertions int
	Failures   int
	Errors     int
	Skips      int
}

var (
	// Pattern to match minitest summary line: "10 tests, 20 assertions, 0 failures, 0 errors, 0 skips"
	summaryPattern = regexp.MustCompile(`(\d+) tests?, (\d+) assertions?, (\d+) failures?, (\d+) errors?(?:, (\d+) skips?)?`)
	// Pattern to remove ANSI color codes
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// ParseOutput parses minitest output and extracts summary information
func ParseOutput(output string) (*OutputSummary, error) {
	// Strip ANSI color codes first
	cleanOutput := ansiPattern.ReplaceAllString(output, "")

	// Look for summary line
	matches := summaryPattern.FindStringSubmatch(cleanOutput)
	if matches == nil {
		return nil, nil // No summary found yet
	}

	summary := &OutputSummary{}

	// Parse the numbers
	if len(matches) > 1 {
		summary.Tests, _ = strconv.Atoi(matches[1])
	}
	if len(matches) > 2 {
		summary.Assertions, _ = strconv.Atoi(matches[2])
	}
	if len(matches) > 3 {
		summary.Failures, _ = strconv.Atoi(matches[3])
	}
	if len(matches) > 4 {
		summary.Errors, _ = strconv.Atoi(matches[4])
	}
	if len(matches) > 5 && matches[5] != "" {
		summary.Skips, _ = strconv.Atoi(matches[5])
	}

	return summary, nil
}

// IsSuccessful returns true if there are no failures or errors
func (s *OutputSummary) IsSuccessful() bool {
	return s.Failures == 0 && s.Errors == 0
}

// ExtractFailureMessages attempts to extract failure messages from output
// This is a simple implementation that looks for common failure patterns
func ExtractFailureMessages(output string) []string {
	// Strip ANSI codes
	cleanOutput := ansiPattern.ReplaceAllString(output, "")

	var failures []string
	lines := strings.Split(cleanOutput, "\n")

	// Look for lines that indicate failures
	// Minitest failure output typically includes:
	// 1) Failure:
	// TestName#test_method [file_path:line]:
	// Expected...
	inFailure := false
	var currentFailure strings.Builder

	for _, line := range lines {
		if strings.Contains(line, ") Failure:") || strings.Contains(line, ") Error:") {
			// Start of a new failure
			if inFailure && currentFailure.Len() > 0 {
				failures = append(failures, currentFailure.String())
			}
			inFailure = true
			currentFailure.Reset()
			currentFailure.WriteString(line)
			currentFailure.WriteString("\n")
		} else if inFailure {
			// Continue collecting failure details
			if line == "" || strings.HasPrefix(line, "Finished in") {
				// End of failure
				if currentFailure.Len() > 0 {
					failures = append(failures, currentFailure.String())
				}
				inFailure = false
				currentFailure.Reset()
			} else {
				currentFailure.WriteString(line)
				currentFailure.WriteString("\n")
			}
		}
	}

	// Don't forget the last failure if we're still in one
	if inFailure && currentFailure.Len() > 0 {
		failures = append(failures, currentFailure.String())
	}

	return failures
}
