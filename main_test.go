package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/alecthomas/kong"
	kongtoml "github.com/rsanheim/plur/internal/kongtoml"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerCountCLIDefaultMatchesRuntimeDefault(t *testing.T) {
	field, ok := reflect.TypeOf(PlurCLI{}).FieldByName("Workers")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeOf(WorkerCount(0)), field.Type)
	assert.Equal(t, strconv.Itoa(DefaultWorkerCount), field.Tag.Get("default"))
}

func TestWorkerCountValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		env  string
	}{
		{name: "rejects zero CLI value", args: []string{"--workers=0"}},
		{name: "rejects negative CLI value", args: []string{"--workers=-1"}},
		{name: "rejects zero environment value", env: "0"},
		{name: "rejects negative environment value", env: "-1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setTestEnv(t, "PARALLEL_TEST_PROCESSORS", tt.env, tt.env != "")

			var parsed workerCountTestCLI
			parser, err := kong.New(&parsed)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "workers must be at least 1")
		})
	}
}

func TestWorkerCountValidationAcceptsPositiveValues(t *testing.T) {
	for _, tt := range []struct {
		name     string
		args     []string
		env      string
		expected int
	}{
		{name: "uses CLI value", args: []string{"--workers=8"}, expected: 8},
		{name: "uses environment value", env: "6", expected: 6},
		{name: "uses default value", expected: DefaultWorkerCount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setTestEnv(t, "PARALLEL_TEST_PROCESSORS", tt.env, tt.env != "")

			var parsed workerCountTestCLI
			parser, err := kong.New(&parsed)
			require.NoError(t, err)

			_, err = parser.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, int(parsed.Workers))
		})
	}
}

func TestWorkerCountValidationRejectsConfigValue(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "plur.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("workers = 0\n"), 0644))
	setTestEnv(t, "PARALLEL_TEST_PROCESSORS", "", false)

	var parsed workerCountTestCLI
	parser, err := kong.New(&parsed, kong.Configuration(kongtoml.Loader, configPath))
	require.NoError(t, err)

	_, err = parser.Parse(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workers must be at least 1")
}

type workerCountTestCLI struct {
	Workers WorkerCount `short:"n" help:"Number of parallel workers" env:"PARALLEL_TEST_PROCESSORS" default:"10"`
}

func (cli *workerCountTestCLI) Validate() error {
	return cli.Workers.Validate()
}

func setTestEnv(t *testing.T, key string, value string, present bool) {
	t.Helper()

	original, wasSet := os.LookupEnv(key)
	if present {
		require.NoError(t, os.Setenv(key, value))
	} else {
		require.NoError(t, os.Unsetenv(key))
	}

	t.Cleanup(func() {
		if wasSet {
			require.NoError(t, os.Setenv(key, original))
		} else {
			require.NoError(t, os.Unsetenv(key))
		}
	})
}

func TestValidateUniqueWatchNames(t *testing.T) {
	t.Run("allows unnamed and unique named watches", func(t *testing.T) {
		err := validateUniqueWatchNames([]watch.WatchMapping{
			{Name: "lib-to-spec", Source: "lib/**/*.rb"},
			{Name: "", Source: "README.md"},
			{Name: "spec-files", Source: "spec/**/*_spec.rb"},
		}, []string{".plur.toml"})

		require.NoError(t, err)
	})

	t.Run("rejects duplicate named watches", func(t *testing.T) {
		err := validateUniqueWatchNames([]watch.WatchMapping{
			{Name: "lint", Source: "config/**/*.yml"},
			{Name: "lint", Source: "README.md"},
		}, []string{".plur.toml"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate watch name")
		assert.Contains(t, err.Error(), `"lint"`)
	})
}

func TestRailsCommandAliasSelectsRakeJob(t *testing.T) {
	setTestEnv(t, "PLUR_HOME", t.TempDir(), true)

	var cli PlurCLI
	parser, err := kong.New(&cli)
	require.NoError(t, err)

	ctx, err := parser.Parse([]string{"--dry-run", "rake", "db:setup", "-n", "2"})
	require.NoError(t, err)

	assert.Equal(t, []string{"db:setup"}, cli.Rails.Args)
	assert.Equal(t, 2, int(cli.Workers))
	assert.Equal(t, "rake", railsCommandJobName(ctx))
}

func TestRailsCommandNameIgnoresFlagValues(t *testing.T) {
	setTestEnv(t, "PLUR_HOME", t.TempDir(), true)

	var cli PlurCLI
	parser, err := kong.New(&cli)
	require.NoError(t, err)

	ctx, err := parser.Parse([]string{"-C", "rake", "rails", "db:prepare"})
	require.NoError(t, err)

	assert.Equal(t, []string{"db:prepare"}, cli.Rails.Args)
	assert.Equal(t, "rails", railsCommandJobName(ctx))
}
