package main

import (
	"testing"
	"time"

	"github.com/rsanheim/rux/types"
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
	assert.Len(t, collector.GetTests(), 3)
	assert.Len(t, collector.GetFailures(), 1)
	assert.Len(t, collector.GetPending(), 1)
	assert.NotNil(t, collector.GetSuiteInfo())
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
	assert.Equal(t, StateFailed, result.State)
	assert.Equal(t, 100*time.Millisecond, result.FileLoadTime)
	assert.Equal(t, "Test output line 1\nTest output line 2\n", result.Output)

	// Verify failures
	require.Len(t, result.Failures, 1)
	failure := result.Failures[0]
	assert.Equal(t, testFile, failure.File)
	assert.Equal(t, "Example fails", failure.Description)
	assert.Equal(t, 20, failure.LineNumber)
	assert.Equal(t, "Expected true to be false", failure.Message)
	assert.Equal(t, []string{"spec/example_spec.rb:20"}, failure.Backtrace)
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

	assert.Equal(t, StateSuccess, result.State)
	assert.Equal(t, 2, result.ExampleCount)
	assert.Equal(t, 0, result.FailureCount)
	assert.Len(t, result.Failures, 0)
}
