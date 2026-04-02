package autodetect

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinDefaultsLoad(t *testing.T) {
	// Verify that builtinDefaults were loaded in init()
	assert.NotEmpty(t, builtinDefaults.Defaults.Jobs)
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "rspec")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "minitest")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "go-test")
}

func TestAnyPatternMatches(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	os.MkdirAll("spec", 0o755)
	os.WriteFile("spec/user_spec.rb", []byte(""), 0o644)

	// Single matching pattern
	matched, err := anyPatternMatches([]string{"spec/**/*_spec.rb"})
	assert.NoError(t, err)
	assert.True(t, matched)

	// Multiple patterns, one matching
	matched, err = anyPatternMatches([]string{"nonexistent", "spec/**/*_spec.rb"})
	assert.NoError(t, err)
	assert.True(t, matched)

	// No matching patterns
	matched, err = anyPatternMatches([]string{"nonexistent", "other/*.go"})
	assert.NoError(t, err)
	assert.False(t, matched)

	// Invalid pattern should return error
	_, err = anyPatternMatches([]string{"["})
	assert.Error(t, err)
}
