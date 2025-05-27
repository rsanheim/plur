package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCreateApp(t *testing.T) {
	app := createApp()

	// Test basic app properties
	if app.Name != "rux" {
		t.Errorf("Expected app name 'rux', got '%s'", app.Name)
	}

	if !strings.Contains(app.Usage, "test runner") {
		t.Errorf("Expected usage to mention 'test runner', got '%s'", app.Usage)
	}

	// Test that expected flags exist
	expectedFlags := []string{"dry-run", "auto", "json", "workers"}
	for _, flagName := range expectedFlags {
		found := false
		for _, flag := range app.Flags {
			if strings.Contains(flag.String(), flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected flag '%s' not found", flagName)
		}
	}
}

func TestRuxHelpOutput(t *testing.T) {
	app := createApp()

	// Capture help output
	var buf bytes.Buffer
	app.Writer = &buf

	err := app.Run([]string{"rux", "--help"})
	if err != nil {
		t.Fatalf("Failed to run rux --help: %v", err)
	}

	output := buf.String()

	// Check for expected help content
	expectedContent := []string{
		"rux",
		"USAGE",
		"GLOBAL OPTIONS",
		"--workers",
		"--dry-run",
		"--auto",
		"--json",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing expected content: %s", expected)
		}
	}
}
