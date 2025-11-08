package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinDefaultsLoad(t *testing.T) {
	// Verify that builtinDefaults were loaded in init()
	assert.NotEmpty(t, builtinDefaults.Defaults)
	assert.Contains(t, builtinDefaults.Defaults, "ruby")
	assert.Contains(t, builtinDefaults.Defaults, "go")
}

func TestGetDefaultProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		wantJobs    []string
		wantWatches int
	}{
		{
			name:        "ruby profile",
			profileName: "ruby",
			wantJobs:    []string{"rspec", "minitest", "rubocop"},
			wantWatches: 6, // lib-to-spec, app-to-spec, spec-files, lib-to-test, app-to-test, test-files
		},
		{
			name:        "go profile",
			profileName: "go",
			wantJobs:    []string{"go-test", "go-lint"},
			wantWatches: 2, // go-source, go-tests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := GetDefaultProfile(tt.profileName)
			require.NotNil(t, profile)

			assert.Len(t, profile.Jobs, len(tt.wantJobs))
			for _, jobName := range tt.wantJobs {
				assert.Contains(t, profile.Jobs, jobName)
			}

			assert.Len(t, profile.Watches, tt.wantWatches)
		})
	}
}

func TestGetDefaultProfileNonexistent(t *testing.T) {
	profile := GetDefaultProfile("nonexistent")
	assert.Nil(t, profile)
}

func TestAutodetectProfileGo(t *testing.T) {
	// Create temp directory with go.mod
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	err = os.WriteFile("go.mod", []byte("module test\n"), 0o644)
	require.NoError(t, err)

	profile := AutodetectProfile()
	assert.Equal(t, "go", profile)
}

func TestAutodetectProfileRubyWithRSpec(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	err = os.WriteFile("Gemfile", []byte("source 'https://rubygems.org'\n"), 0o644)
	require.NoError(t, err)

	err = os.Mkdir("spec", 0o755)
	require.NoError(t, err)

	profile := AutodetectProfile()
	assert.Equal(t, "ruby", profile)
}

func TestAutodetectProfileRubyWithMinitest(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	err = os.WriteFile("Gemfile", []byte("source 'https://rubygems.org'\n"), 0o644)
	require.NoError(t, err)

	err = os.Mkdir("test", 0o755)
	require.NoError(t, err)

	profile := AutodetectProfile()
	assert.Equal(t, "ruby", profile)
}

func TestAutodetectProfileRubyLibOnly(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	err = os.Mkdir("lib", 0o755)
	require.NoError(t, err)

	err = os.Mkdir("spec", 0o755)
	require.NoError(t, err)

	profile := AutodetectProfile()
	assert.Equal(t, "ruby", profile)
}

func TestAutodetectProfileNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Empty directory
	profile := AutodetectProfile()
	assert.Equal(t, "", profile)
}

func TestGetAutodetectedDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a Ruby project
	err = os.WriteFile("Gemfile", []byte("source 'https://rubygems.org'\n"), 0o644)
	require.NoError(t, err)

	err = os.Mkdir("spec", 0o755)
	require.NoError(t, err)

	jobs, watches := GetAutodetectedDefaults()

	assert.NotEmpty(t, jobs)
	assert.Contains(t, jobs, "rspec")
	assert.Contains(t, jobs, "minitest")

	assert.NotEmpty(t, watches)

	// Verify job names are set
	for name, job := range jobs {
		assert.Equal(t, name, job.Name)
	}
}

func TestGetAutodetectedDefaultsNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	jobs, watches := GetAutodetectedDefaults()

	assert.Empty(t, jobs)
	assert.Empty(t, watches)
}

func TestRubyDefaultsConfiguration(t *testing.T) {
	profile := GetDefaultProfile("ruby")
	require.NotNil(t, profile)

	// Test RSpec job
	rspecJob, exists := profile.Jobs["rspec"]
	require.True(t, exists)
	assert.Contains(t, rspecJob.Cmd, "rspec")
	assert.Contains(t, rspecJob.Cmd, "{{target}}")

	// Test lib-to-spec watch
	var libToSpec *WatchMapping
	for i := range profile.Watches {
		if profile.Watches[i].Name == "lib-to-spec" {
			libToSpec = &profile.Watches[i]
			break
		}
	}
	require.NotNil(t, libToSpec)
	assert.Equal(t, "lib/**/*.rb", libToSpec.Source)
	assert.NotNil(t, libToSpec.Targets)
	assert.Contains(t, *libToSpec.Targets, "spec/{{match}}_spec.rb")
	assert.Contains(t, libToSpec.Jobs, "rspec")
}

