package watch

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

// WatchConfig represents the watch section of the config file
type WatchConfig struct {
	Mappings MappingConfig `toml:"mappings"`
}

// LoadMappingConfig loads mapping configuration from a TOML file with framework
func LoadMappingConfig(configPath string, framework string) (*MappingConfig, error) {
	// Start with framework-specific config
	config := NewMappingConfigForFramework(framework)

	// Compile the default rules first
	if err := config.CompileRules(); err != nil {
		return nil, err
	}

	// Try to load the config file
	if configPath == "" {
		// Look for .plur.toml in current directory
		configPath = ".plur.toml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Try home directory
			home, _ := os.UserHomeDir()
			if home != "" {
				configPath = filepath.Join(home, ".plur.toml")
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					// No config file found, use defaults
					return config, nil
				}
			}
		}
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, just use defaults
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	// Parse the TOML
	var fullConfig struct {
		Watch WatchConfig `toml:"watch"`
	}

	if err := toml.Unmarshal(data, &fullConfig); err != nil {
		// If parsing fails, use defaults but don't error
		// This allows the config to have other sections we don't care about
		return config, nil
	}

	// Merge custom rules with defaults
	if len(fullConfig.Watch.Mappings.CustomRules) > 0 {
		config.CustomRules = fullConfig.Watch.Mappings.CustomRules
		// Recompile with custom rules
		if err := config.CompileRules(); err != nil {
			return nil, err
		}
	}

	// Apply settings - these should only be set if explicitly configured
	// Don't override defaults with false values from empty config
	if fullConfig.Watch.Mappings.ShowSuggestions {
		config.ShowSuggestions = fullConfig.Watch.Mappings.ShowSuggestions
	}
	if fullConfig.Watch.Mappings.ProvideFeedback {
		config.ProvideFeedback = fullConfig.Watch.Mappings.ProvideFeedback
	}

	return config, nil
}
