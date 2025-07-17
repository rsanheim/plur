package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunDatabaseTaskDryRun(t *testing.T) {
	// Test that dry-run shows the correct commands
	err := RunDatabaseTask("db:test", 3, true)
	assert.NoError(t, err, "RunDatabaseTask dry-run should not error")

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

			if tt.shouldError {
				assert.Error(t, err, "Expected error for task %s with %d workers", tt.task, tt.workerCount)
			} else {
				assert.NoError(t, err, "Unexpected error for task %s with %d workers", tt.task, tt.workerCount)
			}
		})
	}
}
