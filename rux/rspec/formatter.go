package rspec

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the formatter Ruby code directly into the binary
//
//go:embed formatter.rb
var jsonRowsFormatterCode string

// GetFormatterPath returns the path to the JSON rows formatter,
// creating it in the cache directory if it doesn't exist
func GetFormatterPath(formattersPath string) (string, error) {
	formatterPath := filepath.Join(formattersPath, "json_rows_formatter.rb")

	if existingContent, err := os.ReadFile(formatterPath); err == nil {
		if string(existingContent) == jsonRowsFormatterCode {
			return formatterPath, nil
		}
	}

	if err := os.WriteFile(formatterPath, []byte(jsonRowsFormatterCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write formatter file: %w", err)
	}

	return formatterPath, nil
}
