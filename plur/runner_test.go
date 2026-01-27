package main

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	t.Run("default behavior (first-is-1)", func(t *testing.T) {
		config := &config.GlobalConfig{
			WorkerCount: 4,
			FirstIs1:    true,
		}

		tests := []struct {
			workerIndex int
			expected    string
		}{
			{0, "1"}, // First worker gets "1"
			{1, "2"}, // Second worker gets "2"
			{2, "3"}, // Third worker gets "3"
			{3, "4"}, // Fourth worker gets "4"
		}

		for _, tt := range tests {
			result := GetTestEnvNumber(tt.workerIndex, config)
			assert.Equal(t, tt.expected, result, "GetTestEnvNumber(%d) with first-is-1", tt.workerIndex)
		}
	})

	t.Run("legacy behavior (no-first-is-1)", func(t *testing.T) {
		config := &config.GlobalConfig{
			WorkerCount: 4,
			FirstIs1:    false,
		}

		tests := []struct {
			workerIndex int
			expected    string
		}{
			{0, ""},  // First worker gets empty string
			{1, "2"}, // Second worker gets "2"
			{2, "3"}, // Third worker gets "3"
			{3, "4"}, // Fourth worker gets "4"
		}

		for _, tt := range tests {
			result := GetTestEnvNumber(tt.workerIndex, config)
			assert.Equal(t, tt.expected, result, "GetTestEnvNumber(%d) with legacy behavior", tt.workerIndex)
		}
	})
}

func TestDryRunString(t *testing.T) {
	t.Run("includes only plur env vars in output", func(t *testing.T) {
		cmd := exec.Command("bundle", "exec", "rspec", "spec/foo_spec.rb")
		cmd.Env = []string{
			"PATH=/usr/bin",
			"HOME=/home/user",
			"PARALLEL_TEST_GROUPS=4",
			"TEST_ENV_NUMBER=2",
			"RAILS_ENV=test",
		}

		result := dryRunString(cmd)

		assert.Contains(t, result, "PARALLEL_TEST_GROUPS=4")
		assert.Contains(t, result, "TEST_ENV_NUMBER=2")
		assert.Contains(t, result, "RAILS_ENV=test")
		assert.Contains(t, result, "bundle exec rspec spec/foo_spec.rb")
		// Should NOT include other env vars
		assert.NotContains(t, result, "PATH=")
		assert.NotContains(t, result, "HOME=")
	})

	t.Run("serial mode only has PARALLEL_TEST_GROUPS", func(t *testing.T) {
		cmd := exec.Command("bundle", "exec", "rspec", "spec/foo_spec.rb")
		cmd.Env = []string{
			"PARALLEL_TEST_GROUPS=1",
		}

		result := dryRunString(cmd)

		assert.Contains(t, result, "PARALLEL_TEST_GROUPS=1")
		assert.NotContains(t, result, "TEST_ENV_NUMBER")
		assert.Contains(t, result, "bundle exec rspec")
	})

	t.Run("no plur env vars returns just command", func(t *testing.T) {
		cmd := exec.Command("echo", "hello")
		cmd.Env = []string{
			"PATH=/usr/bin",
			"HOME=/home/user",
		}

		result := dryRunString(cmd)

		assert.Equal(t, "echo hello", result)
	})

	t.Run("empty env returns just command", func(t *testing.T) {
		cmd := exec.Command("ls", "-la")
		cmd.Env = []string{}

		result := dryRunString(cmd)

		assert.Equal(t, "ls -la", result)
	})

	t.Run("nil env returns just command", func(t *testing.T) {
		cmd := exec.Command("pwd")
		// cmd.Env is nil by default

		result := dryRunString(cmd)

		assert.Equal(t, "pwd", result)
	})

	t.Run("env vars appear before command", func(t *testing.T) {
		cmd := exec.Command("bundle", "exec", "rspec")
		cmd.Env = []string{
			"TEST_ENV_NUMBER=1",
			"PARALLEL_TEST_GROUPS=2",
		}

		result := dryRunString(cmd)

		// order is not guaranteed
		assert.True(t,
			result == "TEST_ENV_NUMBER=1 PARALLEL_TEST_GROUPS=2 bundle exec rspec" ||
				result == "PARALLEL_TEST_GROUPS=2 TEST_ENV_NUMBER=1 bundle exec rspec",
			"expected env vars before command, got: %s", result)
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("parallel mode includes TEST_ENV_NUMBER", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 4,
			FirstIs1:    true,
		}
		runner := &Runner{config: cfg}

		env := runner.buildEnv(0, 4)

		// Should contain our env vars
		assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=4")
		assertEnvContains(t, env, "TEST_ENV_NUMBER=1")
	})

	t.Run("serial mode excludes TEST_ENV_NUMBER", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 1, // Serial mode
			FirstIs1:    true,
		}
		runner := &Runner{config: cfg}

		env := runner.buildEnv(0, 1)

		assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=1")
		assertEnvNotContains(t, env, "TEST_ENV_NUMBER=")
	})

	t.Run("worker indices map correctly with FirstIs1", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 3,
			FirstIs1:    true,
		}
		runner := &Runner{config: cfg}

		worker0 := runner.buildEnv(0, 3)
		worker1 := runner.buildEnv(1, 3)
		worker2 := runner.buildEnv(2, 3)

		assertEnvContains(t, worker0, "TEST_ENV_NUMBER=1")
		assertEnvContains(t, worker1, "TEST_ENV_NUMBER=2")
		assertEnvContains(t, worker2, "TEST_ENV_NUMBER=3")
	})

	t.Run("first-is-1=false gives empty string for first worker", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 3,
			FirstIs1:    false,
		}
		runner := &Runner{config: cfg}

		worker0 := runner.buildEnv(0, 3)
		worker1 := runner.buildEnv(1, 3)

		// First worker gets empty string when first-is-1 is false
		assertEnvContains(t, worker0, "TEST_ENV_NUMBER=")
		assertEnvContains(t, worker1, "TEST_ENV_NUMBER=2")
	})

	t.Run("inherits system environment", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 2,
			FirstIs1:    true,
		}
		runner := &Runner{config: cfg}

		env := runner.buildEnv(0, 2)

		// Should have more than just our two vars (inherited from os.Environ())
		assert.Greater(t, len(env), 2, "should inherit system environment")
	})
}

