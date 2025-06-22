package rspec

import (
	"encoding/json"
	"fmt"
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

// OutputParser parses RSpec JSON output into notifications
type OutputParser struct{}

// ParseLine parses a single line of RSpec output
func (p *OutputParser) ParseLine(line string) ([]TestNotification, bool) {
	notifications := []TestNotification{}

	// Check if it's a JSON line
	if strings.HasPrefix(line, "RUX_JSON:") {
		jsonStr := strings.TrimPrefix(line, "RUX_JSON:")

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
			return nil, false
		}

		msgType, _ := msg["type"].(string)

		switch msgType {
		case "load_summary":
			if summary, ok := msg["summary"].(map[string]interface{}); ok {
				count := getInt(summary, "count")
				loadTime := getFloat(summary, "load_time")

				notifications = append(notifications, SuiteNotification{
					Event:     SuiteStarted,
					TestCount: count,
					LoadTime:  time.Duration(loadTime * float64(time.Second)),
				})
			}

		case "example_passed", "example_failed", "example_pending":
			if example, ok := msg["example"].(map[string]interface{}); ok {
				notification := p.parseExample(msgType, example)
				if notification != nil {
					notifications = append(notifications, notification)
				}
			}

		case "dump_summary":
			count := getInt(msg, "example_count")
			failures := getInt(msg, "failure_count")
			pending := getInt(msg, "pending_count")
			duration := getFloat(msg, "duration")

			notifications = append(notifications, SuiteNotification{
				Event:        SuiteFinished,
				TestCount:    count,
				FailureCount: failures,
				PendingCount: pending,
				Duration:     time.Duration(duration * float64(time.Second)),
			})
		}

		return notifications, true // Line was consumed
	}

	// Not a JSON line - return as raw output
	if line != "" {
		notifications = append(notifications, OutputNotification{
			Event:   RawOutput,
			Content: line,
		})
	}

	return notifications, false // Line was not consumed
}

func (p *OutputParser) parseExample(msgType string, example map[string]interface{}) TestNotification {
	desc := getString(example, "description")
	fullDesc := getString(example, "full_description")
	location := getString(example, "location")
	filePath := getString(example, "file_path")
	lineNum := getInt(example, "line_number")
	runTime := getFloat(example, "run_time")
	status := getString(example, "status")

	testID := location
	if testID == "" && filePath != "" {
		testID = fmt.Sprintf("%s:%d", filePath, lineNum)
	}

	// Map RSpec type to our TestEvent
	var event TestEvent
	switch msgType {
	case "example_passed":
		event = TestPassed
	case "example_failed":
		event = TestFailed
	case "example_pending":
		event = TestPending
	}

	notification := TestCaseNotification{
		Event:           event,
		TestID:          testID,
		Description:     desc,
		FullDescription: fullDesc,
		Location:        location,
		FilePath:        filePath,
		LineNumber:      lineNum,
		Status:          status,
		Duration:        time.Duration(runTime * float64(time.Second)),
	}

	// Handle failure details
	if msgType == "example_failed" {
		if exception, ok := example["exception"].(map[string]interface{}); ok {
			notification.Exception = &TestException{
				Class:   getString(exception, "class"),
				Message: getString(exception, "message"),
			}
			if backtrace, ok := exception["backtrace"].([]interface{}); ok {
				for _, line := range backtrace {
					if str, ok := line.(string); ok {
						notification.Exception.Backtrace = append(notification.Exception.Backtrace, str)
					}
				}
			}
		}
	}

	// Handle pending message
	if msgType == "example_pending" {
		notification.PendingMessage = getString(example, "pending_message")
	}

	return notification
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}
