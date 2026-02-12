package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFilesFromJob(t *testing.T) {
	// Test the job-based file discovery
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	require.NoError(t, os.MkdirAll("tmp", 0o755), "Failed to create tmp dir")

	tmpBase, err := filepath.Abs("tmp")
	require.NoError(t, err, "Failed to resolve tmp dir")

	tempDir, err := os.MkdirTemp(tmpBase, "test-runner-specs-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	os.Chdir(tempDir)

	// Test empty directory
	rspecJob := job.Job{
		Name:          "rspec",
		TargetPattern: "spec/**/*_spec.rb",
	}
	files, err := FindFilesFromJob(rspecJob)
	assert.NoError(t, err, "FindFilesFromJob() should not return error")
	assert.Empty(t, files, "FindFilesFromJob() should return empty slice for empty directory")

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

	files, err = FindFilesFromJob(rspecJob)
	assert.NoError(t, err, "FindFilesFromJob() should not return error")

	expectedFiles := 5 // Only *_spec.rb files
	assert.Len(t, files, expectedFiles, "FindFilesFromJob() should find exactly 5 spec files")

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

func TestExpandPatternsFromJobUsesFrameworkDetectPatterns(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.MkdirAll("tmp", 0755)
	tempDir, err := os.MkdirTemp("tmp", "test-runner-specs-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	os.Chdir(tempDir)

	os.MkdirAll("spec/models", 0755)
	os.WriteFile("spec/models/user_spec.rb", []byte(""), 0o644)
	os.WriteFile("spec/models/post_spec.rb", []byte(""), 0o644)
	os.WriteFile("spec/models/readme.txt", []byte(""), 0o644)

	rspecJob := job.Job{
		Name:      "fast",
		Framework: "rspec",
	}

	files, err := ExpandPatternsFromJob([]string{"spec/models"}, rspecJob)
	require.NoError(t, err)

	expected := map[string]bool{
		"spec/models/user_spec.rb": false,
		"spec/models/post_spec.rb": false,
	}

	for _, file := range files {
		if _, ok := expected[file]; ok {
			expected[file] = true
		} else {
			assert.Fail(t, "Unexpected file found: %s", file)
		}
	}

	for file, found := range expected {
		assert.True(t, found, "Expected spec file not found: %s", file)
	}
}
