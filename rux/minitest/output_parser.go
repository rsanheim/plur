package minitest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/types"
)

// ParsingState represents the current state of the parser
type ParsingState int

const (
	Started ParsingState = iota
	TestsRunning
	TestsComplete
	SummaryStarted
	SummaryComplete
)

// ProgressCounts tracks test progress indicators
type ProgressCounts struct {
	examples int // Total tests run
	passed   int
	failed   int
	errors   int
	pending  int
}

// OutputParser parses minitest text output into notifications
type OutputParser struct {
	state             ParsingState
	progress          ProgressCounts
	failureBuffer     strings.Builder
	currentFailure    *FailureInfo                 // Accumulating failure details
	collectedFailures []types.TestCaseNotification // All failures for formatting
}

// FailureInfo holds temporary failure details while parsing
type FailureInfo struct {
	testName   string
	fileName   string
	lineNumber int
	message    strings.Builder
}

// Converts a TestNotification to a progress type (just a string for now) for streaming to output
func (p *OutputParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
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
func (p *OutputParser) FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string {
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

	summary := fmt.Sprintf("\nFinished in %.6fs.\n", wallTime)
	summary += fmt.Sprintf("%s, %s, %s, %s, %s", runText, assertionText, failureText, errorText, skipText)

	return summary
}

// ParseLine parses a single line of minitest output
func (p *OutputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	logger.Logger.Debug("[ParseLine]", "line", line, "state", p.state)

	notifications := []types.TestNotification{}

	switch p.state {
	case Started:
		// Looking for test start
		if strings.HasPrefix(line, "# Running:") {
			p.state = TestsRunning
			notifications = append(notifications, types.SuiteNotification{
				Event: types.SuiteStarted,
			})
			logger.Logger.Debug("Transitioned to TestsRunning")
			return notifications, false
		}

	case TestsRunning:
		// Check if tests completed
		if strings.HasPrefix(line, "Finished in") {
			p.state = TestsComplete
			logger.Logger.Debug("Transitioned to TestsComplete")
			return notifications, false
		}
		// Parse progress indicators and create notifications
		if line != "" {
			progressNotifications := p.parseProgressLine(line)
			notifications = append(notifications, progressNotifications...)
		}

	case TestsComplete:
		// Skip empty lines
		if line == "" {
			return notifications, false
		}

		// Check for summary line
		if summaryNotifications := p.parseSummaryLine(line); summaryNotifications != nil {
			notifications = append(notifications, summaryNotifications...)
			p.state = SummaryComplete
			logger.Logger.Debug("Transitioned to SummaryComplete")
			return notifications, false
		}

		// Check if this is a failure header
		if p.parseFailureHeader(line) {
			p.state = SummaryStarted
			logger.Logger.Debug("Transitioned to SummaryStarted")
			return notifications, false
		}

	case SummaryStarted:
		// Handle failure detail parsing
		logger.Logger.Debug("In SummaryStarted state", "line", line)

		// Check if this is the test location line
		if p.parseFailureLocation(line) {
			return notifications, false
		}

		// Check if we've reached the end of this failure (empty line or next failure)
		if line == "" || regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):`).MatchString(line) {
			// Create a new TestCaseNotification with failure details
			if p.currentFailure != nil && p.currentFailure.testName != "" {
				logger.Logger.Debug("Creating failure notification", "testName", p.currentFailure.testName)
				notification := p.createFailureNotification()
				if notification != nil {
					notifications = append(notifications, notification)
					// Also collect it for formatted output
					if testCase, ok := notification.(types.TestCaseNotification); ok {
						p.collectedFailures = append(p.collectedFailures, testCase)
					}
				}
			}
			p.currentFailure = nil

			// If this was another failure header, parse it
			if line != "" && p.parseFailureHeader(line) {
				// Stay in SummaryStarted state
			} else if line == "" {
				// Might be transitioning to summary line
				p.state = TestsComplete
				logger.Logger.Debug("Transitioned back to TestsComplete")
			}
		} else {
			// Accumulate failure message
			if p.currentFailure != nil {
				if p.currentFailure.message.Len() > 0 {
					p.currentFailure.message.WriteString("\n")
				}
				p.currentFailure.message.WriteString(line)
			}
		}

	case SummaryComplete:
		// Parser is done, ignore remaining lines
		logger.Logger.Debug("Parser in SummaryComplete state, ignoring line")
	}

	return notifications, false // Minitest output is always preserved
}

func (p *OutputParser) parseSummaryLine(line string) []types.TestNotification {
	// Check for summary line
	if match := regexp.MustCompile(`(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?`).FindStringSubmatch(line); match != nil {
		runs, _ := strconv.Atoi(match[1])
		failures, _ := strconv.Atoi(match[3])
		errors, _ := strconv.Atoi(match[4])
		skips, _ := strconv.Atoi(match[5])

		notifications := []types.TestNotification{}

		// If we have failures, emit a FormattedFailuresNotification
		if len(p.collectedFailures) > 0 {
			formattedFailures := p.FormatFailures(p.collectedFailures)
			if formattedFailures != "" {
				notifications = append(notifications, types.FormattedFailuresNotification{
					Content: formattedFailures,
				})
			}
		}

		// Create the suite finished notification
		finishNotification := types.SuiteNotification{
			Event:        types.SuiteFinished,
			TestCount:    runs,
			FailureCount: failures + errors,
			PendingCount: skips,
		}
		notifications = append(notifications, finishNotification)

		// Also emit a formatted summary notification
		// Note: We don't have wallTime here, so the summary will be generated later in PrintResults
		// This is just for consistency with RSpec's approach

		return notifications
	}
	return nil // Return nil if not a summary line
}

func (p *OutputParser) parseProgressLine(line string) []types.TestNotification {
	notifications := []types.TestNotification{}

	// Check for progress indicators and create progress events
	for _, char := range line {
		index := p.progress.examples // 0-based index

		switch char {
		case '.':
			p.progress.passed++
			p.progress.examples++
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: ".",
				Index:     index,
			})
			logger.Logger.Debug("Progress: passed test", "index", index)

		case 'F':
			p.progress.failed++
			p.progress.examples++
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: "F",
				Index:     index,
			})
			logger.Logger.Debug("Progress: failed test", "index", index)

		case 'E':
			p.progress.errors++
			p.progress.examples++
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: "E",
				Index:     index,
			})
			logger.Logger.Debug("Progress: error test", "index", index)

		case 'S':
			p.progress.pending++
			p.progress.examples++
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: "S",
				Index:     index,
			})
			logger.Logger.Debug("Progress: pending test", "index", index)

		default:
			// Ignore other characters
			continue
		}
	}

	return notifications
}

// parseFailureHeader checks if a line is a failure header and extracts info
// Example: "  1) Failure:"
func (p *OutputParser) parseFailureHeader(line string) bool {
	// Check for failure header pattern: "  N) Failure:" or "  N) Error:"
	match := regexp.MustCompile(`^\s*(\d+)\)\s+(Failure|Error):\s*$`).FindStringSubmatch(line)
	if match != nil {
		failureType := match[2]

		logger.Logger.Debug("Found failure header", "line", line, "type", failureType)
		p.currentFailure = &FailureInfo{}
		return true
	}
	return false
}

// parseFailureLocation extracts test name and location from failure line
// Example: "MixedResultsTest#test_email_validation_mixed [test/mixed_results_test.rb:54]:"
func (p *OutputParser) parseFailureLocation(line string) bool {
	match := regexp.MustCompile(`^(\w+)#(\w+)\s+\[([^:]+):(\d+)\]:`).FindStringSubmatch(line)
	if match != nil && p.currentFailure != nil {
		p.currentFailure.testName = match[1] + "#" + match[2]
		p.currentFailure.fileName = match[3]
		p.currentFailure.lineNumber, _ = strconv.Atoi(match[4])
		logger.Logger.Debug("Parsed failure location",
			"testName", p.currentFailure.testName,
			"fileName", p.currentFailure.fileName,
			"lineNumber", p.currentFailure.lineNumber)
		return true
	}
	return false
}

// createFailureNotification creates a new test notification with failure details
func (p *OutputParser) createFailureNotification() types.TestNotification {
	if p.currentFailure == nil || p.currentFailure.testName == "" {
		return nil
	}

	// Use the actual test name as the TestID
	testID := p.currentFailure.testName

	// Create a new notification with failure details
	notification := types.TestCaseNotification{
		Event:           types.TestFailed,
		TestID:          testID,
		Description:     p.currentFailure.testName,
		FullDescription: p.currentFailure.testName,
		Location:        fmt.Sprintf("%s:%d", p.currentFailure.fileName, p.currentFailure.lineNumber),
		FilePath:        p.currentFailure.fileName,
		LineNumber:      p.currentFailure.lineNumber,
		Status:          "failed",
		Exception: &types.TestException{
			Class:     "Minitest::Assertion", // Minitest uses this for failures
			Message:   strings.TrimSpace(p.currentFailure.message.String()),
			Backtrace: []string{}, // TODO: Parse backtrace if needed
		},
	}

	logger.Logger.Debug("Created failure notification",
		"testID", notification.TestID,
		"testName", notification.Description,
		"location", notification.Location)

	return notification
}

// FormatFailures formats individual failure details in Minitest style
func (p *OutputParser) FormatFailures(failures []types.TestCaseNotification) string {
	if len(failures) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nFailures:\n")

	for i, failure := range failures {
		sb.WriteString(fmt.Sprintf("  %d) %s\n", i+1, failure.FullDescription))
		sb.WriteString("     Failure/Error: ")
		sb.WriteString("\n")

		// Error message - check if Exception exists
		if failure.Exception != nil {
			// Format the message with proper indentation
			lines := strings.Split(strings.TrimSpace(failure.Exception.Message), "\n")
			for _, line := range lines {
				if line != "" {
					sb.WriteString("       " + line + "\n")
				}
			}

			// Backtrace - Minitest shows more than one line typically
			// TODO: Capture full backtrace in parser
			if len(failure.Exception.Backtrace) > 0 {
				for _, trace := range failure.Exception.Backtrace {
					sb.WriteString(fmt.Sprintf("     # %s\n", trace))
				}
			}
		}

		if i < len(failures)-1 {
			sb.WriteString("\n") // Extra line between failures
		}
	}

	return sb.String()
}

// FormatFailuresList formats a list of failures with file:line references for re-running
func (p *OutputParser) FormatFailuresList(failures []types.TestCaseNotification) string {
	// Minitest doesn't typically show a re-run command list like RSpec
	return ""
}

// ColorizeSummary applies color to a summary based on success/failure state
func (p *OutputParser) ColorizeSummary(summary string, hasFailures bool) string {
	if hasFailures {
		return fmt.Sprintf("\033[31m%s\033[0m", summary)
	}
	return fmt.Sprintf("\033[32m%s\033[0m", summary)
}
