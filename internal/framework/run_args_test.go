package framework

import (
	"os"
	"path/filepath"
	"strings"
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

// minitestJob returns a resolved minitest job plus the config its plugin
// load path is derived from.
func minitestJob(t *testing.T) (Job, *config.GlobalConfig, string) {
	t.Helper()
	t.Setenv("PLUR_HOME", t.TempDir())

	cfg := &config.GlobalConfig{ConfigPaths: config.InitConfigPaths()}
	j := mustResolveJob(t, Job{
		FrameworkName: "minitest",
		Cmd:           []string{"bundle", "exec", "ruby", "-Itest"},
	})

	return j, cfg, "-I" + filepath.Join(cfg.ConfigPaths.FormatterDir, "rubylib")
}

func TestBuildRunArgsMinitestRubyRequire(t *testing.T) {
	j, cfg, loadPath := minitestJob(t)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb", "test/bar_test.rb"}, cfg, nil)
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest", loadPath,
		"-e", `["foo_test", "bar_test"].each { |f| require f }`,
	}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestSingleFile(t *testing.T) {
	j, cfg, loadPath := minitestJob(t)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb"}, cfg, nil)
	require.NoError(t, err)

	expected := []string{"bundle", "exec", "ruby", "-Itest", loadPath, "test/foo_test.rb"}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestSingleFileWithExtraArgs(t *testing.T) {
	j, cfg, loadPath := minitestJob(t)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb"}, cfg, []string{"--seed", "1234"})
	require.NoError(t, err)

	expected := []string{"bundle", "exec", "ruby", "-Itest", loadPath, "test/foo_test.rb", "--seed", "1234"}
	assert.Equal(t, expected, args)
}

func TestBuildRunArgsMinitestRubyRequireWithExtraArgs(t *testing.T) {
	j, cfg, loadPath := minitestJob(t)

	args, err := j.BuildRunArgs([]string{"test/foo_test.rb", "test/bar_test.rb"}, cfg, []string{"--seed", "1234"})
	require.NoError(t, err)

	expected := []string{
		"bundle", "exec", "ruby", "-Itest", loadPath,
		"-e", `["foo_test", "bar_test"].each { |f| require f }`,
		"--seed", "1234",
	}
	assert.Equal(t, expected, args)
}

// The plugin has to land in a minitest/ subdirectory of the load path, since
// that is where minitest globs for plugins.
func TestBuildRunArgsMinitestWritesPlugin(t *testing.T) {
	j, cfg, loadPath := minitestJob(t)

	_, err := j.BuildRunArgs([]string{"test/foo_test.rb"}, cfg, nil)
	require.NoError(t, err)

	plugin := filepath.Join(strings.TrimPrefix(loadPath, "-I"), "minitest", "plur_plugin.rb")
	contents, err := os.ReadFile(plugin)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "def self.plugin_plur_init")
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
