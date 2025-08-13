package main

import (
	"strings"
	"time"

	"github.com/rsanheim/plur/types"
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
	tc := &TestCollector{
		tests:    make([]types.TestCaseNotification, 0, 100), // Pre-allocate for ~100 tests
		failures: make([]types.TestCaseNotification, 0, 10),  // Pre-allocate for ~10 failures
		pending:  make([]types.TestCaseNotification, 0, 10),  // Pre-allocate for ~10 pending
	}
	// Pre-allocate string builder for typical output size (4KB)
	tc.rawOutput.Grow(4096)
	return tc
}

// NewTestCollectorWithHints creates a test collector with size hints based on test suite characteristics
func NewTestCollectorWithHints(numFiles int, estimatedTestsPerFile int) *TestCollector {
	// Calculate capacity hints based on suite size
	expectedTests := numFiles * estimatedTestsPerFile
	if expectedTests < 10 {
		expectedTests = 10 // Minimum capacity
	}

	// Assume 5% failure rate, 5% pending rate (adjustable based on project history)
	expectedFailures := expectedTests / 20
	if expectedFailures < 5 {
		expectedFailures = 5
	}

	tc := &TestCollector{
		tests:    make([]types.TestCaseNotification, 0, expectedTests),
		failures: make([]types.TestCaseNotification, 0, expectedFailures),
		pending:  make([]types.TestCaseNotification, 0, expectedFailures),
	}

	// Pre-allocate string builder based on expected output
	// Assume ~100 bytes per test + 2KB base overhead
	outputSize := expectedTests*100 + 2048
	if outputSize > 1024*1024 { // Cap at 1MB to avoid over-allocation
		outputSize = 1024 * 1024
	}
	tc.rawOutput.Grow(outputSize)

	return tc
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
	case types.SuiteStarted:
		if suite, ok := n.(types.SuiteNotification); ok {
			// Store the suite info from SuiteStarted which contains the load time
			if a.suiteInfo == nil {
				a.suiteInfo = &suite
			} else {
				// Preserve the load time from SuiteStarted
				a.suiteInfo.LoadTime = suite.LoadTime
				if suite.TestCount > 0 {
					a.suiteInfo.TestCount = suite.TestCount
				}
			}
		}
	case types.SuiteFinished:
		if suite, ok := n.(types.SuiteNotification); ok {
			if a.suiteInfo == nil {
				a.suiteInfo = &suite
			} else {
				// Update suite info with finish data, but preserve LoadTime from SuiteStarted
				loadTime := a.suiteInfo.LoadTime
				a.suiteInfo = &suite
				a.suiteInfo.LoadTime = loadTime
			}
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

// BuildResult creates a WorkerResult from collected notifications
func (a *TestCollector) BuildResult(testFile *TestFile, duration time.Duration) WorkerResult {
	result := WorkerResult{
		File:              testFile,
		Output:            a.rawOutput.String(),
		Duration:          duration,
		ExampleCount:      len(a.tests),
		FailureCount:      len(a.failures),
		PendingCount:      len(a.pending),
		Tests:             a.tests,
		State:             types.StateSuccess,
		FormattedFailures: a.formattedFailures,
		FormattedSummary:  a.formattedSummary,
	}

	// Set state based on failures
	if len(a.failures) > 0 {
		result.State = types.StateFailed
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
