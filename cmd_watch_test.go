package main

import (
	"testing"

	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWatchConfigurationMergesUserAndDefaultWatches(t *testing.T) {
	cli := &PlurCLI{
		WatchMappings: []watch.WatchMapping{
			{
				Name:    "custom-config-watch",
				Source:  "config/**/*.yml",
				Targets: []string{"spec/config_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
	}

	resolved, watches, err := loadWatchConfiguration(cli, "rspec")
	require.NoError(t, err)
	require.Equal(t, "rspec", resolved.Name)

	var names []string
	for _, mapping := range watches {
		names = append(names, mapping.Name)
	}

	assert.Contains(t, names, "custom-config-watch")
	assert.Contains(t, names, "lib-to-spec")
	assert.Contains(t, names, "spec-files")
}
