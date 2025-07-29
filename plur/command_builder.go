package main

import (
	"strings"

	"github.com/rsanheim/plur/minitest"
)

// CommandBuilder is an interface for building test framework commands
type CommandBuilder interface {
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

type RSpecCommandBuilder struct{}

// BuildCommand constructs the RSpec command arguments
func (r *RSpecCommandBuilder) BuildCommand(files []string, globalConfig *GlobalConfig, command string) []string {
	args := strings.Fields(command)
	formatterPath := globalConfig.ConfigPaths.GetJSONRowsFormatterPath()
	args = append(args, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")

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
// For minitest we currently default to 'bundle exec ruby -Itest' if UseBundler is true
// Otherwise we use 'ruby -Itest'
func (m *MinitestCommandBuilder) BuildCommand(files []string, globalConfig *GlobalConfig, command string) []string {
	options := minitest.BuildOptions{
		Verbose:     false,
		TestOptions: []string{},
		UseBundler:  true, // Use bundle exec by default
	}

	return minitest.BuildCommand(files, options)
}
