package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// getRuxCacheDir returns the rux cache directory (~/.cache/rux)
func getRuxCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cache", "rux"), nil
}
