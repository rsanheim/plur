package watch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobRunCommand(t *testing.T) {
	cwd := t.TempDir()
	run := JobRun{
		Job: framework.Job{
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec"},
			Env:  []string{"RAILS_ENV=test"},
		},
		Targets: []string{"spec/user_spec.rb", "spec/post_spec.rb"},
	}

	cmd := run.Command(cwd)

	assert.Equal(t, []string{"bundle", "exec", "rspec", "spec/user_spec.rb", "spec/post_spec.rb"}, cmd.Args)
	assert.Equal(t, cwd, cmd.Dir)
	assert.Contains(t, cmd.Env, "RAILS_ENV=test")
	assert.GreaterOrEqual(t, len(cmd.Env), len(os.Environ()), "inherited environment is preserved")
}

func TestJobRunCommand_NoTargets(t *testing.T) {
	run := JobRun{Job: framework.Job{Name: "build", Cmd: []string{"bin/rake", "install"}}}

	cmd := run.Command(t.TempDir())

	assert.Equal(t, []string{"bin/rake", "install"}, cmd.Args)
}

func TestJobRunCommand_JobEnvOverridesInherited(t *testing.T) {
	t.Setenv("PLUR_TEST_VAR", "from-environ")
	run := JobRun{
		Job: framework.Job{
			Name: "rspec",
			Cmd:  []string{"rspec"},
			Env:  []string{"PLUR_TEST_VAR=from-job"},
		},
	}

	cmd := run.Command(t.TempDir())

	env := cmd.Environ()
	assert.Contains(t, env, "PLUR_TEST_VAR=from-job")
	assert.NotContains(t, env, "PLUR_TEST_VAR=from-environ")
}

func TestCommandString(t *testing.T) {
	run := JobRun{
		Job: framework.Job{
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec"},
			Env:  []string{"RAILS_ENV=test"},
		},
		Targets: []string{"spec/user_spec.rb"},
	}
	cmd := run.Command(t.TempDir())

	assert.Equal(t,
		"RAILS_ENV=test bundle exec rspec spec/user_spec.rb",
		CommandString(cmd, run.Job.Env))
}

func TestCommandString_NoAddedEnv(t *testing.T) {
	run := JobRun{Job: framework.Job{Name: "rspec", Cmd: []string{"rspec"}}}
	cmd := run.Command(t.TempDir())

	assert.Equal(t, "rspec", CommandString(cmd, nil))
}

func TestExecuteJob_BatchesMultipleTargets(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	run := JobRun{
		Job: framework.Job{
			Name: "test-batch",
			Cmd:  []string{"sh", "-c", "echo \"$@\" > " + outputFile, "--"},
		},
		Targets: []string{"file1.rb", "file2.rb", "file3.rb"},
	}

	err := ExecuteJob(run, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	output := string(content)
	assert.Contains(t, output, "file1.rb")
	assert.Contains(t, output, "file2.rb")
	assert.Contains(t, output, "file3.rb")
}

func TestExecuteJob_NoTargets(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	run := JobRun{
		Job: framework.Job{
			Name: "test-empty",
			Cmd:  []string{"sh", "-c", "echo ran > args.txt", "--"},
		},
	}

	err := ExecuteJob(run, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, "ran\n", string(content))
}

func TestExecuteJob_JobEnvIsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "env.txt")

	run := JobRun{
		Job: framework.Job{
			Name: "test-env",
			Cmd:  []string{"sh", "-c", "echo \"$PLUR_TEST_VAR\" > " + outputFile},
			Env:  []string{"PLUR_TEST_VAR=from-job-config"},
		},
	}

	err := ExecuteJob(run, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, "from-job-config\n", string(content))
}

func TestExecuteJob_EmptyCmdErrors(t *testing.T) {
	err := ExecuteJob(JobRun{Job: framework.Job{Name: "broken"}}, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `job "broken" must define a command`)
}

// A burst of saves must not put two suites on the same test database at once.
func TestExecuteJob_RunsAreSerialized(t *testing.T) {
	tmpDir := t.TempDir()
	log := filepath.Join(tmpDir, "log")

	// Each run brackets itself with start/end markers. Serialized runs produce
	// strictly alternating pairs; any overlap interleaves them.
	run := JobRun{
		Job: framework.Job{
			Name: "overlap-detector",
			Cmd:  []string{"sh", "-c", "echo start >> " + log + "; sleep 0.2; echo end >> " + log},
		},
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			assert.NoError(t, ExecuteJob(run, tmpDir))
		})
	}
	wg.Wait()

	content, err := os.ReadFile(log)
	require.NoError(t, err)
	assert.Equal(t, "start\nend\nstart\nend\nstart\nend\nstart\nend\n", string(content),
		"jobs overlapped instead of running one at a time")
}

func TestTerminateRunningJob_StopsInFlightJob(t *testing.T) {
	tmpDir := t.TempDir()
	run := JobRun{
		Job: framework.Job{Name: "sleeper", Cmd: []string{"sleep", "60"}},
	}

	done := make(chan error, 1)
	go func() { done <- ExecuteJob(run, tmpDir) }()

	require.Eventually(t, func() bool { return currentProcess() != nil }, 5*time.Second, 10*time.Millisecond,
		"job never started")

	TerminateRunningJob()

	select {
	case err := <-done:
		assert.Error(t, err, "a terminated job reports a non-zero exit")
	case <-time.After(5 * time.Second):
		t.Fatal("job outlived TerminateRunningJob")
	}
	assert.Nil(t, currentProcess(), "process handle is cleared once the job is reaped")
}

// Ruby test runners rescue Interrupt, so plur must escalate past SIGTERM.
func TestTerminateRunningJob_KillsJobThatIgnoresSIGTERM(t *testing.T) {
	tmpDir := t.TempDir()
	run := JobRun{
		Job: framework.Job{
			Name: "stubborn",
			Cmd:  []string{"sh", "-c", "trap '' TERM; sleep 60"},
		},
	}

	done := make(chan error, 1)
	go func() { done <- ExecuteJob(run, tmpDir) }()

	require.Eventually(t, func() bool { return currentProcess() != nil }, 5*time.Second, 10*time.Millisecond,
		"job never started")

	TerminateRunningJob()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("job survived the SIGKILL escalation")
	}
}

func TestTerminateRunningJob_NoJobRunning(t *testing.T) {
	assert.NotPanics(t, TerminateRunningJob)
}
