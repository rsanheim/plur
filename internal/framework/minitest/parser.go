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

// streamRow represents a single PLUR_JSON line from the embedded plugin
// (see plur_plugin.rb). Field names mirror the canonical notification types
// in types/notifications.go that each row feeds: test_result rows fill
// TestCaseNotification, suite_finished rows fill SuiteNotification.
type streamRow struct {
	Type string `json:"type"`

	// test_result fields -> TestCaseNotification
	Status     string  `json:"status"` // "passed", "failed", "error", "skipped"
	ID         string  `json:"id"`     // Klass#test_name -> TestID
	FilePath   string  `json:"file_path"`
	LineNumber int     `json:"line_number"`
	RunTime    float64 `json:"run_time"` // seconds -> Duration

	// dump_failures field -> FormattedFailuresNotification
	FormattedOutput string `json:"formatted_output"`

	// suite_finished fields -> SuiteNotification
	TestCount      int `json:"test_count"`
	AssertionCount int `json:"assertion_count"`
	FailureCount   int `json:"failure_count"`
	ErrorCount     int `json:"error_count"`
	PendingCount   int `json:"pending_count"`
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
	case "suite_finished":
		return append(prefix, types.SuiteNotification{
			Event:          types.SuiteFinished,
			TestCount:      row.TestCount,
			AssertionCount: row.AssertionCount,
			FailureCount:   row.FailureCount,
			ErrorCount:     row.ErrorCount,
			PendingCount:   row.PendingCount,
		}), true
	}

	return prefix, true // unknown row types are reporter-internal, never test output
}

// testNotification converts a test_result row to a TestCaseNotification.
// FilePath and Duration feed runtime-based distribution via the tracker.
func testNotification(row streamRow) types.TestCaseNotification {
	var event types.TestEvent
	switch row.Status {
	case "passed":
		event = types.TestPassed
	case "skipped":
		event = types.TestPending
	default: // "failed", "error", and anything minitest adds later
		event = types.TestFailed
	}

	notification := types.TestCaseNotification{
		Event:           event,
		TestID:          row.ID,
		Description:     row.ID,
		FullDescription: row.ID,
		FilePath:        row.FilePath,
		LineNumber:      row.LineNumber,
		Status:          row.Status,
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
	default:
		return "", false
	}
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
	if suite != nil {
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
