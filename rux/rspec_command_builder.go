package main

import "strings"

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
