package framework

import (
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRunArgsRSpecDefaults(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{
		ColorOutput: false,
		ConfigPaths: config.InitConfigPaths(),
	}

	j := Job{
		FrameworkName: "rspec",
		Cmd:           []string{"bundle", "exec", "rspec", "--fail-fast"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"spec/example_spec.rb"}, cfg, nil)
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
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{ConfigPaths: config.InitConfigPaths()}
	j := Job{
		FrameworkName: "minitest",
		Cmd:           []string{"bundle", "exec", "ruby", "-Itest"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb", "test/bar_test.rb"}, cfg, nil)
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest",
		"-I" + cfg.ConfigPaths.RubyLibDir,
		"-e", `["test/foo_test.rb", "test/bar_test.rb"].each { |f| require File.expand_path(f) }; ` + minitestPluginEpilogue,
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestSingleFile(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{ConfigPaths: config.InitConfigPaths()}
	j := Job{
		FrameworkName: "minitest",
		Cmd:           []string{"bundle", "exec", "ruby", "-Itest"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb"}, cfg, nil)
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest",
		"-I" + cfg.ConfigPaths.RubyLibDir,
		"-e", `["test/foo_test.rb"].each { |f| require File.expand_path(f) }; ` + minitestPluginEpilogue,
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestSingleFileWithExtraArgs(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{ConfigPaths: config.InitConfigPaths()}
	j := Job{
		FrameworkName: "minitest",
		Cmd:           []string{"bundle", "exec", "ruby", "-Itest"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb"}, cfg, []string{"--seed", "1234"})
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest",
		"-I" + cfg.ConfigPaths.RubyLibDir,
		"-e", `["test/foo_test.rb"].each { |f| require File.expand_path(f) }; ` + minitestPluginEpilogue,
		"--seed", "1234",
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestRubyRequireWithExtraArgs(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{ConfigPaths: config.InitConfigPaths()}
	j := Job{
		FrameworkName: "minitest",
		Cmd:           []string{"bundle", "exec", "ruby", "-Itest"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb", "test/bar_test.rb"}, cfg, []string{"--seed", "1234"})
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest",
		"-I" + cfg.ConfigPaths.RubyLibDir,
		"-e", `["test/foo_test.rb", "test/bar_test.rb"].each { |f| require File.expand_path(f) }; ` + minitestPluginEpilogue,
		"--seed", "1234",
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsRSpecWithExtraArgs(t *testing.T) {
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{
		ColorOutput: false,
		ConfigPaths: config.InitConfigPaths(),
	}

	j := Job{
		FrameworkName: "rspec",
		Cmd:           []string{"bundle", "exec", "rspec"},
	}
	j = mustResolveJob(t, j)

	args, err := j.BuildRunArgs([]string{"spec/example_spec.rb"}, cfg, []string{"--tag", "slow"})
	require.NoError(t, err)

	formatterPath := filepath.Join(cfg.ConfigPaths.FormatterDir, "json_rows_formatter.rb")
	expected := []string{
		"bundle", "exec", "rspec",
		"-r", formatterPath, "--format", "Plur::JsonRowsFormatter",
		"--no-color",
		"--tag", "slow",
		"spec/example_spec.rb",
	}
	assert.Equal(t, expected, args)
}
