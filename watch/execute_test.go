package watch

import (
	"os"
	"path/filepath"
	"testing"

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
