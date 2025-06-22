package main

import (
	"strings"

	"github.com/rsanheim/rux/minitest"
)

// CommandBuilder is an interface for building test framework commands
type CommandBuilder interface {
	// BuildCommand constructs the command arguments for running tests
	BuildCommand(files []string, config *Config) []string
}

// NewCommandBuilder creates the appropriate command builder based on the framework
func NewCommandBuilder(framework TestFramework) CommandBuilder {
	switch framework {
	case FrameworkMinitest:
		return &MinitestCommandBuilder{}
	case FrameworkRSpec:
		fallthrough
	default:
		return &RSpecCommandBuilder{}
	}
}

// RSpecCommandBuilder builds commands for running RSpec tests
type RSpecCommandBuilder struct{}

// BuildCommand constructs the RSpec command arguments
func (r *RSpecCommandBuilder) BuildCommand(files []string, config *Config) []string {
	// Split the command string into parts
	args := strings.Fields(config.SpecCommand)

	// Add formatter arguments
	args = append(args, "-r", config.ConfigPaths.JSONRowsFormatter, "--format", "Rux::JsonRowsFormatter")

	// Add color flags based on preference
	if !config.ColorOutput {
		args = append(args, "--no-color")
	} else {
		args = append(args, "--force-color", "--tty")
	}

	args = append(args, files...)
	return args
}

// MinitestCommandBuilder builds commands for running Minitest tests
type MinitestCommandBuilder struct{}

// BuildCommand constructs the Minitest command arguments
func (m *MinitestCommandBuilder) BuildCommand(files []string, config *Config) []string {
	// For minitest, we could use config.Command if it's set to something like "bundle exec ruby"
	// Otherwise default to "ruby -Itest"

	options := minitest.BuildOptions{
		Verbose:     false, // Don't use verbose mode for now, as per our plan
		TestOptions: []string{},
		UseBundler:  true, // Use bundle exec by default
	}

	// Minitest doesn't have built-in color control like RSpec
	// We could potentially add color support via minitest-reporters gem later

	return minitest.BuildCommand(files, options)
}
