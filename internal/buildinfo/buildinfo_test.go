package buildinfo_test

import (
	"regexp"
	"testing"

	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersionInfo_ReturnsValidFormat(t *testing.T) {
	version := buildinfo.GetVersionInfo()

	// Version should never be empty
	require.NotEmpty(t, version)

	// Version should match one of the expected patterns:
	// - "dev" (fallback)
	// - "dev-{commit}" (7 char hash)
	// - "dev-{commit}-dirty"
	// - semver from ldflags or module (e.g., "0.12.0-dev-c366cf1" or "v0.12.0")
	validPatterns := []string{
		`^dev$`,                   // Pure dev fallback
		`^dev-[0-9a-f]{7}$`,       // Dev with commit
		`^dev-[0-9a-f]{7}-dirty$`, // Dev with dirty flag
		`^v?\d+\.\d+\.\d+.*$`,     // Semver (with optional v prefix and any suffix)
	}

	matched := false
	for _, pattern := range validPatterns {
		if regexp.MustCompile(pattern).MatchString(version) {
			matched = true
			break
		}
	}
	assert.True(t, matched, "version %q should match one of the expected patterns", version)
}

func TestGetVersionInfo_IsConsistent(t *testing.T) {
	v1 := buildinfo.GetVersionInfo()
	v2 := buildinfo.GetVersionInfo()
	v3 := buildinfo.GetVersionInfo()

	assert.Equal(t, v1, v2)
	assert.Equal(t, v2, v3)
}

func TestGetVersionInfo_NeverEmpty(t *testing.T) {
	version := buildinfo.GetVersionInfo()
	require.NotEmpty(t, version)
}

func TestGetVersionInfo_CommitHashFormat(t *testing.T) {
	version := buildinfo.GetVersionInfo()

	// If version includes a commit hash (dev-{hash} format), verify it's valid
	devCommitPattern := regexp.MustCompile(`^dev-([0-9a-f]{7})(-dirty)?$`)
	if matches := devCommitPattern.FindStringSubmatch(version); matches != nil {
		commit := matches[1]
		assert.Len(t, commit, 7, "commit hash should be exactly 7 characters")
		assert.Regexp(t, `^[0-9a-f]+$`, commit, "commit should contain only hex characters")
	}
}
