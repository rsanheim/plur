package main

import (
	"fmt"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml"
)

// ValidateTOMLFiles checks if TOML files are valid before Kong processes them
// If a file has invalid TOML, it temporarily renames it to prevent Kong from panicking
func ValidateTOMLFiles(paths ...string) (cleanup func()) {
	var renamedFiles []struct {
		original string
		backup   string
	}

	// Return a cleanup function to restore files
	cleanup = func() {
		for _, rf := range renamedFiles {
			os.Rename(rf.backup, rf.original)
		}
	}

	for _, path := range paths {
		originalPath := path

		// Expand ~ to home directory
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			path = home + path[1:]
		}

		// Check if file exists
		if _, err := os.Stat(path); err != nil {
			continue // File doesn't exist, skip
		}

		// Try to read and parse the file
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Try to parse TOML
		var config map[string]interface{}
		if err := toml.Unmarshal(data, &config); err != nil {
			// Invalid TOML - temporarily rename the file
			backupPath := path + ".invalid"
			fmt.Fprintf(os.Stderr, "Warning: Invalid TOML in %s: %v\n", originalPath, err)
			fmt.Fprintf(os.Stderr, "Temporarily disabling invalid config file. Using defaults.\n")

			// Rename the file
			if renameErr := os.Rename(path, backupPath); renameErr == nil {
				renamedFiles = append(renamedFiles, struct {
					original string
					backup   string
				}{path, backupPath})
			}
		}
	}

	return cleanup
}
