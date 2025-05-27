package main

import (
	"fmt"
	"testing"
	"time"
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
			Failures:     []FailureDetail{},
		},
		{
			SpecFile:     "spec/controller_spec.rb",
			Success:      false,
			ExampleCount: 5,
			FailureCount: 2,
			Duration:     200 * time.Millisecond,
			Failures: []FailureDetail{
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

	// Test the summary values
	if summary.TotalExamples != 15 {
		t.Errorf("Expected 15 total examples, got %d", summary.TotalExamples)
	}

	if summary.TotalFailures != 2 {
		t.Errorf("Expected 2 total failures, got %d", summary.TotalFailures)
	}

	if len(summary.AllFailures) != 2 {
		t.Errorf("Expected 2 failure details, got %d", len(summary.AllFailures))
	}

	if summary.TotalCPUTime != 350*time.Millisecond {
		t.Errorf("Expected 350ms total CPU time, got %v", summary.TotalCPUTime)
	}

	if summary.WallTime != wallTime {
		t.Errorf("Expected %v wall time, got %v", wallTime, summary.WallTime)
	}

	if !summary.HasFailures {
		t.Error("Expected HasFailures to be true")
	}

	if summary.Success {
		t.Error("Expected Success to be false when there are failures")
	}

	if len(summary.ErroredFiles) != 1 {
		t.Errorf("Expected 1 errored file, got %d", len(summary.ErroredFiles))
	}

	if summary.ErroredFiles[0].SpecFile != "spec/broken_spec.rb" {
		t.Errorf("Expected errored file to be spec/broken_spec.rb, got %s", summary.ErroredFiles[0].SpecFile)
	}
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

	if summary.HasFailures {
		t.Error("Expected HasFailures to be false when all tests pass")
	}

	if !summary.Success {
		t.Error("Expected Success to be true when all tests pass")
	}

	if len(summary.AllFailures) != 0 {
		t.Errorf("Expected no failures, got %d", len(summary.AllFailures))
	}

	if len(summary.ErroredFiles) != 0 {
		t.Errorf("Expected no errored files, got %d", len(summary.ErroredFiles))
	}
}
