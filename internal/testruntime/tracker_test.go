package testruntime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeTracker(t *testing.T) {
	t.Run("AddRuntime accumulates runtimes for same file", func(t *testing.T) {
		tempDir := t.TempDir()
		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)

		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)
		rt.AddRuntime("spec/foo_spec.rb", 0.5)

		pending := rt.PendingFileRuntimes()
		assert.Equal(t, 2.0, pending["spec/foo_spec.rb"])
		assert.Equal(t, 2.0, pending["spec/bar_spec.rb"])
	})

	t.Run("AddTestNotification extracts runtime from test notification", func(t *testing.T) {
		tempDir := t.TempDir()
		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)

		notification := types.TestCaseNotification{
			FilePath: "spec/test_spec.rb",
			Duration: time.Duration(123 * time.Millisecond),
		}

		rt.AddTestNotification(notification)

		assert.InDelta(t, 0.123, rt.PendingFileRuntimes()["spec/test_spec.rb"], 0.001)
	})

	t.Run("SaveToFile creates v2 runtime file under returned path", func(t *testing.T) {
		tempDir := t.TempDir()
		specPath := writeFixtureSpec(t, tempDir, "foo_spec.rb")

		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt.AddRuntime(specPath, 1.5)

		require.NoError(t, rt.SaveToFile(RunKindAggregate))

		runtimeFile := rt.RuntimeFilePath()
		_, err = os.Stat(runtimeFile)
		assert.NoError(t, err)

		reloaded := LoadRuntimeCache(runtimeFile)
		assert.Equal(t, RuntimeCacheSchemaVersion, reloaded.Meta.SchemaVersion)
		assert.NotEmpty(t, reloaded.Meta.PlurVersion)
		assert.NotEmpty(t, reloaded.Run.Cwd)
		assert.NotEmpty(t, reloaded.Run.LastRunAt)
		_, err = time.Parse(time.RFC3339, reloaded.Run.LastRunAt)
		assert.NoError(t, err)
		entry := reloaded.File(specPath)
		require.NotNil(t, entry)
		assert.Equal(t, 1.5, entry.RuntimeSeconds)
		assert.True(t, entry.ExampleIndexComplete)
	})

	t.Run("Aggregate-eligible runs overwrite file-level runtime; partial runs preserve it", func(t *testing.T) {
		tempDir := t.TempDir()
		specPath := writeFixtureSpec(t, tempDir, "foo_spec.rb")

		rt1, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt1.AddRuntime(specPath, 5.0)
		require.NoError(t, rt1.SaveToFile(RunKindAggregate))

		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt2.AddRuntime(specPath, 0.1) // a focused observation
		require.NoError(t, rt2.SaveToFile(RunKindPartial))

		rt3, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		assert.Equal(t, 5.0, rt3.LoadedData()[specPath],
			"partial run must not overwrite the aggregate")
	})

	t.Run("Per-example observations merge by TestID into existing aggregate", func(t *testing.T) {
		tempDir := t.TempDir()
		specPath := writeFixtureSpec(t, tempDir, "foo_spec.rb")

		rt1, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt1.AddTestNotification(types.TestCaseNotification{
			TestID:     "./" + specPath + "[1:1]",
			FilePath:   specPath,
			LineNumber: 5,
			Duration:   500 * time.Millisecond,
		})
		rt1.AddTestNotification(types.TestCaseNotification{
			TestID:     "./" + specPath + "[1:2]",
			FilePath:   specPath,
			LineNumber: 10,
			Duration:   1500 * time.Millisecond,
		})
		require.NoError(t, rt1.SaveToFile(RunKindAggregate))

		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		entry := rt2.Cache().File(specPath)
		require.NotNil(t, entry)
		assert.Len(t, entry.Examples, 2)
		assert.Equal(t, 5, entry.Examples["./"+specPath+"[1:1]"].LineNumber)
	})

	t.Run("LoadedData reflects loaded v2 cache", func(t *testing.T) {
		tempDir := t.TempDir()
		specPath := writeFixtureSpec(t, tempDir, "foo_spec.rb")

		rt1, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt1.AddRuntime(specPath, 7.0)
		require.NoError(t, rt1.SaveToFile(RunKindAggregate))

		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		assert.Equal(t, 7.0, rt2.LoadedData()[specPath])
	})
}

func writeFixtureSpec(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("# fixture\n"), 0644))
	return path
}
