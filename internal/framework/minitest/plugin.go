package minitest

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the plugin Ruby code directly into the binary
//
//go:embed plur_plugin.rb
var pluginCode string

// GetPluginLoadPath writes the embedded plugin to
// <rubyLibDir>/minitest/plur_plugin.rb (creating it if missing or stale) and
// returns rubyLibDir, the directory to add to ruby's $LOAD_PATH via -I so
// minitest's plugin discovery (Gem.find_files "minitest/*_plugin.rb") finds it.
func GetPluginLoadPath(rubyLibDir string) (string, error) {
	pluginDir := filepath.Join(rubyLibDir, "minitest")
	pluginPath := filepath.Join(pluginDir, "plur_plugin.rb")

	if existingContent, err := os.ReadFile(pluginPath); err == nil {
		if string(existingContent) == pluginCode {
			return rubyLibDir, nil
		}
	}

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugin directory: %w", err)
	}
	if err := os.WriteFile(pluginPath, []byte(pluginCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write plugin file: %w", err)
	}

	return rubyLibDir, nil
}
