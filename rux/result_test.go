package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/rsanheim/rux/rspec"
	"github.com/stretchr/testify/assert"
)

func TestBuildTestSummary(t *testing.T) {
	// Create test results
	results := []TestResult{
		{
			SpecFile:     "spec/model_spec.rb",
			Success:      true,
			ExampleCount: 10,
			FailureCount: 0,
			Duration:     100 * time.Millisecond,
			Failures:     []rspec.FailureDetail{},
		},
		{
			SpecFile:     "spec/controller_spec.rb",
			Success:      false,
			ExampleCount: 5,
			FailureCount: 2,
			Duration:     200 * time.Millisecond,
			Failures: []rspec.FailureDetail{
				{
					Description: "Controller GET /index returns 200",
					Message:     "expected 200, got 404",
					Backtrace:   []string{"spec/controller_spec.rb:10"},
				},
				{
					Description: "Controller POST /create creates resource",
					Message:     "expected resource to be created",
					Backtrace:   []string{"spec/controller_spec.rb:20"},
				},
			},
		},
		{
			SpecFile:     "spec/broken_spec.rb",
			Success:      false,
			ExampleCount: 0,
			FailureCount: 0,
			Duration:     50 * time.Millisecond,
			Error:        fmt.Errorf("Failed to load spec file"),
		},
	}

	wallTime := 250 * time.Millisecond
	summary := BuildTestSummary(results, wallTime)

	assert := assert.New(t)
	assert.Equal(15, summary.TotalExamples)
	assert.Equal(2, summary.TotalFailures, "total failures")
	assert.Equal(2, len(summary.AllFailures), "failure details")

	assert.Equal(350*time.Millisecond, summary.TotalCPUTime, "total CPU time")
	assert.Equal(wallTime, summary.WallTime, "wall time")

	assert.True(summary.HasFailures, "should have failures")
	assert.False(summary.Success, "should not be successful when there are failures")

	assert.Len(summary.ErroredFiles, 1, "errored files")
	assert.Equal("spec/broken_spec.rb", summary.ErroredFiles[0].SpecFile, "errored file name")
}

// Test that summary correctly identifies when there are no failures
func TestBuildTestSummaryNoFailures(t *testing.T) {
	results := []TestResult{
		{
			SpecFile:     "spec/model_spec.rb",
			Success:      true,
			ExampleCount: 10,
			FailureCount: 0,
			Duration:     100 * time.Millisecond,
		},
		{
			SpecFile:     "spec/controller_spec.rb",
			Success:      true,
			ExampleCount: 5,
			FailureCount: 0,
			Duration:     200 * time.Millisecond,
		},
	}

	summary := BuildTestSummary(results, 250*time.Millisecond)

	assert.Equal(t, 15, summary.TotalExamples)
	assert.Equal(t, 0, summary.TotalFailures)
	assert.False(t, summary.HasFailures, "should have no failures when all tests pass")
	assert.True(t, summary.Success, "should be successful when all tests pass")
	assert.Empty(t, summary.AllFailures, "should have no failures")
	assert.Empty(t, summary.ErroredFiles, "should have no errored files")
	assert.Equal(t, "", summary.FormattedSummary, "summary with multiple results should be empty")
}

func TestSingleTestResultIsSingleWorkerMode(t *testing.T) {
	results := []TestResult{
		{
			SpecFile:         "spec/model_spec.rb",
			Success:          true,
			ExampleCount:     10,
			FailureCount:     0,
			Duration:         100 * time.Millisecond,
			FormattedSummary: "10 examples, 0 failures",
		},
	}

	summary := BuildTestSummary(results, 100*time.Millisecond)

	assert.Equal(t, 10, summary.TotalExamples)
	assert.True(t, summary.Success)
	assert.Equal(t, summary.FormattedSummary, "10 examples, 0 failures")
}
