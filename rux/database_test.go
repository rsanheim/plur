package main

import (
	"fmt"
	"testing"
)

func TestRunDatabaseTaskDryRun(t *testing.T) {
	// Test that dry-run shows the correct commands
	err := RunDatabaseTask("db:test", 3, true)
	if err != nil {
		t.Errorf("RunDatabaseTask dry-run should not error: %v", err)
	}

	// This test just verifies the function doesn't crash
	// In a real test we'd capture stdout to verify the output
}

func TestRunDatabaseTaskValidation(t *testing.T) {
	tests := []struct {
		task        string
		workerCount int
		dryRun      bool
		shouldError bool
	}{
		{"db:create", 1, true, false},
		{"db:migrate", 2, true, false},
		{"db:test:prepare", 3, true, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%d_workers", tt.task, tt.workerCount), func(t *testing.T) {
			err := RunDatabaseTask(tt.task, tt.workerCount, tt.dryRun)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for task %s with %d workers", tt.task, tt.workerCount)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for task %s with %d workers: %v", tt.task, tt.workerCount, err)
			}
		})
	}
}
