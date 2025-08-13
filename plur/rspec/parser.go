package rspec

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/format"
	"github.com/rsanheim/plur/types"
)

// outputParser parses RSpec JSON output into notifications
type outputParser struct{}

// NewOutputParser creates a new RSpec output parser
func NewOutputParser() types.TestOutputParser {
	return &outputParser{}
}

func (p *outputParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	switch notification.GetEvent() {
	case types.TestPassed:
		return "dot", true
	case types.TestFailed:
		return "failure", true
	case types.TestError:
		return "error", true
	case types.TestPending:
		return "pending", true
	}
	return "", false
}

// FormatSummary formats a test summary in RSpec style
func (p *outputParser) FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string {
	summary := fmt.Sprintf("Finished in %s (files took %s to load)\n", format.FormatDuration(wallTime), format.FormatDuration(loadTime))

	// Format example count
	exampleText := "1 example"
	if totalExamples != 1 {
		exampleText = fmt.Sprintf("%d examples", totalExamples)
	}

	// Format failure count
	failureText := "0 failures"
	if totalFailures == 1 {
		failureText = "1 failure"
	} else if totalFailures > 1 {
		failureText = fmt.Sprintf("%d failures", totalFailures)
	}

	// Format pending count if any
	pendingText := ""
	if totalPending > 0 {
		if totalPending == 1 {
			pendingText = ", 1 pending"
		} else {
			pendingText = fmt.Sprintf(", %d pending", totalPending)
		}
	}

	summary += fmt.Sprintf("%s, %s%s", exampleText, failureText, pendingText)
	return summary
}

// ParseLine parses a single line of RSpec output
func (p *outputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	notifications := []types.TestNotification{}

	// Check if it's a JSON line
	if strings.HasPrefix(line, "PLUR_JSON:") {
		jsonStr := strings.TrimPrefix(line, "PLUR_JSON:")

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
			return nil, false
		}

		msgType, _ := msg["type"].(string)

		switch msgType {
		case "message":
			// Handle error messages from RSpec (e.g., syntax errors, load errors)
			if message, ok := msg["message"].(string); ok && message != "" {
				notifications = append(notifications, types.OutputNotification{
					Event:   types.RawOutput,
					Content: message,
				})
			}

		case "load_summary":
			if summary, ok := msg["summary"].(map[string]interface{}); ok {
				count := getInt(summary, "count")
				loadTime := getFloat(summary, "load_time")

				notifications = append(notifications, types.SuiteNotification{
					Event:     types.SuiteStarted,
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

		case "dump_failures":
			// Handle formatted failure output from RSpec
			if formattedOutput, ok := msg["formatted_output"].(string); ok && formattedOutput != "" {
				notifications = append(notifications, types.FormattedFailuresNotification{
					Content: formattedOutput,
				})
			}

		case "dump_summary":
			count := getInt(msg, "example_count")
			failures := getInt(msg, "failure_count")
			pending := getInt(msg, "pending_count")
			duration := getFloat(msg, "duration")

			notifications = append(notifications, types.SuiteNotification{
				Event:        types.SuiteFinished,
				TestCount:    count,
				FailureCount: failures,
				PendingCount: pending,
				Duration:     time.Duration(duration * float64(time.Second)),
			})

			// Also handle formatted summary output if present
			if formattedOutput, ok := msg["formatted_output"].(string); ok && formattedOutput != "" {
				notifications = append(notifications, types.FormattedSummaryNotification{
					Content: formattedOutput,
				})
			}
		}

		return notifications, true // Line was consumed
	}

	// Not a JSON line - return as raw output
	if line != "" {
		notifications = append(notifications, types.OutputNotification{
			Event:   types.RawOutput,
			Content: line,
		})
	}

	return notifications, false // Line was not consumed
}

func (p *outputParser) parseExample(msgType string, example map[string]interface{}) types.TestNotification {
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
	var event types.TestEvent
	switch msgType {
	case "example_passed":
		event = types.TestPassed
	case "example_failed":
		event = types.TestFailed
	case "example_pending":
		event = types.TestPending
	}

	notification := types.TestCaseNotification{
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
			notification.Exception = &types.TestException{
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

// FormatFailures formats individual failure details in RSpec style
func (p *outputParser) FormatFailures(failures []types.TestCaseNotification) string {
	// RSpec provides pre-formatted failures via FormattedFailuresNotification
	// This method is only used as a fallback
	if len(failures) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nFailures:\n")

	// Convert TestCaseNotification to FailureDetail and use existing formatter
	for i, failure := range failures {
		detail := FailureDetail{
			Description: failure.FullDescription,
			FilePath:    failure.FilePath,
			LineNumber:  failure.LineNumber,
		}

		if failure.Exception != nil {
			detail.ErrorClass = failure.Exception.Class
			detail.Message = failure.Exception.Message
			detail.Backtrace = failure.Exception.Backtrace
		}

		sb.WriteString(FormatFailure(i+1, detail))
		sb.WriteString("\n") // Extra line between failures
	}

	return sb.String()
}

// FormatFailuresList formats a list of failures with file:line references for re-running
func (p *outputParser) FormatFailuresList(failures []types.TestCaseNotification) string {
	if len(failures) == 0 {
		return ""
	}

	// Convert to FailureDetail and use existing formatter
	var details []FailureDetail
	for _, failure := range failures {
		details = append(details, FailureDetail{
			Description: failure.FullDescription,
			FilePath:    failure.FilePath,
			LineNumber:  failure.LineNumber,
		})
	}

	return FormatFailedExamples(details)
}

// ColorizeSummary applies color to a summary based on success/failure state
func (p *outputParser) ColorizeSummary(summary string, hasFailures bool) string {
	if hasFailures {
		return fmt.Sprintf("\033[31m%s\033[0m", summary)
	}
	return fmt.Sprintf("\033[32m%s\033[0m", summary)
}
