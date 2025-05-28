package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the formatter Ruby code directly into the binary
//
//go:embed lib/rux/json_rows_formatter.rb
var jsonRowsFormatterCode string

// GetFormatterPath returns the path to the JSON rows formatter,
// creating it in the XDG cache directory if it doesn't exist
func GetFormatterPath() (string, error) {
	defer TraceFunc("formatter.get_path")()
	
	// Get XDG cache directory (~/.cache on Linux/Mac)
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Create the rux formatters directory
	formattersDir := filepath.Join(cacheDir, "rux", "formatters")
	if err := os.MkdirAll(formattersDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create formatters directory: %w", err)
	}

	// Path to the formatter file
	formatterPath := filepath.Join(formattersDir, "json_rows_formatter.rb")

	// Check if formatter already exists and has the same content
	if existingContent, err := os.ReadFile(formatterPath); err == nil {
		if string(existingContent) == jsonRowsFormatterCode {
			// Formatter already exists with correct content
			return formatterPath, nil
		}
	}

	// Write the formatter to the cache directory
	func() {
		defer TraceFunc("formatter.write_file")()
		err = os.WriteFile(formatterPath, []byte(jsonRowsFormatterCode), 0644)
	}()
	if err != nil {
		return "", fmt.Errorf("failed to write formatter file: %w", err)
	}

	return formatterPath, nil
}
