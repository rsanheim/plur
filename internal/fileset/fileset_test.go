package fileset

import (
	"os"
	"path"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discoverChdir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	return tempDir
}

func writeStubFiles(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		require.NoError(t, os.MkdirAll(path.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(""), 0o644))
	}
}

func TestDiscover_NoInputsUsesFrameworkPatterns(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/z_spec.rb",
		"spec/a_spec.rb",
		"spec/m_spec.rb",
		"spec/sub/x_spec.rb",
		"spec/README.md",
	)

	j := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	files, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/a_spec.rb",
		"spec/m_spec.rb",
		"spec/sub/x_spec.rb",
		"spec/z_spec.rb",
	}, files, "no inputs => framework patterns drive discovery, sorted")
}

func TestDiscover_PlainFilePassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")
	j := job.Job{Name: "rspec", Framework: "rspec"}

	files, err := Discover(j, []string{"spec/foo_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/foo_spec.rb"}, files)
}

func TestDiscover_PlainFileMissingErrors(t *testing.T) {
	discoverChdir(t)
	j := job.Job{Name: "rspec", Framework: "rspec"}

	_, err := Discover(j, []string{"does_not_exist.rb"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestDiscover_DirectoryExpansion(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/models/user_spec.rb",
		"spec/models/post_spec.rb",
		"spec/models/readme.txt",
	)

	j := job.Job{Name: "rspec", Framework: "rspec"}
	files, err := Discover(j, []string{"spec/models"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/models/post_spec.rb",
		"spec/models/user_spec.rb",
	}, files)
}

func TestDiscover_GlobPassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/a_spec.rb",
		"spec/b_spec.rb",
		"spec/sub/c_spec.rb",
	)

	j := job.Job{Name: "rspec", Framework: "rspec"}
	files, err := Discover(j, []string{"spec/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/a_spec.rb", "spec/b_spec.rb"}, files)
}

func TestDiscover_MixedInputs(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/a_spec.rb",
		"spec/models/user_spec.rb",
		"spec/sub/d_spec.rb",
	)

	j := job.Job{Name: "rspec", Framework: "rspec"}
	files, err := Discover(j, []string{"spec/a_spec.rb", "spec/models", "spec/sub/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/a_spec.rb",
		"spec/models/user_spec.rb",
		"spec/sub/d_spec.rb",
	}, files)
}

func TestDiscover_ExcludesFiltering(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/system/login_spec.rb",
		"spec/system/checkout_spec.rb",
		"spec/legacy/old_spec.rb",
		"spec/models/user_spec.rb",
	)

	j := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	files, err := Discover(j, nil, []string{
		"spec/system/**/*_spec.rb",
		"spec/legacy/**/*_spec.rb",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/models/user_spec.rb"}, files)
}

func TestDiscover_InvalidExcludePatternErrors(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")

	j := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	_, err := Discover(j, nil, []string{"spec/[unclosed"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exclude pattern")
}

func TestDiscover_DeterministicAcrossCalls(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/z_spec.rb",
		"spec/a_spec.rb",
		"spec/m_spec.rb",
	)

	j := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	first, err := Discover(j, nil, nil)
	require.NoError(t, err)
	second, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestDiscover_EmptyResultIsOk(t *testing.T) {
	discoverChdir(t)

	j := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	files, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestDiscover_PassthroughJobWithExplicitFile(t *testing.T) {
	// Regression: a passthrough job has no target_pattern. When the user
	// passes an explicit file we must not call into the framework's target
	// pattern lookup, since passthrough has no detect patterns.
	discoverChdir(t)
	writeStubFiles(t, "spec/calculator_spec.rb")

	j := job.Job{Name: "lint", Framework: "passthrough"}
	files, err := Discover(j, []string{"spec/calculator_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/calculator_spec.rb"}, files)
}

func TestDiscover_DedupsAcrossInputs(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/a_spec.rb")

	j := job.Job{Name: "rspec", Framework: "rspec"}
	files, err := Discover(j, []string{"spec/a_spec.rb", "spec/a_spec.rb", "spec/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/a_spec.rb"}, files)
}
