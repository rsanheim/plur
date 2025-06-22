package main

import (
	"github.com/rsanheim/rux/minitest"
)

// MinitestCommandBuilder builds commands for running Minitest tests
type MinitestCommandBuilder struct{}

// BuildCommand constructs the Minitest command arguments
func (m *MinitestCommandBuilder) BuildCommand(files []string, config *Config) []string {
	// For minitest, we could use config.Command if it's set to something like "bundle exec ruby"
	// Otherwise default to "ruby -Itest"
	
	options := minitest.BuildOptions{
		Verbose:     false, // Don't use verbose mode for now, as per our plan
		TestOptions: []string{},
		UseBundler:  true,  // Use bundle exec by default
	}
	
	// Minitest doesn't have built-in color control like RSpec
	// We could potentially add color support via minitest-reporters gem later
	
	return minitest.BuildCommand(files, options)
}