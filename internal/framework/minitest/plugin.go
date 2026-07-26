package minitest

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the stdout tagging plugin directly into the binary
//
//go:embed plur_plugin.rb
var plurPluginCode string

// GetPluginLoadPath writes the minitest plugin under baseDir and returns the
// directory to add to Ruby's $LOAD_PATH. Minitest finds plugins by globbing
// $LOAD_PATH for "minitest/*_plugin.rb", so the file has to sit in a minitest/
// subdirectory of the returned path.
func GetPluginLoadPath(baseDir string) (string, error) {
	loadPath := filepath.Join(baseDir, "rubylib")
	pluginPath := filepath.Join(loadPath, "minitest", "plur_plugin.rb")

	if existing, err := os.ReadFile(pluginPath); err == nil && string(existing) == plurPluginCode {
		return loadPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(pluginPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create minitest plugin directory: %w", err)
	}
	if err := os.WriteFile(pluginPath, []byte(plurPluginCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write minitest plugin file: %w", err)
	}

	return loadPath, nil
}
