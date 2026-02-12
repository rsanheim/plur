package main

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFilesFromJob(t *testing.T) {
	// Save/restore original working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Use a writable temp dir (avoids CI checkout permission issues)
	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))

	// Test empty directory
	rspecJob := job.Job{
		Name:          "rspec",
		TargetPattern: "spec/**/*_spec.rb",
	}
	files, err := FindFilesFromJob(rspecJob)
	assert.NoError(t, err, "FindFilesFromJob() should not return error")
	assert.Empty(t, files, "FindFilesFromJob() should return empty slice for empty directory")

	// Create complex directory structure
	require.NoError(t, os.MkdirAll("spec/models", 0o755))
	require.NoError(t, os.MkdirAll("spec/controllers", 0o755))
	require.NoError(t, os.MkdirAll("spec/lib/utils", 0o755))

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
		require.NoError(t, os.MkdirAll(dirOf(file), 0o755))
		f, err := os.Create(file)
		require.NoError(t, err)
		require.NoError(t, f.Close())
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
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))

	require.NoError(t, os.MkdirAll("spec/models", 0o755))
	require.NoError(t, os.WriteFile("spec/models/user_spec.rb", []byte(""), 0o644))
	require.NoError(t, os.WriteFile("spec/models/post_spec.rb", []byte(""), 0o644))
	require.NoError(t, os.WriteFile("spec/models/readme.txt", []byte(""), 0o644))

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

// dirOf returns the directory portion of a relative path, or "." if none.
func dirOf(path string) string {
	// We avoid importing filepath just for this tiny helper.
	// Paths in these tests are always forward-slashed.
	lastSlash := -1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			lastSlash = i
		}
	}
	if lastSlash <= 0 {
		return "."
	}
	return path[:lastSlash]
}
