package runtime

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRuntimeConfig_PreservesUseJobsAndUserWatches(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]job.Job{
			"rspec": {Cmd: []string{"custom-rspec"}, Framework: "rspec"},
		},
		WatchMappings: []watch.WatchMapping{{Name: "custom-watch", Source: "spec/**/*_spec.rb", Jobs: []string{"rspec"}}},
		ConfigFiles:   []string{"~/.plur.toml", ".plur.toml"},
	}

	rc, err := BuildRuntimeConfig(cli)
	require.NoError(t, err)
	assert.Equal(t, "rspec", rc.Use)
	assert.Contains(t, rc.Jobs, "rspec")
	assert.Len(t, rc.Watches, 1)
	assert.Equal(t, "custom-watch", rc.Watches[0].Name)
}

func TestBuildRuntimeConfig_FallsBackToSelectedBuiltinWatches(t *testing.T) {
	cli := &CLIInput{
		Use:         "rspec",
		ConfigFiles: []string{"~/.plur.toml", ".plur.toml"},
	}

	rc, err := BuildRuntimeConfig(cli)
	require.NoError(t, err)
	require.NotEmpty(t, rc.Watches)
	for _, watch := range rc.Watches {
		assert.Contains(t, watch.Jobs, "rspec")
	}
}

func TestValidateRuntimeConfigRejectsUndefinedWatchJob(t *testing.T) {
	rc := &RuntimeConfig{
		Use: "rspec",
		Jobs: map[string]job.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, Framework: "rspec"},
		},
		Watches: []watch.WatchMapping{
			{Name: "bad-watch", Source: "spec/**/*_spec.rb", Jobs: []string{"missing"}},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-watch")
	assert.Contains(t, err.Error(), "missing")
}

func TestValidateRuntimeConfigRejectsJobWithoutCommand(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]job.Job{
			"custom": {Name: "custom", Framework: "passthrough"},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "job \"custom\" must define a command")
}

func TestSelectJobFromRuntimeConfig_UsesExplicitUse(t *testing.T) {
	rc := &RuntimeConfig{
		Use: "custom",
		Jobs: map[string]job.Job{
			"custom": {Name: "custom", Cmd: []string{"custom-runner"}, Framework: "passthrough"},
		},
		Inherited: map[string]autodetect.InheritedFields{},
	}

	selected, err := SelectJobFromRuntimeConfig(rc, nil)
	require.NoError(t, err)
	assert.Equal(t, "custom", selected.Name)
	assert.Equal(t, ResolveReasonExplicitName, selected.Reason)
}

func TestSelectJobFromRuntimeConfig_InfersFrameworkFromPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/example_spec.rb", []byte(""), 0o644)

	rc := &RuntimeConfig{
		Jobs: map[string]job.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, Framework: "rspec", TargetPattern: "spec/**/*_spec.rb"},
		},
		Inherited: map[string]autodetect.InheritedFields{},
	}

	selected, err := SelectJobFromRuntimeConfig(rc, []string{"spec/example_spec.rb"})
	require.NoError(t, err)
	assert.Equal(t, "rspec", selected.Name)
	assert.Equal(t, ResolveReasonExplicitPatterns, selected.Reason)
}

func TestSelectJobFromRuntimeConfig_FallsBackToAutodetect(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/example_spec.rb", []byte(""), 0o644)

	rc := &RuntimeConfig{
		Jobs: map[string]job.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, Framework: "rspec", TargetPattern: "spec/**/*_spec.rb"},
		},
		Inherited: map[string]autodetect.InheritedFields{
			"rspec": {Framework: true, TargetPattern: true},
		},
	}

	selected, err := SelectJobFromRuntimeConfig(rc, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", selected.Name)
	assert.Equal(t, ResolveReasonAutodetect, selected.Reason)
	assert.True(t, selected.Inherited.Framework)
}