func TestRunner_DryRunReturnsNil(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 2,
		DryRun:      true,
		FirstIs1:    true,
		ColorOutput: false,
		RuntimeDir:  t.TempDir(),
	}
	// Use a custom job to avoid needing ConfigPaths for RSpec formatter
	testJob := job.Job{
		Name:          "custom",
		Cmd:           []string{"echo", "test"},
		TargetPattern: "**/*_test.rb",
	}

	runner, err := NewRunner(cfg, []string{"a_test.rb", "b_test.rb"}, testJob)
	require.NoError(t, err)
	results, wallTime, err := runner.Run()

	assert.Nil(t, err, "dry-run should not error")
	assert.Nil(t, results, "dry-run should return nil results")
	assert.Equal(t, int64(0), wallTime.Nanoseconds(), "dry-run should return zero wall time")
}

func TestRunner_WorkerCountAdjustment(t *testing.T) {
	t.Run("more workers than files reduces to file count", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 10, // Way more than files
			DryRun:      true,
			FirstIs1:    true,
			RuntimeDir:  t.TempDir(),
		}
		testJob := job.Job{
			Name:          "custom",
			Cmd:           []string{"echo"},
			TargetPattern: "**/*_test.rb",
		}

		files := []string{"a_test.rb", "b_test.rb"} // Only 2 files
		runner, err := NewRunner(cfg, files, testJob)
		require.NoError(t, err)
		groups := runner.groupFiles()
		assert.Equal(t, 2, len(groups), "should have 2 groups")

		// Run should work without error
		_, _, err = runner.Run()
		assert.NoError(t, err)
	})

	t.Run("single file uses single worker", func(t *testing.T) {
		cfg := &config.GlobalConfig{
			WorkerCount: 4,
			DryRun:      true,
			FirstIs1:    true,
			RuntimeDir:  t.TempDir(),
		}
		testJob := job.Job{
			Name:          "custom",
			Cmd:           []string{"echo"},
			TargetPattern: "**/*_test.rb",
		}

		files := []string{"only_test.rb"}
		runner, err := NewRunner(cfg, files, testJob)
		require.NoError(t, err)

		_, _, err = runner.Run()
		assert.NoError(t, err)
	})
}

