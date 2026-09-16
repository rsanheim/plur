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
	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})

	files, err := Discover(j, []string{"spec/foo_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/foo_spec.rb"}, files)
}

func TestDiscover_MissingFilePassthrough(t *testing.T) {
	discoverChdir(t)
	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})

	files, err := Discover(j, []string{"does_not_exist.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"does_not_exist.rb"}, files)
}

func TestDiscover_DirectoryExpansion(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/models/user_spec.rb",
		"spec/models/post_spec.rb",
		"spec/models/readme.txt",
	)

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	files, err := Discover(j, []string{"spec/models"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"spec/models/post_spec.rb",
		"spec/models/user_spec.rb",
	}, files)
}

func TestDiscover_GlobDirectories(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/top_spec.rb", "spec/models/user_spec.rb", "spec/models/nested/post_spec.rb", "spec/[id]/route_spec.rb")
	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	all := []string{"spec/[id]/route_spec.rb", "spec/models/nested/post_spec.rb", "spec/models/user_spec.rb", "spec/top_spec.rb"}
	for _, tc := range []struct {
		name                   string
		inputs, excludes, want []string
	}{
		{"mixed files and directories", []string{"spec/*"}, nil, all},
		{"overlapping recursive matches", []string{"spec/**"}, nil, all},
		{"only directories", []string{"spec/m*"}, nil, []string{"spec/models/nested/post_spec.rb", "spec/models/user_spec.rb"}},
		{"excludes after expansion", []string{"spec/*"}, []string{"spec/models/**"}, []string{"spec/[id]/route_spec.rb", "spec/top_spec.rb"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, err := Discover(j, tc.inputs, tc.excludes)
			require.NoError(t, err)
			assert.Equal(t, tc.want, files)
		})
	}
}

func TestDiscover_GlobPassthrough(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t,
		"spec/a_spec.rb",
		"spec/b_spec.rb",
		"spec/sub/c_spec.rb",
	)

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
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

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
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

	j := framework.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}
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

	j := resolveJob(t, framework.Job{Name: "lint", FrameworkName: "passthrough"})
	files, err := Discover(j, []string{"spec/calculator_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/calculator_spec.rb"}, files)
}

func TestDiscover_SelectorsAndExclusions(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/foo_spec.rb", "spec/bar_spec.rb")
	for _, tc := range []struct {
		name, framework        string
		inputs, excludes, want []string
	}{
		{"line", "rspec", []string{"spec/foo_spec.rb:12"}, nil, []string{"spec/foo_spec.rb:12"}},
		{"multiple lines", "rspec", []string{"spec/foo_spec.rb:12:38"}, nil, []string{"spec/foo_spec.rb:12:38"}},
		{"missing file", "rspec", []string{"spec/missing_spec.rb:12"}, nil, []string{"spec/missing_spec.rb:12"}},
		{"scoped example", "rspec", []string{"spec/foo_spec.rb[1:2]"}, nil, []string{"spec/foo_spec.rb[1:2]"}},
		{"scoped group", "rspec", []string{"spec/foo_spec.rb[1]"}, nil, []string{"spec/foo_spec.rb[1]"}},
		{"multiple IDs", "rspec", []string{"spec/foo_spec.rb[1:2, 1:3]"}, nil, []string{"spec/foo_spec.rb[1:2, 1:3]"}},
		{"exclude lines", "rspec", []string{"spec/missing_spec.rb:12:38"}, []string{"spec/missing_spec.rb"}, nil},
		{"exclude IDs", "rspec", []string{"spec/foo_spec.rb[1:2,1:3]"}, []string{"spec/foo_spec.rb"}, nil},
		{"separate glob and selector inputs", "rspec", []string{"spec/[b]*_spec.rb", "spec/foo_spec.rb[1:2]"}, nil, []string{"spec/bar_spec.rb", "spec/foo_spec.rb[1:2]"}},
		{"generic colon filename", "passthrough", []string{"notes:123"}, []string{"notes:123"}, nil},
		{"generic glob", "passthrough", []string{"spec/foo_spec.rb[1:2]"}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j := resolveJob(t, framework.Job{Name: tc.framework, FrameworkName: tc.framework})
			files, err := Discover(j, tc.inputs, tc.excludes)
			require.NoError(t, err)
			if len(tc.want) == 0 {
				assert.Empty(t, files)
			} else {
				assert.Equal(t, tc.want, files)
			}
		})
	}
}

func TestDiscover_DedupsAcrossInputs(t *testing.T) {
	discoverChdir(t)
	writeStubFiles(t, "spec/a_spec.rb")

	j := resolveJob(t, framework.Job{Name: "rspec", FrameworkName: "rspec"})
	files, err := Discover(j, []string{"spec/a_spec.rb", "spec/a_spec.rb", "spec/*_spec.rb"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"spec/a_spec.rb"}, files)
}
