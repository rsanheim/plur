package minitest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/types"
)

// OutputParser parses minitest text output into notifications
type OutputParser struct {
	inFailure     bool
	testsRunning  bool
	failureBuffer strings.Builder
	testCounter   int

	// New states for parsing failure details after tests complete
	afterFinished    bool         // Set to true after "Finished in..."
	inFailureDetails bool         // Currently parsing a failure block
	currentFailure   *FailureInfo // Accumulating failure details
}

// FailureInfo holds temporary failure details while parsing
type FailureInfo struct {
	testName   string
	fileName   string
	lineNumber int
	message    strings.Builder
}

// ParseLine parses a single line of minitest output
func (p *OutputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	logger.Logger.Debug("[ParseLine]", "line", line)

	notifications := []types.TestNotification{}

	// Don't skip empty lines if we're parsing failure details
	if line == "" && !p.inFailureDetails {
		return notifications, false
	}

	// Tests are starting, progress will begin.
	if strings.HasPrefix(line, "# Running:") {
		p.testsRunning = true
		notifications = append(notifications, types.SuiteNotification{
			Event: types.SuiteStarted,
		})
		return notifications, false
	}

	// We know the suite is done and progress is done being reported
	if strings.HasPrefix(line, "Finished in") {
		p.testsRunning = false
		p.afterFinished = true
		logger.Logger.Debug("Entered afterFinished state")
		return notifications, false
	}

	// If we get down here, we can try to parse progress
	if p.testsRunning {
		notifications = p.parseProgressLine(line)
		return notifications, false
	}

	// After tests are finished, look for failure details and summary
	if p.afterFinished {
		// Check for summary line first
		if summaryNotification := p.parseSummaryLine(line); summaryNotification != nil {
			notifications = append(notifications, summaryNotification)
			p.afterFinished = false // Reset state after summary
			return notifications, false
		}

		// Check if this is a failure header
		if p.parseFailureHeader(line) {
			return notifications, false
		}

		// If we're in failure details, parse them
		if p.inFailureDetails {
			logger.Logger.Debug("In failure details mode", "line", line)
			// Check if this is the test location line
			if p.parseFailureLocation(line) {
				return notifications, false
			}

			// Check if we've reached the end of this failure (empty line or next failure)
			if line == "" || regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):`).MatchString(line) {
				// Emit the failure notification
				if p.currentFailure != nil && p.currentFailure.testName != "" {
					logger.Logger.Debug("Creating failure notification for", "testName", p.currentFailure.testName)
					notification := p.createFailureNotification()
					if notification != nil {
						notifications = append(notifications, notification)
						logger.Logger.Debug("Added failure notification to list")
					}
				} else {
					var testName string
					if p.currentFailure != nil {
						testName = p.currentFailure.testName
					}
					logger.Logger.Debug("Not creating failure notification",
						"currentFailure", p.currentFailure != nil,
						"testName", testName)
				}
				p.inFailureDetails = false
				p.currentFailure = nil

				// If this was another failure header, parse it
				if line != "" {
					p.parseFailureHeader(line)
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
			return notifications, false
		}
	}

	return notifications, false // Minitest output is always preserved
}

func (p *OutputParser) parseSummaryLine(line string) types.TestNotification {
	// Check for summary line
	if match := regexp.MustCompile(`(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?`).FindStringSubmatch(line); match != nil {
		runs, _ := strconv.Atoi(match[1])
		failures, _ := strconv.Atoi(match[3])
		errors, _ := strconv.Atoi(match[4])
		skips, _ := strconv.Atoi(match[5])
		finishNotification := types.SuiteNotification{
			Event:        types.SuiteFinished,
			TestCount:    runs,
			FailureCount: failures + errors,
			PendingCount: skips,
		}
		return finishNotification
	}
	return nil // Return nil if not a summary line
}

func (p *OutputParser) parseProgressLine(line string) []types.TestNotification {
	notifications := []types.TestNotification{}

	// Check for progress indicators
	for _, char := range line {
		switch char {
		case '.':
			p.testCounter++
			testID := fmt.Sprintf("test_%d", p.testCounter)

			notifications = append(notifications, types.TestCaseNotification{
				Event:       types.TestPassed,
				TestID:      testID,
				Description: testID,
				Status:      "passed",
			})

		case 'F':
			p.testCounter++
			testID := fmt.Sprintf("test_%d", p.testCounter)

			notifications = append(notifications, types.TestCaseNotification{
				Event:       types.TestFailed,
				TestID:      testID,
				Description: testID,
				Status:      "failed",
			})

		case 'E':
			p.testCounter++
			testID := fmt.Sprintf("test_%d", p.testCounter)

			notifications = append(notifications, types.TestCaseNotification{
				Event:       types.TestError,
				TestID:      testID,
				Description: testID,
				Status:      "error",
			})

		case 'S':
			p.testCounter++
			testID := fmt.Sprintf("test_%d", p.testCounter)

			notifications = append(notifications, types.TestCaseNotification{
				Event:          types.TestPending,
				TestID:         testID,
				Description:    testID,
				Status:         "skipped",
				PendingMessage: "Skipped",
			})
		default:
			// do nothing for now
		}
	}
	return notifications
}

// parseFailureHeader checks if a line is a failure header and extracts info
// Example: "  1) Failure:"
func (p *OutputParser) parseFailureHeader(line string) bool {
	// Check for failure header pattern: "  N) Failure:" or "  N) Error:"
	match := regexp.MustCompile(`^\s*\d+\)\s+(Failure|Error):\s*$`).MatchString(line)
	if match {
		logger.Logger.Debug("Found failure header", "line", line)
		p.inFailureDetails = true
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

// createFailureNotification creates a TestCaseNotification from accumulated failure info
func (p *OutputParser) createFailureNotification() types.TestNotification {
	if p.currentFailure == nil {
		return nil
	}

	// Extract the test ID from the test name (e.g., "MixedResultsTest#test_email_validation_mixed")
	testID := p.currentFailure.testName

	// Create the exception info
	exception := &types.TestException{
		Class:   "Minitest::Assertion", // Minitest uses this for failures
		Message: strings.TrimSpace(p.currentFailure.message.String()),
		// TODO: Backtrace would need to be parsed from additional lines if present
		Backtrace: []string{},
	}

	notification := types.TestCaseNotification{
		Event:           types.TestFailed,
		TestID:          testID,
		Description:     testID,
		FullDescription: testID,
		Location:        fmt.Sprintf("%s:%d", p.currentFailure.fileName, p.currentFailure.lineNumber),
		FilePath:        p.currentFailure.fileName,
		LineNumber:      p.currentFailure.lineNumber,
		Status:          "failed",
		Exception:       exception,
	}

	logger.Logger.Debug("Created failure notification",
		"testID", notification.TestID,
		"location", notification.Location,
		"message", exception.Message)

	return notification
}
