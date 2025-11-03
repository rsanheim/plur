package main

import (
	"testing"

	"github.com/rsanheim/plur/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTaskExists(t *testing.T) {
	// Create a PlurCLI with some custom tasks
	cli := &PlurCLI{
		Tasks: map[string]*task.Task{
			"custom": {
				Name: "custom",
				Run:  "echo custom",
			},
			"lint": {
				Name: "lint",
				Run:  "rubocop",
			},
		},
	}

	t.Run("auto-detected task always passes", func(t *testing.T) {
		err := cli.validateTaskExists("rspec", false)
		assert.NoError(t, err)

		err = cli.validateTaskExists("nonexistent", false)
		assert.NoError(t, err, "auto-detected tasks should never error")
	})

	t.Run("explicit built-in tasks pass", func(t *testing.T) {
		err := cli.validateTaskExists("rspec", true)
		assert.NoError(t, err)

		err = cli.validateTaskExists("minitest", true)
		assert.NoError(t, err)
	})

	t.Run("explicit custom tasks pass", func(t *testing.T) {
		err := cli.validateTaskExists("custom", true)
		assert.NoError(t, err)

		err = cli.validateTaskExists("lint", true)
		assert.NoError(t, err)
	})

	t.Run("explicit non-existent task fails", func(t *testing.T) {
		err := cli.validateTaskExists("nonexistent", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task 'nonexistent' not found")
		assert.Contains(t, err.Error(), "rspec")
		assert.Contains(t, err.Error(), "minitest")
		assert.Contains(t, err.Error(), "custom")
		assert.Contains(t, err.Error(), "lint")
	})

	t.Run("empty task name with explicit flag fails", func(t *testing.T) {
		err := cli.validateTaskExists("", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGetTaskWithOverrides(t *testing.T) {
	t.Run("built-in rspec task", func(t *testing.T) {
		cli := &PlurCLI{
			Tasks: map[string]*task.Task{},
		}

		result := cli.getTaskWithOverrides("rspec")
		require.NotNil(t, result)
		assert.Equal(t, "rspec", result.Name)
		assert.Equal(t, "bundle exec rspec", result.Run)
	})

	t.Run("built-in minitest task", func(t *testing.T) {
		cli := &PlurCLI{
			Tasks: map[string]*task.Task{},
		}

		result := cli.getTaskWithOverrides("minitest")
		require.NotNil(t, result)
		assert.Equal(t, "minitest", result.Name)
	})

	t.Run("override built-in rspec with custom run", func(t *testing.T) {
		cli := &PlurCLI{
			Tasks: map[string]*task.Task{
				"rspec": {
					Name: "rspec",
					Run:  "bin/rspec",
				},
			},
		}

		result := cli.getTaskWithOverrides("rspec")
		require.NotNil(t, result)
		assert.Equal(t, "rspec", result.Name, "should preserve built-in task name")
		assert.Equal(t, "bin/rspec", result.Run, "should use custom run command")
	})

	t.Run("custom task inherits defaults but preserves name", func(t *testing.T) {
		cli := &PlurCLI{
			Tasks: map[string]*task.Task{
				"watch": {
					Name: "watch",
					Run:  "bin/rspec",
					// Sparse config - should inherit source dirs from auto-detected base
				},
			},
		}

		result := cli.getTaskWithOverrides("watch")
		require.NotNil(t, result)
		assert.Equal(t, "watch", result.Name, "should preserve custom task name, not show auto-detected 'rspec'")
		assert.Equal(t, "bin/rspec", result.Run, "should use custom run command")
		assert.NotEmpty(t, result.SourceDirs, "should inherit default RSpec source dirs")
		assert.Equal(t, "spec/**/*_spec.rb", result.TestGlob, "should inherit default RSpec test glob")
	})

	t.Run("custom task with minimal config inherits defaults", func(t *testing.T) {
		cli := &PlurCLI{
			Tasks: map[string]*task.Task{
				"lint": {
					Name: "lint",
					Run:  "rubocop",
					// No other config - should inherit from auto-detected base
				},
			},
		}

		result := cli.getTaskWithOverrides("lint")
		require.NotNil(t, result)
		assert.Equal(t, "lint", result.Name, "should preserve custom task name")
		assert.Equal(t, "rubocop", result.Run, "should use custom run command")
		assert.NotEmpty(t, result.SourceDirs, "should inherit default source dirs")
	})
}
