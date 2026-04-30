package runtime

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRuntimeConfig_MergesBuiltinAndUserWatches(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]job.Job{
			"rspec": {Cmd: []string{"custom-rspec"}, Framework: "rspec"},
		},
		WatchMappings: []watch.WatchMapping{{Name: "custom-watch", Source: "config/**/*.yml", Jobs: []string{"rspec"}}},
		ConfigFiles:   []string{"~/.plur.toml", ".plur.toml"},
	}

	rc, err := BuildRuntimeConfig(cli)
	require.NoError(t, err)
	assert.Equal(t, "rspec", rc.Use)
	assert.Contains(t, rc.Jobs, "rspec")
	var names []string
	for _, watch := range rc.Watches {
		names = append(names, watch.Name)
	}
	assert.Contains(t, names, "custom-watch")
	assert.Contains(t, names, "lib-to-spec")
	assert.Contains(t, names, "spec-files")
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

func TestBuildRuntimeConfig_UserWatchOverridesBuiltinByName(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]job.Job{
			"rspec": {Cmd: []string{"bin/rspec"}, Framework: "rspec"},
		},
		WatchMappings: []watch.WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/override_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
	}

	rc, err := BuildRuntimeConfig(cli)
	require.NoError(t, err)

	var override *watch.WatchMapping
	var count int
	for i := range rc.Watches {
		if rc.Watches[i].Name == "lib-to-spec" {
			override = &rc.Watches[i]
			count++
		}
	}

	require.NotNil(t, override)
	assert.Equal(t, 1, count)
	assert.Equal(t, []string{"spec/override_spec.rb"}, override.Targets)
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
		Inherited: map[string]InheritedFields{},
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
		Inherited: map[string]InheritedFields{},
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
		Inherited: map[string]InheritedFields{
			"rspec": {Framework: true, TargetPattern: true},
		},
	}

	selected, err := SelectJobFromRuntimeConfig(rc, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", selected.Name)
	assert.Equal(t, ResolveReasonAutodetect, selected.Reason)
	assert.True(t, selected.Inherited.Framework)
}

func TestBuildRuntimeConfigIncludesRailsAndRakeJobs(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	require.Contains(t, rc.Jobs, "rake")

	assert.Equal(t, []string{"bin/rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rails"].Framework)
	assert.Empty(t, rc.Jobs["rails"].Env)

	assert.Equal(t, []string{"bundle", "exec", "rake"}, rc.Jobs["rake"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rake"].Framework)
	assert.Empty(t, rc.Jobs["rake"].Env)
}

func TestBuildRuntimeConfigRailsJobOverrides(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{
		Jobs: map[string]job.Job{
			"rails": {
				Cmd: []string{"bundle", "exec", "rails"},
				Env: []string{"RAILS_ENV=test"},
			},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	assert.Equal(t, []string{"bundle", "exec", "rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rails"].Framework)
	assert.Equal(t, []string{"RAILS_ENV=test"}, rc.Jobs["rails"].Env)
	assert.True(t, rc.Inherited["rails"].Framework)
	assert.False(t, rc.Inherited["rails"].Env)
}

func TestBuildRuntimeConfigRailsJobEnvOnlyOverrideInheritsCommand(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{
		Jobs: map[string]job.Job{
			"rails": {Env: []string{"RAILS_ENV=test"}},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	assert.Equal(t, []string{"bin/rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rails"].Framework)
	assert.Equal(t, []string{"RAILS_ENV=test"}, rc.Jobs["rails"].Env)
	assert.True(t, rc.Inherited["rails"].Cmd)
	assert.True(t, rc.Inherited["rails"].Framework)
	assert.False(t, rc.Inherited["rails"].Env)
}
