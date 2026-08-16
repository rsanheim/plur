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
	loadTime          time.Duration           // from suite_started, else suite_finished
	suiteFinished     bool
	suiteCounts       types.SuiteNotification // counts from suite_finished; authoritative once suiteFinished
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
			collector.loadTime = suite.LoadTime
		}
	case types.SuiteFinished:
		if suite, ok := n.(types.SuiteNotification); ok {
			collector.suiteFinished = true
			collector.suiteCounts = suite
			// suite_started's load time wins when it reported one
			if collector.loadTime == 0 {
				collector.loadTime = suite.LoadTime
			}
		}
	case types.RawOutput, types.TestStdout:
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

	result.FileLoadTime = collector.loadTime

	// Counts from suite_finished are authoritative, zeros included: minitest
	// reports errors separately from failures, so an error-only run has a
	// genuine failure_count of 0 while the per-test tallies above count every
	// non-passing test as a failure. If the worker died before suite_finished
	// arrived, the per-test tallies stand.
	if collector.suiteFinished {
		result.ExampleCount = collector.suiteCounts.TestCount
		result.AssertionCount = collector.suiteCounts.AssertionCount
		result.FailureCount = collector.suiteCounts.FailureCount
		result.ErrorCount = collector.suiteCounts.ErrorCount
		result.PendingCount = collector.suiteCounts.PendingCount
	}

	return result
}
