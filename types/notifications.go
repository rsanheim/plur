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
	TestStdout    TestEvent = "test_stdout" // Test-written stdout split off a consumed line; streamed live
)

// TestNotification is the interface that all notifications implement
type TestNotification interface {
	GetEvent() TestEvent
}

// TestCaseNotification represents events for individual test cases
type TestCaseNotification struct {
	Event           TestEvent
	TestID          string // RSpec canonical example.id when available; otherwise location-derived
	Description     string
	FullDescription string
	Location        string // e.g. "./spec/foo_spec.rb:42"
	FilePath        string // project-relative path (no "./" prefix)
	LineNumber      int
	Status          string // Original status from framework
	Duration        time.Duration

	// RSpec-specific identifiers (empty for other frameworks)
	AbsoluteFilePath      string // file_path metadata.absolute_file_path
	LocationRerunArgument string // file:line argument RSpec recommends for re-running
	ScopedID              string // RSpec metadata[:scoped_id]

	// Only populated for pending tests
	PendingMessage string
}

func (n TestCaseNotification) GetEvent() TestEvent { return n.Event }

// SuiteNotification represents suite-level events
type SuiteNotification struct {
	Event          TestEvent
	TestCount      int
	AssertionCount int // Only populated for minitest
	FailureCount   int
	ErrorCount     int
	PendingCount   int
	LoadTime       time.Duration
	Duration       time.Duration
}

func (n SuiteNotification) GetEvent() TestEvent { return n.Event }

// OutputNotification represents raw output that doesn't match patterns
type OutputNotification struct {
	Event   TestEvent // RawOutput, or TestStdout for output split off a consumed line
	Content string
}

func (n OutputNotification) GetEvent() TestEvent { return n.Event }

// FormattedFailuresNotification is a special notification for RSpec's formatted failure output
type FormattedFailuresNotification struct {
	Content string
}

func (n FormattedFailuresNotification) GetEvent() TestEvent { return RawOutput }

// FormattedPendingNotification is a special notification for RSpec's formatted pending output
type FormattedPendingNotification struct {
	Content string
}

func (n FormattedPendingNotification) GetEvent() TestEvent { return RawOutput }

// FormattedSummaryNotification is a special notification for RSpec's formatted summary
type FormattedSummaryNotification struct {
	Content string
}

func (n FormattedSummaryNotification) GetEvent() TestEvent { return RawOutput }
