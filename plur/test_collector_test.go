package main

import (
	"testing"
	"time"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestCollector_AddNotification(t *testing.T) {
	collector := NewTestCollector()

	// Test adding a passed test
	passedTest := types.TestCaseNotification{
		Event:           types.TestPassed,
		TestID:          "test-1",
		Description:     "should pass",
		FullDescription: "Test should pass",
		Location:        "spec/example_spec.rb:10",
		FilePath:        "spec/example_spec.rb",
		LineNumber:      10,
		Duration:        100 * time.Millisecond,
	}
	collector.AddNotification(passedTest)

	// Test adding a failed test
	failedTest := types.TestCaseNotification{
		Event:           types.TestFailed,
		TestID:          "test-2",
		Description:     "should fail",
		FullDescription: "Test should fail",
		Location:        "spec/example_spec.rb:20",
		FilePath:        "spec/example_spec.rb",
		LineNumber:      20,
		Duration:        50 * time.Millisecond,
		Exception: &types.TestException{
			Class:     "RSpec::Expectations::ExpectationNotMetError",
			Message:   "expected true to be false",
			Backtrace: []string{"spec/example_spec.rb:20:in `block (2 levels) in <top (required)>'"},
		},
	}
	collector.AddNotification(failedTest)

	// Test adding a pending test
	pendingTest := types.TestCaseNotification{
		Event:           types.TestPending,
		TestID:          "test-3",
		Description:     "should be pending",
		FullDescription: "Test should be pending",
		Location:        "spec/example_spec.rb:30",
		FilePath:        "spec/example_spec.rb",
		LineNumber:      30,
		PendingMessage:  "Not yet implemented",
	}
	collector.AddNotification(pendingTest)

	// Test adding suite finished notification
	suiteFinished := types.SuiteNotification{
		Event:        types.SuiteFinished,
		TestCount:    3,
		FailureCount: 1,
		PendingCount: 1,
		LoadTime:     200 * time.Millisecond,
		Duration:     1 * time.Second,
	}
	collector.AddNotification(suiteFinished)

	// Test adding raw output
	rawOutput := types.OutputNotification{
		Event:   types.RawOutput,
		Content: "Some test output",
	}
	collector.AddNotification(rawOutput)

	// Verify collector state
	assert.Len(t, collector.tests, 3)
	assert.Len(t, collector.failures, 1)
	assert.Len(t, collector.pending, 1)
	assert.NotNil(t, collector.suiteInfo)
	assert.Equal(t, "Some test output\n", collector.rawOutput.String())
}

func TestTestCollector_BuildResult(t *testing.T) {
	collector := NewTestCollector()

	testFile := &TestFile{
		Path:     "spec/example_spec.rb",
		Filename: "example_spec.rb",
	}

	// Add a passing test
	collector.AddNotification(types.TestCaseNotification{
		Event:           types.TestPassed,
		TestID:          "test-1",
		FullDescription: "Example passes",
		LineNumber:      10,
	})

	// Add a failing test
	collector.AddNotification(types.TestCaseNotification{
		Event:           types.TestFailed,
		TestID:          "test-2",
		FullDescription: "Example fails",
		LineNumber:      20,
		Exception: &types.TestException{
			Message:   "Expected true to be false",
			Backtrace: []string{"spec/example_spec.rb:20"},
		},
	})

	// Add suite info
	collector.AddNotification(types.SuiteNotification{
		Event:        types.SuiteFinished,
		LoadTime:     100 * time.Millisecond,
		TestCount:    2,
		FailureCount: 1,
		PendingCount: 0,
	})

	// Add some output
	collector.AddNotification(types.OutputNotification{
		Event:   types.RawOutput,
		Content: "Test output line 1",
	})
	collector.AddNotification(types.OutputNotification{
		Event:   types.RawOutput,
		Content: "Test output line 2",
	})

	// Build result
	duration := 500 * time.Millisecond
	result := collector.BuildResult(testFile, duration)

	// Verify result
	require.Equal(t, testFile, result.File)
	assert.Equal(t, duration, result.Duration)
	assert.Equal(t, 2, result.ExampleCount)
	assert.Equal(t, 1, result.FailureCount)
	assert.Equal(t, types.StateFailed, result.State)
	assert.Equal(t, 100*time.Millisecond, result.FileLoadTime)
	assert.Equal(t, "Test output line 1\nTest output line 2\n", result.Output)

	// Verify failures through Tests array
	var failures []types.TestCaseNotification
	for _, test := range result.Tests {
		if test.Event == types.TestFailed {
			failures = append(failures, test)
		}
	}
	require.Len(t, failures, 1)
	failure := failures[0]
	assert.Equal(t, "Example fails", failure.FullDescription)
	assert.Equal(t, 20, failure.LineNumber)
	assert.Equal(t, "Expected true to be false", failure.Exception.Message)
	assert.Equal(t, []string{"spec/example_spec.rb:20"}, failure.Exception.Backtrace)
}

func TestTestCollector_BuildResult_Success(t *testing.T) {
	collector := NewTestCollector()

	testFile := &TestFile{
		Path:     "spec/example_spec.rb",
		Filename: "example_spec.rb",
	}

	// Add only passing tests
	collector.AddNotification(types.TestCaseNotification{
		Event:  types.TestPassed,
		TestID: "test-1",
	})
	collector.AddNotification(types.TestCaseNotification{
		Event:  types.TestPassed,
		TestID: "test-2",
	})

	result := collector.BuildResult(testFile, 100*time.Millisecond)

	assert.Equal(t, types.StateSuccess, result.State)
	assert.Equal(t, 2, result.ExampleCount)
	assert.Equal(t, 0, result.FailureCount)
	// Verify no failures in Tests array
	var failureCount int
	for _, test := range result.Tests {
		if test.Event == types.TestFailed {
			failureCount++
		}
	}
	assert.Equal(t, 0, failureCount)
}

func TestTestCollector_SuiteStartedPreservesLoadTime(t *testing.T) {
	collector := NewTestCollector()

	testFile := &TestFile{
		Path:     "spec/example_spec.rb",
		Filename: "example_spec.rb",
	}

	// First, add SuiteStarted with LoadTime (this comes from RSpec's start notification)
	collector.AddNotification(types.SuiteNotification{
		Event:     types.SuiteStarted,
		TestCount: 5,
		LoadTime:  1500 * time.Millisecond, // 1.5 seconds load time
	})

	// Add some test notifications
	collector.AddNotification(types.TestCaseNotification{
		Event:  types.TestPassed,
		TestID: "test-1",
	})
	collector.AddNotification(types.TestCaseNotification{
		Event:  types.TestPassed,
		TestID: "test-2",
	})

	// Then add SuiteFinished (which typically doesn't have LoadTime in the same way)
	collector.AddNotification(types.SuiteNotification{
		Event:        types.SuiteFinished,
		TestCount:    2,
		FailureCount: 0,
		PendingCount: 0,
		Duration:     2 * time.Second,
		// Note: No LoadTime here, or a different value
	})

	// Build result
	result := collector.BuildResult(testFile, 2*time.Second)

	// Verify that LoadTime from SuiteStarted is preserved
	assert.Equal(t, 1500*time.Millisecond, result.FileLoadTime, "LoadTime from SuiteStarted should be preserved")
	assert.Equal(t, types.StateSuccess, result.State)
	assert.Equal(t, 2, result.ExampleCount)
}

func TestTestCollector_SuiteStartedAndFinishedBothHaveLoadTime(t *testing.T) {
	collector := NewTestCollector()

	testFile := &TestFile{
		Path:     "spec/example_spec.rb",
		Filename: "example_spec.rb",
	}

	// Add SuiteStarted with LoadTime
	collector.AddNotification(types.SuiteNotification{
		Event:     types.SuiteStarted,
		TestCount: 3,
		LoadTime:  800 * time.Millisecond,
	})

	// Add SuiteFinished with a different LoadTime (should preserve the one from Started)
	collector.AddNotification(types.SuiteNotification{
		Event:        types.SuiteFinished,
		TestCount:    3,
		FailureCount: 0,
		LoadTime:     500 * time.Millisecond, // Different value
	})

	result := collector.BuildResult(testFile, 1*time.Second)

	// Should preserve LoadTime from SuiteStarted, not SuiteFinished
	assert.Equal(t, 800*time.Millisecond, result.FileLoadTime, "LoadTime from SuiteStarted should take precedence")
}
