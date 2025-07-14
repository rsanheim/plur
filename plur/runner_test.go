package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSpecFilesRunner(t *testing.T) {
	// Test the runner version more thoroughly
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create temp directory in rux/tmp/
	os.MkdirAll("tmp", 0755)
	tempDir, err := os.MkdirTemp("tmp", "test-runner-specs-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	os.Chdir(tempDir)

	// Test empty directory
	files, err := FindTestFiles(FrameworkRSpec)
	assert.NoError(t, err, "FindTestFiles() should not return error")
	assert.Empty(t, files, "FindTestFiles() should return empty slice for empty directory")

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

	files, err = FindTestFiles(FrameworkRSpec)
	assert.NoError(t, err, "FindTestFiles() should not return error")

	expectedFiles := 5 // Only *_spec.rb files
	assert.Len(t, files, expectedFiles, "FindSpecFiles() should find exactly 5 spec files")

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
			assert.Fail(t, "Unexpected spec file found: %s", file)
		}
	}

	for specFile, found := range expectedSpecs {
		assert.True(t, found, "Expected spec file not found: %s", specFile)
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
			assert.Equal(t, tt.expected, result, "GetWorkerCount(%d)", tt.cliWorkers)
		})
	}
}

func TestGetTestEnvNumber(t *testing.T) {
	tests := []struct {
		workerIndex int
		expected    string
	}{
		{0, ""},    // First worker gets empty string
		{1, "2"},   // Second worker gets "2"
		{2, "3"},   // Third worker gets "3"
		{3, "4"},   // Fourth worker gets "4"
		{10, "11"}, // Nth worker gets "N+1"
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("worker_%d", tt.workerIndex), func(t *testing.T) {
			result := GetTestEnvNumber(tt.workerIndex)
			assert.Equal(t, tt.expected, result, "GetTestEnvNumber(%d)", tt.workerIndex)
		})
	}
}