func TestGoDefaultsConfiguration(t *testing.T) {
	profile := GetDefaultProfile("go")
	require.NotNil(t, profile)

	// Test go-test job
	goTestJob, exists := profile.Jobs["go-test"]
	require.True(t, exists)
	assert.Contains(t, goTestJob.Cmd, "go")
	assert.Contains(t, goTestJob.Cmd, "test")

	// Test go-source watch
	var goSource *WatchMapping
	for i := range profile.Watches {
		if profile.Watches[i].Name == "go-source" {
			goSource = &profile.Watches[i]
			break
		}
	}
	require.NotNil(t, goSource)
	assert.Equal(t, "**/*.go", goSource.Source)
	assert.NotNil(t, goSource.Targets)
	assert.Contains(t, *goSource.Targets, "{{dir_relative}}")
	assert.Contains(t, goSource.Jobs, "go-test")
	assert.Contains(t, goSource.Exclude, "vendor/**")
}

func TestDefaultJobCommands(t *testing.T) {
	tests := []struct {
		profile string
		job     string
		wantCmd []string
	}{
		{
			profile: "ruby",
			job:     "rspec",
			wantCmd: []string{"bundle", "exec", "rspec", "{{target}}"},
		},
		{
			profile: "ruby",
			job:     "minitest",
			wantCmd: []string{"bundle", "exec", "ruby", "-Itest", "{{target}}"},
		},
		{
			profile: "go",
			job:     "go-test",
			wantCmd: []string{"go", "test", "{{target}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.profile+"/"+tt.job, func(t *testing.T) {
			profile := GetDefaultProfile(tt.profile)
			require.NotNil(t, profile)

			job, exists := profile.Jobs[tt.job]
			require.True(t, exists)
			assert.Equal(t, tt.wantCmd, job.Cmd)
		})
	}
}

func TestFileAndDirHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create test file and directory
	err = os.WriteFile("test.txt", []byte("test"), 0o644)
	require.NoError(t, err)

	err = os.Mkdir("testdir", 0o755)
	require.NoError(t, err)

	// Test fileExists
	assert.True(t, fileExists("test.txt"))
	assert.False(t, fileExists("nonexistent.txt"))
	assert.False(t, fileExists("testdir")) // Directory should return false

	// Test dirExists
	assert.True(t, dirExists("testdir"))
	assert.False(t, dirExists("nonexistent"))
	assert.False(t, dirExists("test.txt")) // File should return false
}

func TestDefaultProfileCopy(t *testing.T) {
	// Get profile twice and verify they are independent copies
	profile1 := GetDefaultProfile("ruby")
	profile2 := GetDefaultProfile("ruby")

	require.NotNil(t, profile1)
	require.NotNil(t, profile2)

	// Modify profile1's jobs
	profile1.Jobs["rspec"] = Job{Name: "modified", Cmd: []string{"modified"}}

	// Verify profile2 is not affected
	assert.NotEqual(t, profile1.Jobs["rspec"].Cmd, profile2.Jobs["rspec"].Cmd)
}

// Integration test: verify defaults work with EventProcessor
func TestDefaultsWithEventProcessor(t *testing.T) {
	profile := GetDefaultProfile("ruby")
	require.NotNil(t, profile)

	// Convert to pointer maps as EventProcessor expects
	jobs := make(map[string]*Job)
	for name, job := range profile.Jobs {
		jobCopy := job
		jobCopy.Name = name
		jobs[name] = &jobCopy
	}

	watches := make([]*WatchMapping, len(profile.Watches))
	for i := range profile.Watches {
		watches[i] = &profile.Watches[i]
	}

	// Create processor
	processor := NewEventProcessor(jobs, watches)

	// Test lib file mapping
	result, err := processor.ProcessPath("lib/user.rb")
	require.NoError(t, err)

	assert.Contains(t, result, "rspec")
	assert.Contains(t, result["rspec"], filepath.FromSlash("spec/user_spec.rb"))
}
