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
}

// ParseLine parses a single line of minitest output
func (p *OutputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	logger.Logger.Debug("[ParseLine]", "line", line)

	notifications := []types.TestNotification{}

	// skip empty lines
	if line == "" {
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
		return notifications, false
	}

	// If we get down here, we can try to parse progress
	if p.testsRunning {
		notifications = p.parseProgressLine(line)
		return notifications, false
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
	finishNotification := types.SuiteNotification{
		Event: types.SuiteFinished,
	}
	return finishNotification
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
