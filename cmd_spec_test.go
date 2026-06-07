package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmatchedCLIExcludeWarningsUsesGoQuotes(t *testing.T) {
	warnings := unmatchedCLIExcludeWarnings(
		[]string{"spec/system/**/*_spec.rb"},
		map[string]int{"spec/system/**/*_spec.rb": 0},
	)

	assert.Equal(t, []string{`--exclude-pattern "spec/system/**/*_spec.rb" matched no selected files`}, warnings)
}

func TestExplicitTargetMismatchWarningsUsesGoQuotes(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	require.NoError(t, os.MkdirAll("spec", 0o755))
	require.NoError(t, os.WriteFile(filepath.Join("spec", "helper.rb"), []byte(""), 0o644))

	warnings, err := explicitTargetMismatchWarnings(
		[]string{"spec/helper.rb"},
		[]string{"spec/**/*_spec.rb"},
		"rspec",
	)
	require.NoError(t, err)

	assert.Equal(t, []string{`target "spec/helper.rb" does not match selected job "rspec" target pattern "spec/**/*_spec.rb"`}, warnings)
}
