package types

// TestOutputParser is the interface for parsing test framework output
type TestOutputParser interface {
	// ParseLine parses a single line of output and returns notifications
	// The bool indicates if the line was consumed (should not be included in raw output)
	ParseLine(line string) ([]TestNotification, bool)
}
