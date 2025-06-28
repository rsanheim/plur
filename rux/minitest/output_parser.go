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
	state          ParsingState
	progress       ProgressCounts
	failureBuffer  strings.Builder
	currentFailure *FailureInfo // Accumulating failure details

	// Index-based tracking for matching progress to details
	failureIndices    []int // Indices of failed tests (0-based)
	errorIndices      []int // Indices of error tests (0-based)
	currentFailureNum int   // Current failure number being parsed (1-based)
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
				logger.Logger.Debug("Creating failure notification", "testName", p.currentFailure.testName, "failureNum", p.currentFailureNum)
				notification := p.createFailureNotification()
				if notification != nil {
					notifications = append(notifications, notification)
				}
			}
			p.currentFailure = nil
			p.currentFailureNum = 0

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

		// Create success notifications for tests that didn't fail
		failureSet := make(map[int]bool)
		for _, idx := range p.failureIndices {
			failureSet[idx] = true
		}
		for _, idx := range p.errorIndices {
			failureSet[idx] = true
		}

		// Create notifications for passed tests
		for i := 0; i < runs; i++ {
			if !failureSet[i] {
				testID := fmt.Sprintf("test_%d", i+1)
				notifications = append(notifications, types.TestCaseNotification{
					Event:       types.TestPassed,
					TestID:      testID,
					Description: testID, // We don't have the actual test names for passed tests
					Status:      "passed",
				})
			}
		}

		// Add suite finished notification
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
			p.failureIndices = append(p.failureIndices, index)
			p.progress.examples++
			notifications = append(notifications, types.ProgressEvent{
				Event:     types.Progress,
				Character: "F",
				Index:     index,
			})
			logger.Logger.Debug("Progress: failed test", "index", index, "failureNum", len(p.failureIndices))

		case 'E':
			p.progress.errors++
			p.errorIndices = append(p.errorIndices, index)
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
		failureNum, _ := strconv.Atoi(match[1])
		failureType := match[2]

		logger.Logger.Debug("Found failure header", "line", line, "num", failureNum, "type", failureType)
		p.currentFailure = &FailureInfo{}
		p.currentFailureNum = failureNum
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
	if p.currentFailure == nil || p.currentFailureNum == 0 {
		return nil
	}

	// Find the index of this failure
	// currentFailureNum is 1-based, so the first failure (1) maps to failureIndices[0]
	failureIndex := p.currentFailureNum - 1
	if failureIndex >= len(p.failureIndices) {
		logger.Logger.Error("Failure number out of range",
			"failureNum", p.currentFailureNum,
			"failureIndicesLen", len(p.failureIndices))
		return nil
	}

	// Get the test index from our failure indices
	testIndex := p.failureIndices[failureIndex]
	testID := fmt.Sprintf("test_%d", testIndex+1)

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
		"testIndex", testIndex,
		"failureNum", p.currentFailureNum,
		"testID", notification.TestID,
		"testName", notification.Description,
		"location", notification.Location)

	return notification
}
