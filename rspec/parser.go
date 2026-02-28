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
type outputParser struct {
	currentFile string // tracks the current file being tested (from group_started)
}

// NewOutputParser creates a new RSpec output parser
func NewOutputParser() types.TestOutputParser {
	return &outputParser{}
}

// CurrentFile returns the current file being tested
func (p *outputParser) CurrentFile() string {
	return p.currentFile
}

func (p *outputParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	switch notification.GetEvent() {
	case types.TestPassed:
		return "dot", true
	case types.TestFailed:
		return "failure", true
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

	errorText := ""
	errorCount := 0
	if suite != nil {
		errorCount = suite.ErrorCount
	}
	if errorCount > 0 {
		if errorCount == 1 {
			errorText = ", 1 error occurred outside of examples"
		} else {
			errorText = fmt.Sprintf(", %d errors occurred outside of examples", errorCount)
		}
	}
	summary += errorText
	return summary
}

const jsonPrefix string = "PLUR_JSON:"
const jsonPrefixLen int = len(jsonPrefix)

// ParseLine parses a single line of RSpec output from our Ruby based Plur::JsonRowsFormatter
// See rspec/formatter.rb for the formatter implementation.
func (p *outputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	notifications := []types.TestNotification{}

	// Check if it's a JSON line
	if strings.HasPrefix(line, jsonPrefix) {
		jsonStr := line[jsonPrefixLen:]

		var msg StreamingMessage
		if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
			return nil, false
		}

		switch msg.Type {
		case "message":
			if msg.Message != "" {
				notifications = append(notifications, types.OutputNotification{
					Event:   types.RawOutput,
					Content: msg.Message,
				})
			}
		case "load_summary":
			if msg.Summary != nil {
				notifications = append(notifications, types.SuiteNotification{
					Event:     types.SuiteStarted,
					TestCount: msg.Summary.Count,
					LoadTime:  time.Duration(msg.Summary.LoadTime * float64(time.Second)),
				})
			}
		case "group_started":
			if msg.ExampleGroup != nil && msg.ExampleGroup.FilePath != "" {
				// RSpec outputs paths with "./" prefix, normalize to match glob discovery
				filePath := strings.TrimPrefix(msg.ExampleGroup.FilePath, "./")
				p.currentFile = filePath
				notifications = append(notifications, types.GroupStartedNotification{
					FilePath: filePath,
				})
			}
		case "example_passed", "example_failed", "example_pending":
			if msg.Example != nil {
				notification := p.parseStreamExample(msg.Type, msg.Example)
				notifications = append(notifications, notification)
			}
		case "dump_failures":
			if msg.FormattedOutput != "" {
				notifications = append(notifications, types.FormattedFailuresNotification{Content: msg.FormattedOutput})
			}
		case "dump_pending":
			if msg.FormattedOutput != "" {
				notifications = append(notifications, types.FormattedPendingNotification{Content: msg.FormattedOutput})
			}
		case "dump_summary":
			notifications = append(notifications, types.SuiteNotification{
				Event:        types.SuiteFinished,
				TestCount:    msg.ExampleCount,
				FailureCount: msg.FailureCount,
				ErrorCount:   msg.ErrorCount,
				PendingCount: msg.PendingCount,
				Duration:     time.Duration(msg.Duration * float64(time.Second)),
			})

			// Also handle formatted summary output if present
			if msg.FormattedOutput != "" {
				notifications = append(notifications, types.FormattedSummaryNotification{
					Content: msg.FormattedOutput,
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

// parseStreamExample converts a StreamExample to a TestCaseNotification
func (p *outputParser) parseStreamExample(msgType string, ex *StreamExample) types.TestNotification {
	testID := ex.Location
	if testID == "" && ex.FilePath != "" {
		testID = fmt.Sprintf("%s:%d", ex.FilePath, ex.LineNumber)
	}

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
		Description:     ex.Description,
		FullDescription: ex.FullDescription,
		Location:        ex.Location,
		FilePath:        strings.TrimPrefix(ex.FilePath, "./"),
		LineNumber:      ex.LineNumber,
		Status:          ex.Status,
		Duration:        time.Duration(ex.RunTime * float64(time.Second)),
	}

	if msgType == "example_failed" && ex.Exception != nil {
		notification.Exception = &types.TestException{
			Class:     ex.Exception.Class,
			Message:   ex.Exception.Message,
			Backtrace: ex.Exception.Backtrace,
		}
	}

	if msgType == "example_pending" {
		notification.PendingMessage = ex.PendingMessage
	}

	return notification
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
