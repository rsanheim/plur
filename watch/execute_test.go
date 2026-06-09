package watch

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
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
