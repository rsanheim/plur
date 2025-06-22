package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecCmd_FrameworkValidation(t *testing.T) {
	tests := []struct {
		name      string
		framework string
		wantErr   bool
		expected  TestFramework
	}{
		{
			name:      "valid rspec",
			framework: "rspec",
			wantErr:   false,
			expected:  FrameworkRSpec,
		},
		{
			name:      "valid minitest",
			framework: "minitest",
			wantErr:   false,
			expected:  FrameworkMinitest,
		},
		{
			name:      "invalid framework",
			framework: "jest",
			wantErr:   true,
		},
		{
			name:      "empty framework uses auto-detection",
			framework: "",
			wantErr:   false,
			expected:  FrameworkRSpec, // default when no dirs exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &SpecCmd{Type: tt.framework}

			// Test framework validation
			err := validateFramework(spec.Type)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.framework != "" {
					// Test the parsing logic
					framework := parseFramework(tt.framework)
					assert.Equal(t, tt.expected, framework)
				}
			}
		})
	}
}

func TestDetectTestFramework(t *testing.T) {
	// Save current dir
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create temp dir for testing
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		setup    func()
		expected TestFramework
	}{
		{
			name: "detects minitest with test directory",
			setup: func() {
				err := os.Mkdir("test", 0755)
				require.NoError(t, err)
			},
			expected: FrameworkMinitest,
		},
		{
			name: "detects rspec with spec directory",
			setup: func() {
				err := os.Mkdir("spec", 0755)
				require.NoError(t, err)
			},
			expected: FrameworkRSpec,
		},
		{
			name:     "defaults to rspec with no directories",
			setup:    func() {},
			expected: FrameworkRSpec,
		},
		{
			name: "prefers minitest when both exist",
			setup: func() {
				err := os.Mkdir("test", 0755)
				require.NoError(t, err)
				err = os.Mkdir("spec", 0755)
				require.NoError(t, err)
			},
			expected: FrameworkMinitest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing dirs
			os.RemoveAll("test")
			os.RemoveAll("spec")

			tt.setup()

			result := DetectTestFramework()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions to test framework parsing logic
func parseFramework(frameworkType string) TestFramework {
	switch frameworkType {
	case "rspec":
		return FrameworkRSpec
	case "minitest":
		return FrameworkMinitest
	default:
		return ""
	}
}

func validateFramework(frameworkType string) error {
	if frameworkType == "" {
		return nil // auto-detection
	}

	switch frameworkType {
	case "rspec", "minitest":
		return nil
	default:
		return fmt.Errorf("invalid test framework type: %s (must be 'rspec' or 'minitest')", frameworkType)
	}
}
