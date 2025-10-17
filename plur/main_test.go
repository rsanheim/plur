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
