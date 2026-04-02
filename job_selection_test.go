package main

import (
	"os"
	"testing"

	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectJobFromRuntimeConfig_UsesExplicitUse(t *testing.T) {
	rc := &RuntimeConfig{
		Use: "custom",
		Jobs: map[string]job.Job{
			"custom": {Name: "custom", Cmd: []string{"custom-runner"}, Framework: "passthrough"},
		},
		Inherited: map[string]autodetect.InheritedFields{},
	}

	selected, err := selectJobFromRuntimeConfig(rc, nil)
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

	selected, err := selectJobFromRuntimeConfig(rc, []string{"spec/example_spec.rb"})
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

	selected, err := selectJobFromRuntimeConfig(rc, nil)
	require.NoError(t, err)
	assert.Equal(t, "rspec", selected.Name)
	assert.Equal(t, ResolveReasonAutodetect, selected.Reason)
	assert.True(t, selected.Inherited.Framework)
}
