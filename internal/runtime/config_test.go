package runtime

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRuntimeConfig_MergesBuiltinAndUserWatches(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]framework.Job{
			"rspec": {Cmd: []string{"custom-rspec"}, FrameworkName: "rspec"},
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

func TestBuildRuntimeConfigMarksBuiltinDefaultsAsInherited(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{Use: "rspec"})

	require.NoError(t, err)
	assert.True(t, rc.Inherited["rspec"].Cmd)
	assert.True(t, rc.Inherited["rspec"].Framework)
	assert.True(t, rc.Inherited["rspec"].TargetPattern)
}

func TestBuildRuntimeConfig_UserWatchOverridesBuiltinByName(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]framework.Job{
			"rspec": {Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
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

func TestBuildRuntimeConfig_PreservesUserExcludePatterns(t *testing.T) {
	cli := &CLIInput{
		Use: "rspec",
		Jobs: map[string]framework.Job{
			"rspec": {
				Cmd:             []string{"bin/rspec"},
				FrameworkName:   "rspec",
				ExcludePatterns: []string{"spec/system/**/*_spec.rb"},
			},
		},
	}

	rc, err := BuildRuntimeConfig(cli)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/system/**/*_spec.rb"}, rc.Jobs["rspec"].ExcludePatterns)
	assert.False(t, rc.Inherited["rspec"].ExcludePatterns,
		"user-supplied excludes should not be marked as inherited")
}

func TestValidateRuntimeConfigRejectsUndefinedWatchJob(t *testing.T) {
	rc := &RuntimeConfig{
		Use: "rspec",
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
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

func TestValidateRuntimeConfigRejectsNoTargetsWithTargets(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"build": {Name: "build", Cmd: []string{"bin/rake", "install"}, FrameworkName: "passthrough"},
		},
		Watches: []watch.WatchMapping{
			{
				Name:      "bad-watch",
				Source:    "**/*.go",
				Targets:   []string{"{{path}}"},
				NoTargets: true,
				Jobs:      []string{"build"},
			},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad-watch")
	assert.Contains(t, err.Error(), "must not define targets when no_targets is true")
}

func TestValidateRuntimeConfigRejectsJobWithoutCommand(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"custom": {Name: "custom", FrameworkName: "passthrough"},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "job \"custom\" must define a command")
}

func TestValidateRuntimeConfigRejectsTemplateTokensInJobCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  []string
	}{
		{
			name: "standalone template argument",
			cmd:  []string{"bin/rspec", "{{target}}"},
		},
		{
			name: "embedded template argument",
			cmd:  []string{"bin/rspec", "--file={{target}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RuntimeConfig{
				Jobs: map[string]framework.Job{
					"custom": {Name: "custom", Cmd: tt.cmd, FrameworkName: "rspec"},
				},
				Sources: []string{".plur.toml"},
			}

			err := validateRuntimeConfig(rc)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "job \"custom\" command must not contain {{target}} tokens")
		})
	}
}

func TestValidateRuntimeConfigRejectsInvalidSourcePattern(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
		},
		Watches: []watch.WatchMapping{
			{Name: "bad-source", Source: "lib/[", Jobs: []string{"rspec"}},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid source pattern "lib/["`)
}

func TestValidateRuntimeConfigRejectsInvalidIgnorePattern(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
		},
		Watches: []watch.WatchMapping{
			{Name: "bad-ignore", Source: "lib/**/*.rb", Ignore: []string{"vendor/["}, Jobs: []string{"rspec"}},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid ignore pattern "vendor/["`)
}

func TestValidateRuntimeConfigRejectsWatchWithoutSource(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
		},
		Watches: []watch.WatchMapping{
			{Name: "missing-source", Jobs: []string{"rspec"}},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `watch "missing-source" must define source`)
}

func TestValidateRuntimeConfigRejectsWatchWithoutJobs(t *testing.T) {
	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec"},
		},
		Watches: []watch.WatchMapping{
			{Name: "missing-jobs", Source: "lib/**/*.rb"},
		},
		Sources: []string{".plur.toml"},
	}

	err := validateRuntimeConfig(rc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `watch "missing-jobs" must define at least one job`)
}

func TestSelectJobFromRuntimeConfig_UsesExplicitUse(t *testing.T) {
	rc := &RuntimeConfig{
		Use: "custom",
		Jobs: map[string]framework.Job{
			"custom": {Name: "custom", Cmd: []string{"custom-runner"}, FrameworkName: "passthrough"},
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
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll("spec", 0o755))
	require.NoError(t, os.WriteFile("spec/example_spec.rb", []byte(""), 0o644))

	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec", TargetPattern: "spec/**/*_spec.rb"},
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
	t.Chdir(tmpDir)

	require.NoError(t, os.MkdirAll("spec", 0o755))
	require.NoError(t, os.WriteFile("spec/example_spec.rb", []byte(""), 0o644))

	rc := &RuntimeConfig{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bin/rspec"}, FrameworkName: "rspec", TargetPattern: "spec/**/*_spec.rb"},
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
	assert.Equal(t, "passthrough", rc.Jobs["rails"].FrameworkName)
	assert.Empty(t, rc.Jobs["rails"].Env)

	assert.Equal(t, []string{"bundle", "exec", "rake"}, rc.Jobs["rake"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rake"].FrameworkName)
	assert.Empty(t, rc.Jobs["rake"].Env)
}

func TestBuildRuntimeConfigRailsJobOverrides(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{
		Jobs: map[string]framework.Job{
			"rails": {
				Cmd: []string{"bundle", "exec", "rails"},
				Env: []string{"CUSTOM_ENV=value"},
			},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	assert.Equal(t, []string{"bundle", "exec", "rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rails"].FrameworkName)
	assert.Equal(t, []string{"CUSTOM_ENV=value"}, rc.Jobs["rails"].Env)
	assert.True(t, rc.Inherited["rails"].Framework)
	assert.False(t, rc.Inherited["rails"].Env)
}

func TestBuildRuntimeConfigRailsJobEnvOnlyOverrideInheritsCommand(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{
		Jobs: map[string]framework.Job{
			"rails": {Env: []string{"CUSTOM_ENV=value"}},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	assert.Equal(t, []string{"bin/rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "passthrough", rc.Jobs["rails"].FrameworkName)
	assert.Equal(t, []string{"CUSTOM_ENV=value"}, rc.Jobs["rails"].Env)
	assert.True(t, rc.Inherited["rails"].Cmd)
	assert.True(t, rc.Inherited["rails"].Framework)
	assert.False(t, rc.Inherited["rails"].Env)
}
