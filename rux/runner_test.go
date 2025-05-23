package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestFindSpecFilesRunner(t *testing.T) {
	// Test the runner version more thoroughly
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create temp directory in rux/tmp/
	os.MkdirAll("tmp", 0755)
	tempDir, err := os.MkdirTemp("tmp", "test-runner-specs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.Chdir(tempDir)

	// Test empty directory
	files, err := FindSpecFiles()
	if err != nil {
		t.Errorf("FindSpecFiles() returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("FindSpecFiles() returned %d files, expected 0", len(files))
	}

	// Create complex directory structure
	os.MkdirAll("spec/models", 0755)
	os.MkdirAll("spec/controllers", 0755)
	os.MkdirAll("spec/lib/utils", 0755)

	specFiles := []string{
		"spec/user_spec.rb",
		"spec/models/post_spec.rb",
		"spec/models/comment_spec.rb",
		"spec/controllers/users_controller_spec.rb",
		"spec/lib/utils/helper_spec.rb",
		"spec/not_a_spec.rb.bak", // Should be ignored
		"spec/README.md",         // Should be ignored
	}

	for _, file := range specFiles {
		f, _ := os.Create(file)
		f.Close()
	}

	files, err = FindSpecFiles()
	if err != nil {
		t.Errorf("FindSpecFiles() returned error: %v", err)
	}

	expectedFiles := 5 // Only *_spec.rb files
	if len(files) != expectedFiles {
		t.Errorf("FindSpecFiles() found %d files, expected %d", len(files), expectedFiles)
	}

	// Verify all expected spec files were found
	expectedSpecs := map[string]bool{
		"spec/user_spec.rb":                         false,
		"spec/models/post_spec.rb":                  false,
		"spec/models/comment_spec.rb":               false,
		"spec/controllers/users_controller_spec.rb": false,
		"spec/lib/utils/helper_spec.rb":             false,
	}

	for _, file := range files {
		if _, exists := expectedSpecs[file]; exists {
			expectedSpecs[file] = true
		} else {
			t.Errorf("Unexpected spec file found: %s", file)
		}
	}

	for specFile, found := range expectedSpecs {
		if !found {
			t.Errorf("Expected spec file not found: %s", specFile)
		}
	}
}

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

func TestGetWorkerCountEdgeCases(t *testing.T) {
	originalEnv := os.Getenv("PARALLEL_TEST_PROCESSORS")
	defer os.Setenv("PARALLEL_TEST_PROCESSORS", originalEnv)

	tests := []struct {
		name       string
		cliWorkers int
		envVar     string
		expected   int
	}{
		{
			name:       "Very high CLI workers",
			cliWorkers: 100,
			envVar:     "4",
			expected:   100,
		},
		{
			name:       "Zero env var",
			cliWorkers: 0,
			envVar:     "0",
			expected:   max(1, runtime.NumCPU()-2),
		},
		{
			name:       "Negative env var",
			cliWorkers: 0,
			envVar:     "-5",
			expected:   max(1, runtime.NumCPU()-2),
		},
		{
			name:       "Empty env var",
			cliWorkers: 0,
			envVar:     "",
			expected:   max(1, runtime.NumCPU()-2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("PARALLEL_TEST_PROCESSORS", tt.envVar)
			} else {
				os.Unsetenv("PARALLEL_TEST_PROCESSORS")
			}

			result := GetWorkerCount(tt.cliWorkers)
			if result != tt.expected {
				t.Errorf("GetWorkerCount(%d) = %d, expected %d", tt.cliWorkers, result, tt.expected)
			}
		})
	}
}

func TestGetTestEnvNumber(t *testing.T) {
	tests := []struct {
		workerIndex int
		expected    string
	}{
		{0, ""},     // First worker gets empty string
		{1, "2"},    // Second worker gets "2"
		{2, "3"},    // Third worker gets "3"
		{3, "4"},    // Fourth worker gets "4"
		{10, "11"},  // Nth worker gets "N+1"
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("worker_%d", tt.workerIndex), func(t *testing.T) {
			result := GetTestEnvNumber(tt.workerIndex)
			if result != tt.expected {
				t.Errorf("GetTestEnvNumber(%d) = %q, expected %q", tt.workerIndex, result, tt.expected)
			}
		})
	}
}
