package types

// TestOutputParser is the interface for parsing test framework output
type TestOutputParser interface {
	// ParseLine parses a single line of output and returns notifications
	// The bool indicates if the line was consumed (should not be included in raw output)
	ParseLine(line string) ([]TestNotification, bool)
	// NotificationToProgress converts a notification to a progress type
	// If the notification is not a progress notification, the second return value is false
	NotificationToProgress(notification TestNotification) (string, bool)
	// FormatSummary formats a test summary in the framework-specific style
	FormatSummary(suite *SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string
	// FormatFailuresList formats a list of failures with file:line references for re-running
	FormatFailuresList(failures []TestCaseNotification) string
	// ColorizeSummary applies color to a summary based on success/failure state
	ColorizeSummary(summary string, hasFailures bool) string
	// CurrentFile returns the current file being tested (for trace-output)
	CurrentFile() string
}
