package main

import (
	"github.com/rsanheim/rux/minitest"
)

// MinitestCommandBuilder builds commands for running Minitest tests
type MinitestCommandBuilder struct{}

// BuildCommand constructs the Minitest command arguments
func (m *MinitestCommandBuilder) BuildCommand(files []string, config *Config) []string {
	// For minitest, we use ruby -Itest pattern
	// The config.SpecCommand is ignored for minitest (we could add a TestCommand field later)
	
	options := minitest.BuildOptions{
		Verbose:     false, // Don't use verbose mode for now, as per our plan
		TestOptions: []string{},
	}
	
	// Minitest doesn't have built-in color control like RSpec
	// We could potentially add color support via minitest-reporters gem later
	
	return minitest.BuildCommand(files, options)
}