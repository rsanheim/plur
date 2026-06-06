package rspec

import (
	"fmt"
	"strings"
)

// FailureDetail represents a formatted failure for display.
type FailureDetail struct {
	Description string
	FilePath    string
	LineNumber  int
	ErrorClass  string
	Message     string
	Backtrace   []string
}

// FormatFailedExamples formats the list of failed examples for the summary.
func FormatFailedExamples(failures []FailureDetail) string {
	var sb strings.Builder

	for _, failure := range failures {
		sb.WriteString(fmt.Sprintf("rspec %s:%d # %s\n",
			failure.FilePath,
			failure.LineNumber,
			failure.Description))
	}

	return sb.String()
}
