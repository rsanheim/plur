package minitest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/format"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

var (
	summaryRegex           = regexp.MustCompile(`(\d+) (?:runs?|tests?), (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?`)
	failureHeaderLineRegex = regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):`)
)

// outputParser parses minitest text output into notifications
type outputParser struct {
	collectingFailures bool                         // Whether we're collecting failure text
	failureBuffer      strings.Builder              // Accumulates failure section
	failures           []types.TestCaseNotification // Extracted failures for runtime tracking
	progressCount      int                          // Track progress index
	startTime          time.Time                    // When the parser was created (for load time calculation)
}

// NewOutputParser creates a new minitest output parser with proper initialization
func NewOutputParser() types.TestOutputParser {
	return &outputParser{
		startTime: time.Now(),
	}
}

// CurrentFile returns empty string for minitest (no structured file tracking)
func (p *outputParser) CurrentFile() string {
	return ""
}

// Converts a TestNotification to a progress type (just a string for now) for streaming to output
func (p *outputParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	if notification.GetEvent() != types.Progress {
		return "", false
	}
	event := notification.(types.ProgressEvent)
	switch event.Character {
	case ".":
		return "dot", true
	case "F":
		return "failure", true
	case "E":
		return "error", true
	case "S":
		return "pending", true
	}
	return "", true
}

// FormatSummary formats a test summary in minitest style
func (p *outputParser) FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string {
	// Minitest doesn't typically show load time in the summary
	// Format: "X runs, Y assertions, Z failures, W errors, V skips"

	// For now, we can't distinguish between failures and errors from the summary data
	// In minitest, totalFailures includes both failures and errors
	// We'll need to track these separately in the future

	runText := "1 run"
	if totalExamples != 1 {
		runText = fmt.Sprintf("%d runs", totalExamples)
	}

	// TODO: Track assertions count properly
	// For now, assume at least one assertion per test
	assertionText := "1 assertion"
	if totalExamples != 1 {
		assertionText = fmt.Sprintf("%d assertions", totalExamples)
	}

	failureText := "0 failures"
	if totalFailures == 1 {
		failureText = "1 failure"
	} else if totalFailures > 1 {
		failureText = fmt.Sprintf("%d failures", totalFailures)
	}

	// TODO: Track errors separately from failures
	errorText := "0 errors"

	skipText := "0 skips"
	if totalPending == 1 {
		skipText = "1 skip"
	} else if totalPending > 1 {
		skipText = fmt.Sprintf("%d skips", totalPending)
	}

	summary := fmt.Sprintf("\nFinished in %s.\n", format.FormatDuration(wallTime))
	summary += fmt.Sprintf("%s, %s, %s, %s, %s", runText, assertionText, failureText, errorText, skipText)

	return summary
}

// ParseLine parses a single line of minitest output
func (p *outputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	logger.Logger.Debug("[ParseLine]", "line", line)

	// Emit suite started on "# Running:" with load time
	if strings.HasPrefix(line, "# Running:") {
		loadTime := time.Duration(0)
		if !p.startTime.IsZero() {
			loadTime = time.Since(p.startTime)
		}
		return []types.TestNotification{types.SuiteNotification{
			Event:    types.SuiteStarted,
			LoadTime: loadTime,
		}}, false
	}

	// Parse progress indicators (., F, E, S)
	if containsProgressChars(line) {
		return p.parseProgressLine(line), false
	}

	// Start collecting failures on first failure header
	if !p.collectingFailures && isFailureHeaderLine(line) {
		p.collectingFailures = true
		p.failureBuffer.WriteString(line + "\n")
		return nil, false // Preserve the line in output
	}

	// Continue collecting failure text until summary
	if p.collectingFailures {
		if isSummaryLine(line) {
			// Extract failures for runtime tracking
			p.failures = ExtractFailures(p.failureBuffer.String())
			return p.parseSummaryLine(line), false
		}
		p.failureBuffer.WriteString(line + "\n")
		return nil, false // Preserve the line in output
	}

	// Check for summary without failures
	if isSummaryLine(line) {
		return p.parseSummaryLine(line), false
	}

	return nil, false // Minitest output is always preserved
}

func (p *outputParser) parseSummaryLine(line string) []types.TestNotification {
	// Check for summary line
	if match := summaryRegex.FindStringSubmatch(line); match != nil {
		runs, _ := strconv.Atoi(match[1])
		failures, _ := strconv.Atoi(match[3])
		errors, _ := strconv.Atoi(match[4])
		skips, _ := strconv.Atoi(match[5])

		notifications := []types.TestNotification{}

		// Emit individual TestCaseNotifications for runtime tracking
		for _, failure := range p.failures {
			notifications = append(notifications, failure)
		}

		// Create the suite finished notification
		finishNotification := types.SuiteNotification{
			Event:        types.SuiteFinished,
			TestCount:    runs,
			FailureCount: failures + errors,
			PendingCount: skips,
		}
		notifications = append(notifications, finishNotification)

		return notifications
	}
	return nil // Return nil if not a summary line
}

func (p *outputParser) parseProgressLine(line string) []types.TestNotification {
	notifications := []types.TestNotification{}

	// Check for progress indicators and create progress events
	for _, char := range line {
		switch char {
		case '.', 'F', 'E', 'S':
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: string(char),
				Index:     p.progressCount,
			})
			p.progressCount++
			logger.Logger.Debug("Progress", "char", string(char), "index", p.progressCount-1)
		default:
			// Ignore other characters
			continue
		}
	}

	return notifications
}

// Helper methods for line classification
func containsProgressChars(line string) bool {
	// Progress lines are typically just progress indicators without other text
	// Avoid matching lines that happen to contain these characters in other contexts
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Check if line consists only of progress characters
	for _, char := range trimmed {
		switch char {
		case '.', 'F', 'E', 'S':
			continue
		default:
			return false
		}
	}
	return true
}

func isFailureHeaderLine(line string) bool {
	return failureHeaderLineRegex.MatchString(line)
}

func isSummaryLine(line string) bool {
	return summaryRegex.MatchString(line)
}

// FormatFailuresList returns empty string since minitest doesn't use failure lists
func (p *outputParser) FormatFailuresList(failures []types.TestCaseNotification) string {
	// Minitest doesn't typically show a re-run command list like RSpec
	return ""
}

// ColorizeSummary applies color to a summary based on success/failure state
func (p *outputParser) ColorizeSummary(summary string, hasFailures bool) string {
	if hasFailures {
		return fmt.Sprintf("\033[31m%s\033[0m", summary)
	}
	return fmt.Sprintf("\033[32m%s\033[0m", summary)
}