func TestRunner_EmptyFiles(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 4,
		DryRun:      true,
		FirstIs1:    true,
		RuntimeDir:  t.TempDir(),
	}
	testJob := job.Job{
		Name:          "custom",
		Cmd:           []string{"echo"},
		TargetPattern: "**/*_test.rb",
	}

	runner, err := NewRunner(cfg, []string{}, testJob)
	require.NoError(t, err)
	results, wallTime, err := runner.Run()

	// Empty files should still work (no workers spawned)
	assert.NoError(t, err)
	assert.Nil(t, results)
	assert.Equal(t, int64(0), wallTime.Nanoseconds())
}

func TestRunner_TrackerInitialized(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 2,
		DryRun:      true,
		FirstIs1:    true,
		RuntimeDir:  t.TempDir(),
	}
	testJob := job.Job{Name: "custom"}

	runner, err := NewRunner(cfg, []string{"a_test.rb"}, testJob)
	require.NoError(t, err)

	require.NotNil(t, runner.Tracker(), "tracker should be initialized")
}

// === Design Edge Cases ===
// These tests document current behavior and design decisions.

func TestRunner_SingleFileStillSetsTestEnvNumber(t *testing.T) {
	// DESIGN DECISION: When you request 4 workers but only have 1 file,
	// TEST_ENV_NUMBER is still set because config.IsSerial() checks
	// WorkerCount (4), not the actual number of groups created (1).
	//
	// This is arguably correct - the user requested parallel mode,
	// so databases etc. should be set up for parallel execution even
	// if this particular run only has 1 file.
	cfg := &config.GlobalConfig{
		WorkerCount: 4, // User requested 4 workers
		DryRun:      true,
		FirstIs1:    true,
		RuntimeDir:  t.TempDir(),
	}
	testJob := job.Job{
		Name: "custom",
		Cmd:  []string{"echo"},
	}

	runner, err := NewRunner(cfg, []string{"single_test.rb"}, testJob)
	require.NoError(t, err)
	_, _, err = runner.Run()

	assert.NoError(t, err)
	// Output will show PARALLEL_TEST_GROUPS=1 TEST_ENV_NUMBER=1
	// This is intentional - we're in parallel mode, just with 1 group
}

func TestRunner_SerialModeNoTestEnvNumber(t *testing.T) {
	// DESIGN DECISION: Only when WorkerCount=1 (true serial mode)
	// do we omit TEST_ENV_NUMBER. This lets the first database be
	// the "default" one without a suffix.
	cfg := &config.GlobalConfig{
		WorkerCount: 1, // Explicit serial mode
		DryRun:      true,
		FirstIs1:    true,
		RuntimeDir:  t.TempDir(),
	}
	testJob := job.Job{
		Name: "custom",
		Cmd:  []string{"echo"},
	}

	runner, err := NewRunner(cfg, []string{"a_test.rb", "b_test.rb", "c_test.rb"}, testJob)
	require.NoError(t, err)

	// In serial mode, buildEnv should NOT include TEST_ENV_NUMBER
	env := runner.buildEnv(0, 1)
	assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=1")
	assertEnvNotContains(t, env, "TEST_ENV_NUMBER=")
}

func TestRunner_GroupCountMatchesActualGroups(t *testing.T) {
	// PARALLEL_TEST_GROUPS reflects actual groups created, not requested workers
	cfg := &config.GlobalConfig{
		WorkerCount: 10, // Requested 10
		DryRun:      true,
		FirstIs1:    true,
		RuntimeDir:  t.TempDir(),
	}
	testJob := job.Job{
		Name: "custom",
		Cmd:  []string{"echo"},
	}

	files := []string{"a.rb", "b.rb", "c.rb"} // Only 3 files
	runner, err := NewRunner(cfg, files, testJob)
	require.NoError(t, err)

	// With 3 files and 10 workers requested, we should get 3 groups
	// Each env should show PARALLEL_TEST_GROUPS=3
	env := runner.buildEnv(0, 3)
	assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=3")
}

// Helper functions for env assertions
func assertEnvContains(t *testing.T, env []string, expected string) {
	t.Helper()
	for _, e := range env {
		if e == expected {
			return
		}
	}
	assert.Fail(t, "env should contain %q", expected)
}

func assertEnvNotContains(t *testing.T, env []string, prefix string) {
	t.Helper()
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			assert.Fail(t, "env should not contain %q but found %q", prefix, e)
		}
	}
}
