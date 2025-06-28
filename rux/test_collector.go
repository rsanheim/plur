package main

import (
	"strings"
	"time"

	"github.com/rsanheim/rux/types"
)

// TestCollector collects test notifications and builds the final test result
type TestCollector struct {
	tests             []types.TestCaseNotification
	failures          []types.TestCaseNotification
	pending           []types.TestCaseNotification
	suiteInfo         *types.SuiteNotification
	rawOutput         strings.Builder
	formattedFailures string
	formattedSummary  string
}

// NewTestCollector creates a new test collector
func NewTestCollector() *TestCollector {
	return &TestCollector{
		tests:    make([]types.TestCaseNotification, 0),
		failures: make([]types.TestCaseNotification, 0),
		pending:  make([]types.TestCaseNotification, 0),
	}
}

// AddNotification adds a notification to the collector
func (a *TestCollector) AddNotification(n types.TestNotification) {
	switch n.GetEvent() {
	case types.TestPassed, types.TestFailed, types.TestPending:
		if tc, ok := n.(types.TestCaseNotification); ok {
			a.tests = append(a.tests, tc)
			switch n.GetEvent() {
			case types.TestFailed:
				a.failures = append(a.failures, tc)
			case types.TestPending:
				a.pending = append(a.pending, tc)
			}
		}
	case types.SuiteFinished:
		if suite, ok := n.(types.SuiteNotification); ok {
			a.suiteInfo = &suite
		}
	case types.RawOutput:
		// Handle special formatted notifications
		switch v := n.(type) {
		case types.FormattedFailuresNotification:
			a.formattedFailures = v.Content
		case types.FormattedSummaryNotification:
			a.formattedSummary = v.Content
		case types.OutputNotification:
			a.rawOutput.WriteString(v.Content + "\n")
		}
	}
}

// BuildResult creates a TestResult from collected notifications
func (a *TestCollector) BuildResult(testFile *TestFile, duration time.Duration) TestResult {
	// Convert TestCaseNotification failures to TestFailure format
	failures := make([]TestFailure, 0)
	for _, notification := range a.failures {
		failure := TestFailure{
			File:        testFile,
			Description: notification.FullDescription,
			LineNumber:  notification.LineNumber,
		}

		if notification.Exception != nil {
			failure.Message = notification.Exception.Message
			failure.Backtrace = notification.Exception.Backtrace
		}

		failures = append(failures, failure)
	}

	result := TestResult{
		File:              testFile,
		Output:            a.rawOutput.String(),
		Duration:          duration,
		Failures:          failures,
		ExampleCount:      len(a.tests),
		FailureCount:      len(a.failures),
		PendingCount:      len(a.pending),
		Tests:             a.tests,
		State:             StateSuccess,
		FormattedFailures: a.formattedFailures,
		FormattedSummary:  a.formattedSummary,
	}

	// Set state based on failures
	if len(a.failures) > 0 {
		result.State = StateFailed
	}

	// If we have suite info, use its values
	if a.suiteInfo != nil {
		result.FileLoadTime = a.suiteInfo.LoadTime
		// For minitest, use the test count from the summary if available
		if a.suiteInfo.TestCount > 0 {
			result.ExampleCount = a.suiteInfo.TestCount
		}
		// Use suite's failure count if available (includes both failures and errors)
		if a.suiteInfo.FailureCount >= 0 {
			result.FailureCount = a.suiteInfo.FailureCount
		}
		// Use suite's pending count if available
		if a.suiteInfo.PendingCount >= 0 {
			result.PendingCount = a.suiteInfo.PendingCount
		}
	}

	return result
}

// GetTests returns all collected test case notifications
func (a *TestCollector) GetTests() []types.TestCaseNotification {
	return a.tests
}

// GetFailures returns all failure notifications
func (a *TestCollector) GetFailures() []types.TestCaseNotification {
	return a.failures
}

// GetPending returns all pending test notifications
func (a *TestCollector) GetPending() []types.TestCaseNotification {
	return a.pending
}

// GetSuiteInfo returns the suite notification if available
func (a *TestCollector) GetSuiteInfo() *types.SuiteNotification {
	return a.suiteInfo
}

// GetFormattedFailures returns the formatted failures if available
func (a *TestCollector) GetFormattedFailures() string {
	return a.formattedFailures
}

// GetFormattedSummary returns the formatted summary if available
func (a *TestCollector) GetFormattedSummary() string {
	return a.formattedSummary
}
