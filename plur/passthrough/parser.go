package passthrough

import (
	"github.com/rsanheim/plur/types"
)

// Parser is a passthrough parser that doesn't parse output
// It's used for custom test frameworks that don't have specific parsers
// Output is forwarded directly without parsing or formatting
type Parser struct{}

// NewOutputParser creates a new passthrough parser
func NewOutputParser() types.TestOutputParser {
	return &Parser{}
}

// ParseLine returns no notifications and indicates line was not consumed
// This causes the line to be printed directly to output
func (p *Parser) ParseLine(line string) ([]types.TestNotification, bool) {
	return nil, false
}

// NotificationToProgress always returns false since we don't parse notifications
func (p *Parser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	return "", false
}

// FormatSummary returns empty string since we don't format summaries
func (p *Parser) FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string {
	return ""
}

// FormatFailuresList returns empty string since we don't format failure lists
func (p *Parser) FormatFailuresList(failures []types.TestCaseNotification) string {
	return ""
}

// ColorizeSummary returns the summary unchanged since we don't colorize
func (p *Parser) ColorizeSummary(summary string, hasFailures bool) string {
	return summary
}
