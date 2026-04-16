package main

import (
	"testing"

	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
