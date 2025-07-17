package main

import (
	"strings"

	"github.com/rsanheim/plur/minitest"
)

// CommandBuilder is an interface for building test framework commands
type CommandBuilder interface {
	// BuildCommand constructs the command arguments for running tests
	BuildCommand(files []string, globalConfig *GlobalConfig, command string) []string
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
func (r *RSpecCommandBuilder) BuildCommand(files []string, globalConfig *GlobalConfig, command string) []string {
	// Split the command string into parts
	args := strings.Fields(command)

	// Add formatter arguments
	args = append(args, "-r", globalConfig.ConfigPaths.JSONRowsFormatter, "--format", "Plur::JsonRowsFormatter")

	// Add color flags based on preference
	if !globalConfig.ColorOutput {
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
func (m *MinitestCommandBuilder) BuildCommand(files []string, globalConfig *GlobalConfig, command string) []string {
	// For minitest, we could use command if it's set to something like "bundle exec ruby"
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
