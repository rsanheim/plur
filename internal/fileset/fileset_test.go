package fileset

import (
	"os"
	"path"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
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

func resolveJob(t *testing.T, j framework.Job) framework.Job {
	t.Helper()
	resolved, err := j.ResolveFramework()
	require.NoError(t, err)
	return resolved
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

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	discovery, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/a_spec.rb",
		"spec/m_spec.rb",
		"spec/sub/x_spec.rb",
		"spec/z_spec.rb",
	}, discovery.Files, "no inputs => framework patterns drive discovery, sorted")
}

func TestDiscover_PlainFilePassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")
	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})

	discovery, err := Discover(j, []string{"spec/foo_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/foo_spec.rb"}, discovery.Files)
}

func TestDiscover_PlainFileMissingErrors(t *testing.T) {
	discoverChdir(t)
	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})

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

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/models"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/models/post_spec.rb",
		"spec/models/user_spec.rb",
	}, discovery.Files)
}

func TestDiscover_GlobPassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/a_spec.rb",
		"spec/b_spec.rb",
		"spec/sub/c_spec.rb",
	)

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/a_spec.rb", "spec/b_spec.rb"}, discovery.Files)
}

func TestDiscover_MixedInputs(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/a_spec.rb",
		"spec/models/user_spec.rb",
		"spec/sub/d_spec.rb",
	)

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/a_spec.rb", "spec/models", "spec/sub/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/a_spec.rb",
		"spec/models/user_spec.rb",
		"spec/sub/d_spec.rb",
	}, discovery.Files)
}

func TestDiscover_ExcludesFiltering(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/system/login_spec.rb",
		"spec/system/checkout_spec.rb",
		"spec/legacy/old_spec.rb",
		"spec/models/user_spec.rb",
	)

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	discovery, err := Discover(j, nil, []string{
		"spec/system/**/*_spec.rb",
		"spec/legacy/**/*_spec.rb",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/models/user_spec.rb"}, discovery.Files)
}

func TestDiscover_InvalidExcludePatternErrors(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
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

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	first, err := Discover(j, nil, nil)
	require.NoError(t, err)
	second, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestDiscover_EmptyResultIsOk(t *testing.T) {
	discoverChdir(t)

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
	discovery, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, discovery.Files)
}

func TestDiscover_PassthroughJobWithExplicitFile(t *testing.T) {
	// Regression: a passthrough job has no target_pattern. When the user
	// passes an explicit file we must not call into the framework's target
	// pattern lookup, since passthrough has no detect patterns.
	discoverChdir(t)
	writeStubFiles(t, "spec/calculator_spec.rb")

	j := resolveJob(t, framework.Job{Name: "lint", FrameworkName: "passthrough"})
	discovery, err := Discover(j, []string{"spec/calculator_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/calculator_spec.rb"}, discovery.Files)
}

func TestDiscover_FileLinePassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/foo_spec.rb:12"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/foo_spec.rb:12"}, discovery.Files)
}

func TestDiscover_ExcludePatternMatchesUnderlyingFileForFileLineTarget(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb")

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/foo_spec.rb:12"}, []string{"spec/foo_spec.rb"})
	require.NoError(t, err)
	assert.Empty(t, discovery.Files)
}

func TestDiscover_MultiFileLinePassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/slow_spec.rb")

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/slow_spec.rb:12:38:91"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/slow_spec.rb:12:38:91"}, discovery.Files)
}

func TestDiscover_FileLineNonExistentFileErrors(t *testing.T) {
	discoverChdir(t)

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	_, err := Discover(j, []string{"spec/missing_spec.rb:12"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestDiscover_FullTreePatternPrunesIgnoredDirs(t *testing.T) {
	// Regression: go-test's full-tree pattern "**/*_test.go" must not hand
	// vendored/generated test files to workers. Only the real file is returned.
	discoverChdir(t)
	writeStubFiles(t,
		"pkg/real_test.go",
		"node_modules/junk/fake_test.go",
		"vendor/dep/dep_test.go",
		"tmp/scratch_test.go",
		".git/hooks/hook_test.go",
	)

	j := framework.Job{Name: "go-test", TargetPattern: "**/*_test.go"}
	discovery, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/real_test.go"}, discovery.Files)
}

func TestDiscover_PatternRootedAtIgnoredDirIsStillSearched(t *testing.T) {
	// A pattern whose own base is an ignored directory must still match its
	// files; only descent into ignored dirs below the base is pruned.
	discoverChdir(t)
	writeStubFiles(t, "vendor/specs/user_spec.rb")

	j := framework.Job{Name: "rspec", TargetPattern: "vendor/**/*_spec.rb"}
	discovery, err := Discover(j, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"vendor/specs/user_spec.rb"}, discovery.Files)
}

func TestDiscover_DedupsAcrossInputs(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/a_spec.rb")

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	discovery, err := Discover(j, []string{"spec/a_spec.rb", "spec/a_spec.rb", "spec/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/a_spec.rb"}, discovery.Files)
}
