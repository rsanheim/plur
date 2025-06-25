package minitest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rsanheim/rux/types"
)

// OutputParser parses minitest text output into notifications
type OutputParser struct {
	currentTest     string
	currentLocation string
	inFailure       bool
	failureBuffer   strings.Builder
	testCounter     int
}

// ParseLine parses a single line of minitest output
func (p *OutputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	notifications := []types.TestNotification{}

	// Check for test execution start
	if strings.Contains(line, "in test_") {
		if match := regexp.MustCompile(`in (test_\w+)`).FindStringSubmatch(line); match != nil {
			p.currentTest = match[1]
			notifications = append(notifications, types.TestCaseNotification{
				Event:       types.TestStarted,
				TestID:      p.currentTest,
				Description: p.currentTest,
				Status:      "running",
			})
		}
	}

	// Check for progress indicators
	for _, char := range line {
		switch char {
		case '.':
			p.testCounter++
			testID := p.currentTest
			if testID == "" {
				testID = fmt.Sprintf("test_%d", p.testCounter)
			}

			notifications = append(notifications, types.TestCaseNotification{
				Event:       types.TestPassed,
				TestID:      testID,
				Description: testID,
				Status:      "passed",
				Location:    p.currentLocation,
			})

		case 'F':
			p.testCounter++
			p.inFailure = true
			p.failureBuffer.Reset()

		case 'E':
			p.testCounter++
			p.inFailure = true
			p.failureBuffer.Reset()

		case 'S':
			p.testCounter++
			testID := p.currentTest
			if testID == "" {
				testID = fmt.Sprintf("test_%d", p.testCounter)
			}

			notifications = append(notifications, types.TestCaseNotification{
				Event:          types.TestPending,
				TestID:         testID,
				Description:    testID,
				Status:         "skipped",
				Location:       p.currentLocation,
				PendingMessage: "Skipped",
			})
		}
	}

	// Handle failure details
	if p.inFailure {
		p.failureBuffer.WriteString(line + "\n")

		// Check if we've captured the whole failure
		if line == "" || strings.HasPrefix(line, "bin/rails test") {
			// Parse the failure
			failureText := p.failureBuffer.String()

			// Extract test name and location
			if match := regexp.MustCompile(`(\w+Test)#(test_\w+) \[(.+):(\d+)\]`).FindStringSubmatch(failureText); match != nil {
				className := match[1]
				testName := match[2]
				filePath := match[3]
				lineNum, _ := strconv.Atoi(match[4])
				location := fmt.Sprintf("%s:%d", filePath, lineNum)

				notification := types.TestCaseNotification{
					Event:           types.TestFailed,
					TestID:          location,
					Description:     testName,
					FullDescription: fmt.Sprintf("%s#%s", className, testName),
					Location:        location,
					FilePath:        filePath,
					LineNumber:      lineNum,
					Status:          "failed",
					Exception: &types.TestException{
						Message: strings.TrimSpace(failureText),
					},
				}

				// Try to extract exception class from error messages
				if strings.Contains(failureText, "Error:") {
					if errorMatch := regexp.MustCompile(`(\w+Error):`).FindStringSubmatch(failureText); errorMatch != nil {
						notification.Exception.Class = errorMatch[1]
					}
				} else if strings.Contains(failureText, "Failure:") {
					notification.Exception.Class = "AssertionFailure"
				}

				// Extract backtrace
				lines := strings.Split(failureText, "\n")
				for _, l := range lines {
					if strings.Contains(l, ".rb:") && strings.TrimSpace(l) != "" {
						notification.Exception.Backtrace = append(notification.Exception.Backtrace, strings.TrimSpace(l))
					}
				}

				notifications = append(notifications, notification)
			}

			p.inFailure = false
		}
	}

	// Check for summary line
	if match := regexp.MustCompile(`(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?`).FindStringSubmatch(line); match != nil {
		runs, _ := strconv.Atoi(match[1])
		failures, _ := strconv.Atoi(match[3])
		errors, _ := strconv.Atoi(match[4])
		skips, _ := strconv.Atoi(match[5])

		notifications = append(notifications, types.SuiteNotification{
			Event:        types.SuiteFinished,
			TestCount:    runs,
			FailureCount: failures + errors,
			PendingCount: skips,
		})
	}

	// Check for duration line
	if match := regexp.MustCompile(`Finished in (\d+\.\d+)s`).FindStringSubmatch(line); match != nil {
		if duration, err := strconv.ParseFloat(match[1], 64); err == nil {
			// Could enhance SuiteNotification with this info
			_ = duration
		}
	}

	return notifications, false // Minitest output is always preserved
}
