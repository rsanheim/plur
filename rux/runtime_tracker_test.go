package main

import (
	"os"
	"testing"

	"github.com/rsanheim/rux/rspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeTracker(t *testing.T) {
	t.Run("AddRuntime accumulates runtimes for same file", func(t *testing.T) {
		rt := NewRuntimeTracker()

		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)
		rt.AddRuntime("spec/foo_spec.rb", 0.5)

		runtimes := rt.GetRuntimes()

		assert.Equal(t, 2.0, runtimes["spec/foo_spec.rb"], "foo_spec.rb runtime should be accumulated")
		assert.Equal(t, 2.0, runtimes["spec/bar_spec.rb"], "bar_spec.rb runtime should be correct")
	})

	t.Run("AddExample extracts runtime from RSpec example", func(t *testing.T) {
		rt := NewRuntimeTracker()

		example := rspec.Example{
			FilePath: "spec/test_spec.rb",
			RunTime:  0.123,
		}

		rt.AddExample(example)

		runtimes := rt.GetRuntimes()
		assert.Equal(t, 0.123, runtimes["spec/test_spec.rb"], "test_spec.rb runtime should be extracted from example")
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
