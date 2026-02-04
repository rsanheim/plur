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

func (c *ConfigInitCmd) Run(parent *ConfigCmd, globals *PlurCLI) error {
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
# See https://github.com/rsanheim/plur/blob/main/docs/configuration.md

# Number of parallel workers
workers = 4
use = "rspec"

[job.rspec]
cmd = ["bundle", "exec", "rspec"]

[[watch]]
name = "spec-files"
source = "spec/**/*_spec.rb"
jobs = ["rspec"]
`

const railsConfigTemplate = `# Plur configuration for Rails applications
# See https://github.com/rsanheim/plur/blob/main/docs/configuration.md

# Use more workers for Rails apps
workers = 8
color = true
use = "rspec"

[job.rspec]
cmd = ["bin/rspec"]

[[watch]]
name = "spec-files"
source = "spec/**/*_spec.rb"
jobs = ["rspec"]
`

const minitestConfigTemplate = `# Plur configuration for Minitest projects
# See https://github.com/rsanheim/plur/blob/main/docs/configuration.md

# Standard worker configuration
workers = 4
use = "minitest"

[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest"]

[[watch]]
name = "test-files"
source = "test/**/*_test.rb"
jobs = ["minitest"]
`
