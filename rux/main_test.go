package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestGetWorkerCount(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("PARALLEL_TEST_PROCESSORS")
	defer os.Setenv("PARALLEL_TEST_PROCESSORS", originalEnv)

	tests := []struct {
		name       string
		cliWorkers int
		envVar     string
		expected   int
	}{
		{
			name:       "CLI flag takes priority",
			cliWorkers: 4,
			envVar:     "8",
			expected:   4,
		},
		{
			name:       "Environment variable when no CLI flag",
			cliWorkers: 0,
			envVar:     "6",
			expected:   6,
		},
		{
			name:       "Default cores-2 when no CLI or env",
			cliWorkers: 0,
			envVar:     "",
			expected:   max(1, runtime.NumCPU()-2),
		},
		{
			name:       "Minimum 1 worker",
			cliWorkers: 0,
			envVar:     "",
			expected:   max(1, runtime.NumCPU()-2),
		},
		{
			name:       "Invalid env var falls back to default",
			cliWorkers: 0,
			envVar:     "invalid",
			expected:   max(1, runtime.NumCPU()-2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envVar != "" {
				os.Setenv("PARALLEL_TEST_PROCESSORS", tt.envVar)
			} else {
				os.Unsetenv("PARALLEL_TEST_PROCESSORS")
			}

			result := getWorkerCount(tt.cliWorkers)
			if result != tt.expected {
				t.Errorf("getWorkerCount(%d) = %d, expected %d", tt.cliWorkers, result, tt.expected)
			}
		})
	}
}

func TestFindSpecFiles(t *testing.T) {
	// Test when no spec directory exists
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create temp directory in rux/tmp/
	os.MkdirAll("tmp", 0755)
	tempDir, err := os.MkdirTemp("tmp", "test-specs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	os.Chdir(tempDir)

	files, err := findSpecFiles()
	if err != nil {
		t.Errorf("findSpecFiles() returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("findSpecFiles() returned %d files, expected 0", len(files))
	}

	// Create spec directory with test files
	os.Mkdir("spec", 0755)
	os.Mkdir("spec/models", 0755)
	
	// Create spec files
	specFiles := []string{
		"spec/user_spec.rb",
		"spec/models/post_spec.rb",
		"spec/not_a_spec.rb.txt", // Should be ignored
		"spec/regular_file.txt",   // Should be ignored
	}
	
	for _, file := range specFiles {
		f, _ := os.Create(file)
		f.Close()
	}

	files, err = findSpecFiles()
	if err != nil {
		t.Errorf("findSpecFiles() returned error: %v", err)
	}

	expectedFiles := 2 // Only user_spec.rb and post_spec.rb
	if len(files) != expectedFiles {
		t.Errorf("findSpecFiles() found %d files, expected %d", len(files), expectedFiles)
	}

	// Verify correct files were found
	foundUserSpec := false
	foundPostSpec := false
	for _, file := range files {
		if file == "spec/user_spec.rb" {
			foundUserSpec = true
		}
		if file == "spec/models/post_spec.rb" {
			foundPostSpec = true
		}
	}

	if !foundUserSpec {
		t.Error("Expected to find spec/user_spec.rb")
	}
	if !foundPostSpec {
		t.Error("Expected to find spec/models/post_spec.rb")
	}
}

func TestRuxHelp(t *testing.T) {
	// Run go run main.go --help
	cmd := exec.Command("go", "run", "main.go", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run rux --help: %v", err)
	}

	outputStr := string(output)
	
	// Check for expected help content
	expectedContent := []string{
		"rux",
		"USAGE",
		"COMMANDS",
		"GLOBAL OPTIONS",
		"--workers",
		"--dry-run",
		"--auto",
		"--help",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Help output missing expected content: %s", expected)
		}
	}
}

// Helper function for Go versions < 1.21
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}