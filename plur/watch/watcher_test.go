package watch

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectories
	for _, d := range []string{"lib", "lib/foo", "lib/bar", "spec", "app", "app/models"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, d), 0755))
	}

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, []string{}},
		{"single", []string{"lib"}, []string{"lib"}},
		{"root subsumes all", []string{".", "lib", "spec"}, []string{"."}},
		{"siblings preserved", []string{"lib", "spec", "app"}, []string{"app", "lib", "spec"}},
		{"nested filtered", []string{"lib", "lib/foo", "lib/bar"}, []string{"lib"}},
		{"mixed", []string{"app", "app/models", "lib", "spec"}, []string{"app", "lib", "spec"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterDirectories(tt.input)
			require.NoError(t, err)
			sort.Strings(result)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterDirectories_SymlinkDedup(t *testing.T) {
	tmpDir := t.TempDir()

	realLib := filepath.Join(tmpDir, "real_lib")
	require.NoError(t, os.MkdirAll(realLib, 0755))

	// Use relative symlink (not absolute) so os.Root accepts it
	symLib := filepath.Join(tmpDir, "lib")
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	require.NoError(t, os.Symlink("real_lib", symLib))

	// Both point to same location - should keep only one
	result, err := FilterDirectories([]string{"lib", "real_lib"})
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestFilterDirectories_RejectsEscapingSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create symlink pointing outside the project (to /tmp or similar)
	escapeLink := filepath.Join(tmpDir, "escape")
	require.NoError(t, os.Symlink("/tmp", escapeLink))

	// Create a valid directory too
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "valid"), 0755))

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// "escape" symlink should be rejected, "valid" should remain
	result, err := FilterDirectories([]string{"escape", "valid"})
	require.NoError(t, err)
	assert.Equal(t, []string{"valid"}, result)
}

func TestFilterDirectories_NonexistentDirSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only one valid directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "exists"), 0755))

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// "nonexistent" should be skipped, "exists" should remain
	result, err := FilterDirectories([]string{"nonexistent", "exists"})
	require.NoError(t, err)
	assert.Equal(t, []string{"exists"}, result)
}

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		patterns []string
		expected bool
	}{
		{
			name:     "matches .git directory",
			path:     ".git/objects/pack/abc123",
			patterns: []string{".git/**"},
			expected: true,
		},
		{
			name:     "matches node_modules",
			path:     "node_modules/lodash/index.js",
			patterns: []string{"node_modules/**"},
			expected: true,
		},
		{
			name:     "matches nested node_modules",
			path:     "packages/api/node_modules/express/lib/router.js",
			patterns: []string{"**/node_modules/**"},
			expected: true,
		},
		{
			name:     "does not match regular file",
			path:     "lib/user.rb",
			patterns: []string{".git/**", "node_modules/**"},
			expected: false,
		},
		{
			name:     "does not match spec file",
			path:     "spec/lib/user_spec.rb",
			patterns: []string{".git/**", "node_modules/**"},
			expected: false,
		},
		{
			name:     "empty patterns ignores nothing",
			path:     ".git/config",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "matches vendor directory",
			path:     "vendor/bundle/ruby/gems/rails/lib/rails.rb",
			patterns: []string{"vendor/**"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIgnored(tt.path, tt.patterns)
			assert.Equal(t, tt.expected, result, "path: %q, patterns: %v", tt.path, tt.patterns)
		})
	}
}

func TestDefaultIgnorePatterns(t *testing.T) {
	// Verify defaults are sensible
	assert.Contains(t, DefaultIgnorePatterns, ".git/**")
	assert.Contains(t, DefaultIgnorePatterns, "node_modules/**")
	assert.Len(t, DefaultIgnorePatterns, 2)

	// Verify they actually work
	assert.True(t, IsIgnored(".git/config", DefaultIgnorePatterns))
	assert.True(t, IsIgnored("node_modules/lodash/index.js", DefaultIgnorePatterns))
	assert.False(t, IsIgnored("lib/user.rb", DefaultIgnorePatterns))
}
