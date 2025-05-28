package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeTracker(t *testing.T) {
	t.Run("AddRuntime accumulates runtimes for same file", func(t *testing.T) {
		rt := NewRuntimeTracker()

		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)
		rt.AddRuntime("spec/foo_spec.rb", 0.5)

		runtimes := rt.GetRuntimes()

		if runtimes["spec/foo_spec.rb"] != 2.0 {
			t.Errorf("Expected foo_spec.rb runtime to be 2.0, got %f", runtimes["spec/foo_spec.rb"])
		}

		if runtimes["spec/bar_spec.rb"] != 2.0 {
			t.Errorf("Expected bar_spec.rb runtime to be 2.0, got %f", runtimes["spec/bar_spec.rb"])
		}
	})

	t.Run("AddExample extracts runtime from RSpec example", func(t *testing.T) {
		rt := NewRuntimeTracker()

		example := RSpecExample{
			FilePath: "spec/test_spec.rb",
			RunTime:  0.123,
		}

		rt.AddExample(example)

		runtimes := rt.GetRuntimes()
		if runtimes["spec/test_spec.rb"] != 0.123 {
			t.Errorf("Expected test_spec.rb runtime to be 0.123, got %f", runtimes["spec/test_spec.rb"])
		}
	})

	t.Run("SaveToFile creates runtime.json", func(t *testing.T) {
		rt := NewRuntimeTracker()
		rt.AddRuntime("spec/foo_spec.rb", 1.5)
		rt.AddRuntime("spec/bar_spec.rb", 2.0)

		// Create temp directory for test
		tmpDir, err := os.MkdirTemp("", "rux-runtime-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Save runtime data
		err = rt.SaveToFile(tmpDir)
		if err != nil {
			t.Errorf("Failed to save runtime data: %v", err)
		}

		// Check file exists
		runtimeFile := filepath.Join(tmpDir, "runtime.json")
		if _, err := os.Stat(runtimeFile); os.IsNotExist(err) {
			t.Error("runtime.json was not created")
		}
	})
}
