package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type ConfigInitCmd struct {
	Force    bool   `help:"Overwrite existing config file" default:"false"`
	Global   bool   `help:"Create global config in home directory" default:"false"`
	Template string `help:"Template to use (simple, rails, minitest)" default:"simple"`
}

func (c *ConfigInitCmd) Run(parent *PlurCLI) error {
	var configPath string
	var configContent string

	// Determine config file path
	if c.Global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".plur.toml")
	} else {
		configPath = ".plur.toml"
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !c.Force {
		return fmt.Errorf("config file %s already exists (use --force to overwrite)", configPath)
	}

	// Select template content
	switch c.Template {
	case "simple":
		configContent = simpleConfigTemplate
	case "rails":
		configContent = railsConfigTemplate
	case "minitest":
		configContent = minitestConfigTemplate
	default:
		return fmt.Errorf("unknown template: %s (available: simple, rails, minitest)", c.Template)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created %s with %s template\n", configPath, c.Template)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the config file to match your project needs")
	fmt.Println("2. Run 'plur doctor' to verify configuration")
	fmt.Println("3. Run 'plur' to execute your tests")

	return nil
}

const simpleConfigTemplate = `# Plur configuration file
# See examples/plur.toml.example for all available options

# Number of parallel workers
workers = 4

# Test command configuration
[spec]
command = "bundle exec rspec"

# Watch mode configuration
[watch.run]
command = "bundle exec rspec --fail-fast"
debounce = 200
`

const railsConfigTemplate = `# Plur configuration for Rails applications
# See examples/plur.toml.example for all available options

# Use more workers for Rails apps
workers = 8

# Enable colored output
color = true

# Rails-specific test command
[spec]
command = "bin/rspec"

# Watch mode with Spring for faster startup
[watch.run]
command = "bin/spring rspec --fail-fast --no-coverage"
debounce = 300
`

const minitestConfigTemplate = `# Plur configuration for Minitest projects
# See examples/plur.toml.example for all available options

# Standard worker configuration
workers = 4

# Minitest-specific settings
[spec]
command = "bundle exec ruby -Itest"
type = "minitest"

# Watch mode for minitest
[watch.run]
command = "bundle exec ruby -Itest"
type = "minitest"
debounce = 150
`
