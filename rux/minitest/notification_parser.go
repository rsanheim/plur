package minitest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Import the main package types - this will need adjustment after we reorganize
type TestEvent string

const (
	TestPassed    TestEvent = "test_passed"
	TestFailed    TestEvent = "test_failed"
	TestPending   TestEvent = "test_pending"
	TestStarted   TestEvent = "test_started"
	SuiteStarted  TestEvent = "suite_started"
	SuiteFinished TestEvent = "suite_finished"
	RawOutput     TestEvent = "raw_output"
)

type TestNotification interface {
	GetEvent() TestEvent
	GetTestID() string
}

type TestCaseNotification struct {
	Event           TestEvent
	TestID          string
	Description     string
	FullDescription string
	Location        string
	FilePath        string
	LineNumber      int
	Status          string
	Duration        time.Duration
	Exception       *TestException
	PendingMessage  string
}

func (n TestCaseNotification) GetEvent() TestEvent { return n.Event }
func (n TestCaseNotification) GetTestID() string   { return n.TestID }

type TestException struct {
	Class     string
	Message   string
	Backtrace []string
}

type SuiteNotification struct {
	Event        TestEvent
	TestCount    int
	FailureCount int
	PendingCount int
	LoadTime     time.Duration
	Duration     time.Duration
}

func (n SuiteNotification) GetEvent() TestEvent { return n.Event }
func (n SuiteNotification) GetTestID() string   { return "" }

type OutputNotification struct {
	Event   TestEvent
	Content string
}

func (n OutputNotification) GetEvent() TestEvent { return n.Event }
func (n OutputNotification) GetTestID() string   { return "" }

// NotificationParser parses minitest text output into notifications
type NotificationParser struct {
	currentTest     string
	currentLocation string
	inFailure       bool
	failureBuffer   strings.Builder
	testCounter     int
}

// ParseLine parses a single line of minitest output
func (p *NotificationParser) ParseLine(line string) ([]TestNotification, bool) {
	notifications := []TestNotification{}

	// Check for test execution start
	if strings.Contains(line, "in test_") {
		if match := regexp.MustCompile(`in (test_\w+)`).FindStringSubmatch(line); match != nil {
			p.currentTest = match[1]
			notifications = append(notifications, TestCaseNotification{
				Event:       TestStarted,
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

			notifications = append(notifications, TestCaseNotification{
				Event:       TestPassed,
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

			notifications = append(notifications, TestCaseNotification{
				Event:          TestPending,
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

				notification := TestCaseNotification{
					Event:           TestFailed,
					TestID:          location,
					Description:     testName,
					FullDescription: fmt.Sprintf("%s#%s", className, testName),
					Location:        location,
					FilePath:        filePath,
					LineNumber:      lineNum,
					Status:          "failed",
					Exception: &TestException{
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

		notifications = append(notifications, SuiteNotification{
			Event:        SuiteFinished,
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
