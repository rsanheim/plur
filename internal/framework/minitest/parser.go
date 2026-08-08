package minitest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/format"
	"github.com/rsanheim/plur/types"
)

const jsonPrefix string = "PLUR_JSON:"
const jsonPrefixLen int = len(jsonPrefix)

// streamRow represents a single PLUR_JSON line from the embedded reporter.
// See framework/minitest/reporter.rb for the reporter implementation.
type streamRow struct {
	Type string `json:"type"`

	// test_result fields
	Code       string  `json:"code"` // ".", "F", "E", "S"
	ID         string  `json:"id"`   // Klass#test_name
	FilePath   string  `json:"file_path"`
	LineNumber int     `json:"line_number"`
	RunTime    float64 `json:"run_time"`

	// dump_failures field
	FormattedOutput string `json:"formatted_output"`

	// summary fields (assertions is shared with test_result)
	Count      int `json:"count"`
	Assertions int `json:"assertions"`
	Failures   int `json:"failures"`
	Errors     int `json:"errors"`
	Skips      int `json:"skips"`
}

// outputParser parses the embedded reporter's JSON rows into notifications.
// Any line without the row prefix is test-written stdout: with minitest's own
// reporters replaced, nothing else writes to the pipe.
type outputParser struct {
	startTime time.Time // When the parser was created (for load time calculation)
}

// NewOutputParser creates a new minitest output parser
func NewOutputParser() types.TestOutputParser {
	return &outputParser{
		startTime: time.Now(),
	}
}

// CurrentFile returns empty string for minitest (no structured file tracking)
func (p *outputParser) CurrentFile() string {
	return ""
}

// ParseLine parses a single line of output from our Ruby based Plur::MinitestReporter
func (p *outputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	idx := strings.Index(line, jsonPrefix)
	if idx == -1 {
		return nil, false // test-written output: preserved and streamed by the caller
	}

	var row streamRow
	if err := json.Unmarshal([]byte(line[idx+jsonPrefixLen:]), &row); err != nil {
		return nil, false
	}

	var prefix []types.TestNotification
	if idx > 0 {
		// A test wrote a partial line (print with no trailing newline), so the
		// reporter's row landed on the same physical line. The bytes before
		// the marker are that test's output.
		prefix = append(prefix, types.OutputNotification{
			Event:   types.TestStdout,
			Content: line[:idx],
		})
	}

	switch row.Type {
	case "suite_started":
		return append(prefix, types.SuiteNotification{
			Event:    types.SuiteStarted,
			LoadTime: time.Since(p.startTime),
		}), true
	case "test_result":
		return append(prefix, testNotification(row)), true
	case "dump_failures":
		if row.FormattedOutput == "" {
			return prefix, true
		}
		return append(prefix, types.FormattedFailuresNotification{Content: row.FormattedOutput}), true
	case "summary":
		return append(prefix, types.SuiteNotification{
			Event:          types.SuiteFinished,
			TestCount:      row.Count,
			AssertionCount: row.Assertions,
			FailureCount:   row.Failures,
			ErrorCount:     row.Errors,
			PendingCount:   row.Skips,
		}), true
	}

	return prefix, true // unknown row types are reporter-internal, never test output
}

// testNotification converts a test_result row to a TestCaseNotification.
// FilePath and Duration feed runtime-based distribution via the tracker.
func testNotification(row streamRow) types.TestCaseNotification {
	var event types.TestEvent
	var status string
	switch row.Code {
	case ".":
		event, status = types.TestPassed, "passed"
	case "S":
		event, status = types.TestPending, "skipped"
	case "E":
		event, status = types.TestFailed, "error"
	default: // "F" and any result code minitest adds later
		event, status = types.TestFailed, "failed"
	}

	notification := types.TestCaseNotification{
		Event:           event,
		TestID:          row.ID,
		Description:     row.ID,
		FullDescription: row.ID,
		FilePath:        row.FilePath,
		LineNumber:      row.LineNumber,
		Status:          status,
		Duration:        time.Duration(row.RunTime * float64(time.Second)),
	}
	if row.FilePath != "" {
		notification.Location = fmt.Sprintf("%s:%d", row.FilePath, row.LineNumber)
	}
	return notification
}

// Converts a TestNotification to a progress type (just a string for now) for streaming to output
func (p *outputParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	switch notification.GetEvent() {
	case types.TestPassed:
		return "dot", true
	case types.TestFailed:
		if test, ok := notification.(types.TestCaseNotification); ok && test.Status == "error" {
			return "error_progress", true
		}
		return "failure", true
	case types.TestPending:
		return "pending", true
	}
	return "", false
}

// FormatSummary formats a test summary in minitest style
func (p *outputParser) FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string {
	// Minitest doesn't typically show load time in the summary
	// Format: "X runs, Y assertions, Z failures, W errors, V skips"

	runText := "1 run"
	if totalExamples != 1 {
		runText = fmt.Sprintf("%d runs", totalExamples)
	}

	assertionCount := totalExamples
	if suite != nil && suite.AssertionCount > 0 {
		assertionCount = suite.AssertionCount
	}
	assertionText := "1 assertion"
	if assertionCount != 1 {
		assertionText = fmt.Sprintf("%d assertions", assertionCount)
	}

	errorCount := 0
	if suite != nil {
		errorCount = suite.ErrorCount
	}
	failureCount := totalFailures
	failureText := "0 failures"
	if failureCount == 1 {
		failureText = "1 failure"
	} else if failureCount > 1 {
		failureText = fmt.Sprintf("%d failures", failureCount)
	}

	errorText := "0 errors"
	if errorCount == 1 {
		errorText = "1 error"
	} else if errorCount > 1 {
		errorText = fmt.Sprintf("%d errors", errorCount)
	}

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
