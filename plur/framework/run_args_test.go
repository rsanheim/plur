package framework

import (
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRunArgsRSpecDefaults(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{
		ColorOutput: false,
		ConfigPaths: config.InitConfigPaths(),
	}

	j := job.Job{
		Framework: "rspec",
		Cmd:       []string{"bundle", "exec", "rspec", "--fail-fast", "{{target}}"},
	}

	args, err := BuildRunArgs(j, []string{"spec/example_spec.rb"}, cfg)
	require.NoError(t, err)

	formatterPath := filepath.Join(cfg.ConfigPaths.FormatterDir, "json_rows_formatter.rb")
	expected := []string{
		"bundle", "exec", "rspec", "--fail-fast",
		"-r", formatterPath, "--format", "Plur::JsonRowsFormatter",
		"--no-color",
		"spec/example_spec.rb",
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestRubyRequire(t *testing.T) {
	cfg := &config.GlobalConfig{}
	j := job.Job{
		Framework: "minitest",
		Cmd:       []string{"bundle", "exec", "ruby", "-Itest", "{{target}}"},
	}

	args, err := BuildRunArgs(j, []string{"test/foo_test.rb", "test/bar_test.rb"}, cfg)
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest",
		"-e", `["foo_test", "bar_test"].each { |f| require f }`,
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestSingleFile(t *testing.T) {
	cfg := &config.GlobalConfig{}
	j := job.Job{
		Framework: "minitest",
		Cmd:       []string{"bundle", "exec", "ruby", "-Itest", "{{target}}"},
	}

	args, err := BuildRunArgs(j, []string{"test/foo_test.rb"}, cfg)
	require.NoError(t, err)

	expected := []string{"bundle", "exec", "ruby", "-Itest", "test/foo_test.rb"}
	assert.Equal(t, expected, args)
}
