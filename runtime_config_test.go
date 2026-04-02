package main

import (
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRuntimeConfig_PreservesUseJobsAndUserWatches(t *testing.T) {
	cli := &PlurCLI{
		Use: "rspec",
		Job: map[string]job.Job{
			"rspec": {Cmd: []string{"custom-rspec"}, Framework: "rspec"},
		},
		WatchMappings: []watch.WatchMapping{
			{Name: "custom-watch", Source: "spec/**/*_spec.rb", Jobs: []string{"rspec"}},
		},
		configFiles: []string{"~/.plur.toml", ".plur.toml"},
	}

	rc, err := buildRuntimeConfig(cli)
	require.NoError(t, err)
	assert.Equal(t, "rspec", rc.Use)
	assert.Contains(t, rc.Jobs, "rspec")
	assert.Len(t, rc.Watches, 1)
	assert.Equal(t, "custom-watch", rc.Watches[0].Name)
}

func TestBuildRuntimeConfig_FallsBackToSelectedBuiltinWatches(t *testing.T) {
	cli := &PlurCLI{
		Use:         "rspec",
		configFiles: []string{"~/.plur.toml", ".plur.toml"},
	}

	rc, err := buildRuntimeConfig(cli)
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
