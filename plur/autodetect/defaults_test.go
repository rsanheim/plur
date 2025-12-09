package autodetect

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinDefaultsLoad(t *testing.T) {
	// Verify that builtinDefaults were loaded in init()
	assert.NotEmpty(t, builtinDefaults.Defaults.Jobs)
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "rspec")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "minitest")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "go-test")
}

func TestResolveJobExplicitUserJob(t *testing.T) {
	userJobs := map[string]job.Job{
		"custom": {Cmd: []string{"custom-runner"}},
	}

	result, err := ResolveJob("custom", userJobs, nil)
	require.NoError(t, err)
	assert.Equal(t, "custom", result.Name)
	assert.Equal(t, []string{"custom-runner"}, result.Job.Cmd)
}

func TestResolveJobExplicitBuiltinJob(t *testing.T) {
	result, err := ResolveJob("rspec", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", result.Name)
	assert.Contains(t, result.Job.Cmd, "rspec")
}

func TestResolveJobExplicitNotFound(t *testing.T) {
	_, err := ResolveJob("nonexistent", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "job 'nonexistent' not found")
}

func TestResolveJobInferFromPatterns(t *testing.T) {
	// Create temp directory with a spec file
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create the spec file so it exists
	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/user_spec.rb", []byte(""), 0o644)

	result, err := ResolveJob("", nil, []string{"spec/user_spec.rb"})
	require.NoError(t, err)
	assert.Equal(t, "rspec", result.Name)
	assert.True(t, result.WasInferred)
}

func TestResolveJobInferMinitest(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	os.MkdirAll("test", 0o755)
	os.WriteFile("test/user_test.rb", []byte(""), 0o644)

	result, err := ResolveJob("", nil, []string{"test/user_test.rb"})
	require.NoError(t, err)
	assert.Equal(t, "minitest", result.Name)
	assert.True(t, result.WasInferred)
}

func TestResolveJobAutodetectRSpec(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create spec directory with a spec file
	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/example_spec.rb", []byte(""), 0o644)

	result, err := ResolveJob("", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", result.Name)
}

func TestResolveJobAutodetectMinitest(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create test directory with a test file (no spec dir)
	os.MkdirAll("test", 0o755)
	os.WriteFile("test/example_test.rb", []byte(""), 0o644)

	result, err := ResolveJob("", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "minitest", result.Name)
}

func TestResolveJobAutodetectGoTest(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Create go.mod and a test file
	os.WriteFile("go.mod", []byte("module test\n"), 0o644)
	os.WriteFile("example_test.go", []byte("package main\n"), 0o644)

	result, err := ResolveJob("", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "go-test", result.Name)
}

func TestResolveJobAutodetectPriority(t *testing.T) {
	// RSpec takes priority over minitest when both exist
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/example_spec.rb", []byte(""), 0o644)
	os.MkdirAll("test", 0o755)
	os.WriteFile("test/example_test.rb", []byte(""), 0o644)

	result, err := ResolveJob("", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", result.Name) // rspec has priority
}

func TestResolveJobAutodetectNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Empty directory - no test files
	_, err := ResolveJob("", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No default spec/test files found using default patterns")
}

func TestResolveJobReturnsWatches(t *testing.T) {
	result, err := ResolveJob("rspec", nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Watches)

	// Should have lib-to-spec watch
	var found bool
	for _, w := range result.Watches {
		if w.Name == "lib-to-spec" {
			found = true
			assert.Equal(t, "lib/**/*.rb", w.Source)
			break
		}
	}
	assert.True(t, found, "expected lib-to-spec watch for rspec job")
}

func TestDefaultJobCommands(t *testing.T) {
	tests := []struct {
		jobName string
		wantCmd []string
	}{
		{
			jobName: "rspec",
			wantCmd: []string{"bundle", "exec", "rspec", "{{target}}"},
		},
		{
			jobName: "minitest",
			wantCmd: []string{"bundle", "exec", "ruby", "-Itest", "{{target}}"},
		},
		{
			jobName: "go-test",
			wantCmd: []string{"go", "test", "{{target}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			result, err := ResolveJob(tt.jobName, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCmd, result.Job.Cmd)
		})
	}
}

func TestGetWatchesForJob(t *testing.T) {
	watches := getWatchesForJob("rspec")
	assert.NotEmpty(t, watches)

	// All watches should reference rspec
	for _, w := range watches {
		assert.Contains(t, w.Jobs, "rspec")
	}
}

func TestGetWatchesForJobNoMatches(t *testing.T) {
	watches := getWatchesForJob("nonexistent")
	assert.Empty(t, watches)
}
