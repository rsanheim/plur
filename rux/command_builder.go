package main

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
