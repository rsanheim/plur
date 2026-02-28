package main

import (
	"os"
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

		assert.Equal(t, 2.0, rt.runtimes["spec/foo_spec.rb"], "foo_spec.rb runtime should be accumulated")
		assert.Equal(t, 2.0, rt.runtimes["spec/bar_spec.rb"], "bar_spec.rb runtime should be correct")
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

		assert.InDelta(t, 0.123, rt.runtimes["spec/test_spec.rb"], 0.001, "test_spec.rb runtime should be extracted from notification")
	})

	t.Run("SaveToFile creates runtime file and RuntimeFilePath returns correct path", func(t *testing.T) {
		tempDir := t.TempDir()
		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)

		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)

		err = rt.SaveToFile()
		assert.NoError(t, err, "Failed to save runtime data")

		runtimeFile := rt.RuntimeFilePath()
		_, err = os.Stat(runtimeFile)
		assert.NoError(t, err, "runtime file should be created at path returned by RuntimeFilePath()")
	})

	t.Run("SaveToFile merges existing data with new data", func(t *testing.T) {
		tempDir := t.TempDir()

		// First tracker: save some data
		rt1, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt1.AddRuntime("spec/foo_spec.rb", 1.5)
		rt1.AddRuntime("spec/bar_spec.rb", 2.0)
		err = rt1.SaveToFile()
		require.NoError(t, err)

		// Second tracker: add new data for one file, leave other untouched
		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		// Should have loaded existing data
		assert.Equal(t, 1.5, rt2.LoadedData()["spec/foo_spec.rb"])
		assert.Equal(t, 2.0, rt2.LoadedData()["spec/bar_spec.rb"])

		// Add new runtime for just one file
		rt2.AddRuntime("spec/foo_spec.rb", 3.0)
		err = rt2.SaveToFile()
		require.NoError(t, err)

		// Third tracker: verify merge happened correctly
		rt3, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		// foo_spec.rb should have new value, bar_spec.rb should be preserved
		assert.Equal(t, 3.0, rt3.LoadedData()["spec/foo_spec.rb"], "foo should have new value")
		assert.Equal(t, 2.0, rt3.LoadedData()["spec/bar_spec.rb"], "bar should be preserved")
	})
}
