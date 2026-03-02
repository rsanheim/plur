package main

import (
	"encoding/json"
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

		rt.AddRuntime("spec/foo_spec.rb", 1.5, 3)
		rt.AddRuntime("spec/bar_spec.rb", 2.0, 5)
		rt.AddRuntime("spec/foo_spec.rb", 0.5, 2)

		assert.Equal(t, 2.0, rt.runtimes["spec/foo_spec.rb"].TotalSeconds)
		assert.Equal(t, 5, rt.runtimes["spec/foo_spec.rb"].ExampleCount)
		assert.Equal(t, 2.0, rt.runtimes["spec/bar_spec.rb"].TotalSeconds)
		assert.Equal(t, 5, rt.runtimes["spec/bar_spec.rb"].ExampleCount)
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

		assert.InDelta(t, 0.123, rt.runtimes["spec/test_spec.rb"].TotalSeconds, 0.001)
		assert.Equal(t, 1, rt.runtimes["spec/test_spec.rb"].ExampleCount)
	})

	t.Run("SaveToFile writes schema v2 format", func(t *testing.T) {
		tempDir := t.TempDir()
		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt.SetFramework("rspec")

		rt.AddRuntime("spec/foo_spec.rb", 1.5, 3)
		rt.AddRuntime("spec/bar_spec.rb", 2.0, 5)

		err = rt.SaveToFile()
		require.NoError(t, err)

		runtimeFile := rt.RuntimeFilePath()
		_, err = os.Stat(runtimeFile)
		require.NoError(t, err)

		rawBytes, err := os.ReadFile(runtimeFile)
		require.NoError(t, err)

		var runtimeData RuntimeData
		err = json.Unmarshal(rawBytes, &runtimeData)
		require.NoError(t, err)

		assert.Equal(t, RuntimeSchemaVersion, runtimeData.SchemaVersion)
		assert.NotEmpty(t, runtimeData.ProjectRoot)
		assert.Len(t, runtimeData.ProjectHash, 8)
		assert.NotEmpty(t, runtimeData.GeneratedAt)
		assert.NotEmpty(t, runtimeData.PlurVersion)
		assert.Equal(t, "rspec", runtimeData.Framework)

		fooRT := runtimeData.Files["spec/foo_spec.rb"]
		assert.Equal(t, 1.5, fooRT.TotalSeconds)
		assert.Equal(t, 3, fooRT.ExampleCount)
		assert.NotEmpty(t, fooRT.LastSeen)

		barRT := runtimeData.Files["spec/bar_spec.rb"]
		assert.Equal(t, 2.0, barRT.TotalSeconds)
		assert.Equal(t, 5, barRT.ExampleCount)
	})

	t.Run("SaveToFile merges existing data with new data", func(t *testing.T) {
		tempDir := t.TempDir()

		// First tracker: save some data
		rt1, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		rt1.SetFramework("rspec")
		rt1.AddRuntime("spec/foo_spec.rb", 1.5, 3)
		rt1.AddRuntime("spec/bar_spec.rb", 2.0, 5)
		err = rt1.SaveToFile()
		require.NoError(t, err)

		// Second tracker: add new data for one file, leave other untouched
		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		// Should have loaded existing data (LoadedData returns float64 values)
		assert.Equal(t, 1.5, rt2.LoadedData()["spec/foo_spec.rb"])
		assert.Equal(t, 2.0, rt2.LoadedData()["spec/bar_spec.rb"])

		// Add new runtime for just one file
		rt2.AddRuntime("spec/foo_spec.rb", 3.0, 6)
		err = rt2.SaveToFile()
		require.NoError(t, err)

		// Third tracker: verify merge happened correctly
		rt3, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)
		assert.Equal(t, 3.0, rt3.LoadedData()["spec/foo_spec.rb"], "foo should have new value")
		assert.Equal(t, 2.0, rt3.LoadedData()["spec/bar_spec.rb"], "bar should be preserved")
	})

	t.Run("ignores legacy flat format files", func(t *testing.T) {
		tempDir := t.TempDir()

		rt, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)

		// Write a legacy-format file (flat key:value, no schema_version)
		legacyData := map[string]float64{
			"spec/foo_spec.rb": 1.5,
			"spec/bar_spec.rb": 2.0,
		}
		legacyBytes, _ := json.Marshal(legacyData)
		err = os.WriteFile(rt.RuntimeFilePath(), legacyBytes, 0644)
		require.NoError(t, err)

		// Re-create tracker to load
		rt2, err := NewRuntimeTracker(tempDir)
		require.NoError(t, err)

		assert.Empty(t, rt2.LoadedData(), "legacy format should be ignored")
	})
}

func TestSanitizeProjectPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/rob/src/myapp", "Users_rob_src_myapp"},
		{"/home/user/project", "home_user_project"},
		{"/tmp/a", "tmp_a"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, sanitizeProjectPath(tc.input))
	}
}
