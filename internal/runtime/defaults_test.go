package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinDefaultsLoad(t *testing.T) {
	assert.NotEmpty(t, builtinDefaults.Defaults.Jobs)
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "rspec")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "minitest")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "go-test")
}

func TestBuiltinGoWatchTargetsUseLocalPackagePatterns(t *testing.T) {
	expected := map[string][]string{
		"go-source": {"./{{dir_relative}}"},
		"go-tests":  {"./{{dir_relative}}"},
	}

	for _, watch := range builtinDefaults.Defaults.Watches {
		targets, ok := expected[watch.Name]
		if !ok {
			continue
		}

		assert.Equal(t, targets, watch.Targets)
		delete(expected, watch.Name)
	}

	assert.Empty(t, expected, "missing built-in Go watch mappings")
}

func writeDetectFile(t *testing.T, path string) {
	t.Helper()
	full := filepath.FromSlash(path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(""), 0o644))
}

func builtinResolvedJobs(t *testing.T) map[string]framework.Job {
	t.Helper()
	jobs, _, err := buildResolvedJobs(nil)
	require.NoError(t, err)
	return jobs
}

func TestAutodetectJobNameIgnoresVendoredTestFiles(t *testing.T) {
	t.Chdir(t.TempDir())

	writeDetectFile(t, "node_modules/pkg/example_test.go")
	writeDetectFile(t, "vendor/gem/spec/example_spec.rb")
	writeDetectFile(t, "tmp/scratch/example_test.go")
	writeDetectFile(t, ".git/hooks/example_test.go")

	_, err := autodetectJobName(builtinResolvedJobs(t))
	require.Error(t, err, "test files inside ignored directories must not drive detection")
	assert.Contains(t, err.Error(), "no default spec/test files found")
}

func TestAnyFileMatches(t *testing.T) {
	t.Run("finds a nested match", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeDetectFile(t, "spec/models/user_spec.rb")

		found, err := anyFileMatches("spec/**/*_spec.rb")
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("no match", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeDetectFile(t, "spec/models/user.rb")

		found, err := anyFileMatches("spec/**/*_spec.rb")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("missing base directory is not an error", func(t *testing.T) {
		t.Chdir(t.TempDir())

		found, err := anyFileMatches("spec/**/*_spec.rb")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("ignored directories are pruned", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeDetectFile(t, "node_modules/pkg/example_test.go")
		writeDetectFile(t, "vendor/pkg/example_test.go")
		writeDetectFile(t, "tmp/example_test.go")

		found, err := anyFileMatches("**/*_test.go")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("ignored directory name as pattern base is still searched", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeDetectFile(t, "vendor/specs/user_spec.rb")

		found, err := anyFileMatches("vendor/**/*_spec.rb")
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("symlinked base directory resolves", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeDetectFile(t, "real/user_spec.rb")
		require.NoError(t, os.Symlink("real", "spec"))

		found, err := anyFileMatches("spec/**/*_spec.rb")
		require.NoError(t, err)
		assert.True(t, found)
	})
}

func TestAutodetectJobNameFindsRealTestFilesNextToVendoredOnes(t *testing.T) {
	t.Chdir(t.TempDir())

	writeDetectFile(t, "node_modules/pkg/example_test.go")
	writeDetectFile(t, "pkg/example_test.go")

	name, err := autodetectJobName(builtinResolvedJobs(t))
	require.NoError(t, err)
	assert.Equal(t, "go-test", name)
}
