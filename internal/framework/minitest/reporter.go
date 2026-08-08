package minitest

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Embed the reporter Ruby code directly into the binary
//
//go:embed reporter.rb
var reporterCode string

// GetReporterPath returns the path to the minitest reporter,
// creating it in the cache directory if it doesn't exist
func GetReporterPath(formattersPath string) (string, error) {
	reporterPath := filepath.Join(formattersPath, "minitest_reporter.rb")

	if existingContent, err := os.ReadFile(reporterPath); err == nil {
		if string(existingContent) == reporterCode {
			return reporterPath, nil
		}
	}

	if err := os.WriteFile(reporterPath, []byte(reporterCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write reporter file: %w", err)
	}

	return reporterPath, nil
}
