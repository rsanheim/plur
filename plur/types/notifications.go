package types

import "time"

// TestState represents the state of a test execution
type TestState string

const (
	StateSuccess TestState = "success" // passing
	StateFailed  TestState = "failed"  // failure - i.e. assertion failure
	StateError   TestState = "error"   // error - i.e. exception
)

// TestEvent represents the type of test event
type TestEvent string

const (
	TestPassed    TestEvent = "test_passed"
	TestFailed    TestEvent = "test_failed"
	TestPending   TestEvent = "test_pending"
	TestStarted   TestEvent = "test_started"
	SuiteStarted  TestEvent = "suite_started"
	SuiteFinished TestEvent = "suite_finished"
	RawOutput     TestEvent = "raw_output"
	Progress      TestEvent = "progress" // Progress indicator for real-time display
)

// TestNotification is the interface that all notifications implement
type TestNotification interface {
	GetEvent() TestEvent
	GetTestID() string
}

// TestCaseNotification represents events for individual test cases
type TestCaseNotification struct {
	Event           TestEvent
	TestID          string
	Description     string
	FullDescription string
	Location        string // e.g. "./spec/foo_spec.rb:42"
	FilePath        string
	LineNumber      int
	Status          string // Original status from framework
	Duration        time.Duration

	// Only populated for failures
	Exception *TestException

	// Only populated for pending tests
	PendingMessage string
}

func (n TestCaseNotification) GetEvent() TestEvent { return n.Event }
func (n TestCaseNotification) GetTestID() string   { return n.TestID }

// TestException contains failure information
type TestException struct {
	Class     string
	Message   string
	Backtrace []string
}

// SuiteNotification represents suite-level events
type SuiteNotification struct {
	Event          TestEvent
	TestCount      int
	AssertionCount int
	FailureCount   int
	ErrorCount     int
	PendingCount   int
	LoadTime       time.Duration
	Duration       time.Duration
}

func (n SuiteNotification) GetEvent() TestEvent { return n.Event }
func (n SuiteNotification) GetTestID() string   { return "" } // Suite events don't have test IDs

// OutputNotification represents raw output that doesn't match patterns
type OutputNotification struct {
	Event   TestEvent // Always RawOutput
	Content string
}

func (n OutputNotification) GetEvent() TestEvent { return n.Event }
func (n OutputNotification) GetTestID() string   { return "" }

// FormattedFailuresNotification is a special notification for RSpec's formatted failure output
type FormattedFailuresNotification struct {
	Content string
}

func (n FormattedFailuresNotification) GetEvent() TestEvent { return RawOutput }
func (n FormattedFailuresNotification) GetTestID() string   { return "" }

// FormattedPendingNotification is a special notification for RSpec's formatted pending output
type FormattedPendingNotification struct {
	Content string
}

func (n FormattedPendingNotification) GetEvent() TestEvent { return RawOutput }
func (n FormattedPendingNotification) GetTestID() string   { return "" }

// FormattedSummaryNotification is a special notification for RSpec's formatted summary
type FormattedSummaryNotification struct {
	Content string
}

func (n FormattedSummaryNotification) GetEvent() TestEvent { return RawOutput }
func (n FormattedSummaryNotification) GetTestID() string   { return "" }

// GroupStartedNotification represents a test group (describe block) starting
type GroupStartedNotification struct {
	FilePath string
}

func (n GroupStartedNotification) GetEvent() TestEvent { return SuiteStarted }
func (n GroupStartedNotification) GetTestID() string   { return "" }

// ProgressEvent represents a progress indicator for real-time display only
// This is not a test result, just a display notification
type ProgressEvent struct {
	Event     TestEvent // Always Progress
	Character string    // '.', 'F', 'E', 'S'
	Index     int       // Position in test run (0-based)
}

func (n ProgressEvent) GetEvent() TestEvent { return n.Event }
func (n ProgressEvent) GetTestID() string   { return "" }
