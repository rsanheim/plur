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
	formattedPending  string
	formattedSummary  string
}

const rawOutputBufferSize = 1024 * 8

// NewTestCollector creates a new test collector
func NewTestCollector() *TestCollector {
	tc := &TestCollector{
		tests:    make([]types.TestCaseNotification, 0, 100), // Pre-allocate for ~100 tests
		failures: make([]types.TestCaseNotification, 0, 10),  // Pre-allocate for ~10 failures
		pending:  make([]types.TestCaseNotification, 0, 10),  // Pre-allocate for ~10 pending
	}
	// Pre-allocate string builder for typical output size (8KB)
	tc.rawOutput.Grow(rawOutputBufferSize)
	return tc
}

// AddNotification adds a notification to the collector
func (collector *TestCollector) AddNotification(n types.TestNotification) {
	switch n.GetEvent() {
	case types.TestPassed, types.TestFailed, types.TestPending:
		if tc, ok := n.(types.TestCaseNotification); ok {
			collector.tests = append(collector.tests, tc)
			switch n.GetEvent() {
			case types.TestFailed:
				collector.failures = append(collector.failures, tc)
			case types.TestPending:
				collector.pending = append(collector.pending, tc)
			}
		}
	case types.SuiteStarted:
		if suite, ok := n.(types.SuiteNotification); ok {
			// Store the suite info from SuiteStarted which contains the load time
			if collector.suiteInfo == nil {
				collector.suiteInfo = &suite
			} else {
				// Preserve the load time from SuiteStarted
				collector.suiteInfo.LoadTime = suite.LoadTime
				if suite.TestCount > 0 {
					collector.suiteInfo.TestCount = suite.TestCount
				}
			}
		}
	case types.SuiteFinished:
		if suite, ok := n.(types.SuiteNotification); ok {
			if collector.suiteInfo == nil {
				collector.suiteInfo = &suite
			} else {
				// Update suite info with finish data, but preserve LoadTime from SuiteStarted
				loadTime := collector.suiteInfo.LoadTime
				collector.suiteInfo = &suite
				collector.suiteInfo.LoadTime = loadTime
			}
		}
	case types.RawOutput:
		// Handle special formatted notifications
		switch v := n.(type) {
		case types.FormattedFailuresNotification:
			collector.formattedFailures = v.Content
		case types.FormattedPendingNotification:
			collector.formattedPending = v.Content
		case types.FormattedSummaryNotification:
			collector.formattedSummary = v.Content
		case types.OutputNotification:
			collector.rawOutput.WriteString(v.Content + "\n")
		}
	}
}

func (collector *TestCollector) BuildResult(duration time.Duration) WorkerResult {
	result := WorkerResult{
		Output:            collector.rawOutput.String(),
		Duration:          duration,
		ExampleCount:      len(collector.tests),
		AssertionCount:    0,
		FailureCount:      len(collector.failures),
		ErrorCount:        0,
		PendingCount:      len(collector.pending),
		Tests:             collector.tests,
		State:             types.StateSuccess,
		FormattedFailures: collector.formattedFailures,
		FormattedPending:  collector.formattedPending,
		FormattedSummary:  collector.formattedSummary,
	}

	// Set state based on failures
	if len(collector.failures) > 0 {
		result.State = types.StateFailed
	}

	// If we have suite info, use its values
	if collector.suiteInfo != nil {
		result.FileLoadTime = collector.suiteInfo.LoadTime
		if collector.suiteInfo.TestCount >= 0 {
			result.ExampleCount = collector.suiteInfo.TestCount
		}
		if collector.suiteInfo.AssertionCount >= 0 {
			result.AssertionCount = collector.suiteInfo.AssertionCount
		}
		// Use suite's failure count if available
		if collector.suiteInfo.FailureCount >= 0 {
			result.FailureCount = collector.suiteInfo.FailureCount
		}
		if collector.suiteInfo.ErrorCount >= 0 {
			result.ErrorCount = collector.suiteInfo.ErrorCount
		}
		// Use suite's pending count if available
		if collector.suiteInfo.PendingCount >= 0 {
			result.PendingCount = collector.suiteInfo.PendingCount
		}
	}

	return result
}
