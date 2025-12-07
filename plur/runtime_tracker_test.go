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
		rt := NewRuntimeTracker()

		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)
		rt.AddRuntime("spec/foo_spec.rb", 0.5)

		assert.Equal(t, 2.0, rt.runtimes["spec/foo_spec.rb"], "foo_spec.rb runtime should be accumulated")
		assert.Equal(t, 2.0, rt.runtimes["spec/bar_spec.rb"], "bar_spec.rb runtime should be correct")
	})

	t.Run("AddTestNotification extracts runtime from test notification", func(t *testing.T) {
		rt := NewRuntimeTracker()

		notification := types.TestCaseNotification{
			FilePath: "spec/test_spec.rb",
			Duration: time.Duration(123 * time.Millisecond),
		}

		rt.AddTestNotification(notification)

		assert.InDelta(t, 0.123, rt.runtimes["spec/test_spec.rb"], 0.001, "test_spec.rb runtime should be extracted from notification")
	})

	t.Run("SaveToFile creates runtime.json", func(t *testing.T) {
		rt := NewRuntimeTracker()
		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)

		// Create temp directory for test
		// Save runtime data (it will use project-specific path)
		err := rt.SaveToFile()
		assert.NoError(t, err, "Failed to save runtime data")

		// Get the runtime file path and check it exists
		runtimeFile, err := GetRuntimeFilePath()
		require.NoError(t, err)
		defer os.Remove(runtimeFile) // Clean up after test

		_, err = os.Stat(runtimeFile)
		assert.NoError(t, err, "runtime.json should be created")
	})
}
