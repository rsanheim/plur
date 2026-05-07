package main

import (
	"os"
	"sort"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chdirToTempDir is a small helper for the tests below.
func chdirToTempDir(t *testing.T) string {
	t.Helper()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))
	return tempDir
}

func writeSpecFiles(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		require.NoError(t, os.MkdirAll(dirOf(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(""), 0o644))
	}
}

func TestApplyExcludesDropsMatchingFiles(t *testing.T) {
	// Input order is preserved so callers can choose their own sort.
	files := []string{
		"spec/controllers/api_spec.rb",
		"spec/models/user_spec.rb",
		"spec/system/checkout_spec.rb",
		"spec/system/login_spec.rb",
	}
	out, err := applyExcludes(files, []string{"spec/system/**/*_spec.rb"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/controllers/api_spec.rb",
		"spec/models/user_spec.rb",
	}, out)
}

func TestApplyExcludesNoPatternsIsPassthrough(t *testing.T) {
	files := []string{"b.rb", "a.rb"}
	out, err := applyExcludes(files, nil)
	require.NoError(t, err)
	// passthrough preserves order
	assert.Equal(t, []string{"b.rb", "a.rb"}, out)
}

func TestApplyExcludesMultiplePatternsOrTogether(t *testing.T) {
	files := []string{
		"spec/system/a_spec.rb",
		"spec/legacy/b_spec.rb",
		"spec/models/c_spec.rb",
	}
	out, err := applyExcludes(files, []string{
		"spec/system/**/*_spec.rb",
		"spec/legacy/**/*_spec.rb",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/models/c_spec.rb"}, out)
}

func TestApplyExcludesInvalidPatternReturnsError(t *testing.T) {
	_, err := applyExcludes([]string{"a.rb"}, []string{"spec/[unclosed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exclude pattern")
}

func TestApplyExcludesEmptyResultStillNoError(t *testing.T) {
	files := []string{"spec/foo_spec.rb"}
	out, err := applyExcludes(files, []string{"spec/**/*_spec.rb"})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestFindFilesFromJobIsDeterministic(t *testing.T) {
	chdirToTempDir(t)
	writeSpecFiles(t,
		"spec/z_spec.rb",
		"spec/a_spec.rb",
		"spec/m_spec.rb",
		"spec/sub/x_spec.rb",
	)

	rspecJob := job.Job{
		Name:          "rspec",
		TargetPattern: "spec/**/*_spec.rb",
	}

	first, err := FindFilesFromJob(rspecJob)
	require.NoError(t, err)
	second, err := FindFilesFromJob(rspecJob)
	require.NoError(t, err)

	expected := []string{
		"spec/a_spec.rb",
		"spec/m_spec.rb",
		"spec/sub/x_spec.rb",
		"spec/z_spec.rb",
	}
	assert.Equal(t, expected, first, "FindFilesFromJob output should be sorted")
	assert.Equal(t, first, second, "FindFilesFromJob output should be deterministic across calls")
}

func TestExpandPatternsFromJobIsDeterministic(t *testing.T) {
	chdirToTempDir(t)
	writeSpecFiles(t,
		"spec/z_spec.rb",
		"spec/a_spec.rb",
		"spec/m_spec.rb",
	)

	j := job.Job{Name: "rspec", Framework: "rspec"}
	first, err := ExpandPatternsFromJob([]string{"spec"}, j)
	require.NoError(t, err)
	sortedCopy := append([]string{}, first...)
	sort.Strings(sortedCopy)
	assert.Equal(t, sortedCopy, first, "ExpandPatternsFromJob output should be sorted")

	second, err := ExpandPatternsFromJob([]string{"spec"}, j)
	require.NoError(t, err)
	assert.Equal(t, first, second, "ExpandPatternsFromJob output should be deterministic across calls")
}
