package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindTargetsForFile_CurrentPlanningCases(t *testing.T) {
	tmpDir := makeWatchTestProject(t)
	writeWatchTestFile(t, tmpDir, "lib/user.rb")
	writeWatchTestFile(t, tmpDir, "lib/ignored.rb")
	writeWatchTestFile(t, tmpDir, "spec/user_spec.rb")
	writeWatchTestFile(t, tmpDir, "spec/self_spec.rb")

	jobs := map[string]job.Job{
		"rspec":   {Name: "rspec", Cmd: []string{"rspec", "{{target}}"}},
		"rubocop": {Name: "rubocop", Cmd: []string{"rubocop", "{{target}}"}},
	}

	t.Run("runnable target", func(t *testing.T) {
		result, err := FindTargetsForFile("lib/user.rb", jobs, []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		}, tmpDir)
		require.NoError(t, err)

		require.Len(t, result.MatchedRules, 1)
		assert.Equal(t, "lib-to-spec", result.MatchedRules[0].Name)
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result.ExistingTargets["rspec"])
		assert.Empty(t, result.MissingTargets)
		assert.True(t, result.HasExistingTargets())
	})

	t.Run("no matching rule", func(t *testing.T) {
		result, err := FindTargetsForFile("spec/spec_helper.rb", jobs, []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		}, tmpDir)
		require.NoError(t, err)

		assert.Empty(t, result.MatchedRules)
		assert.Empty(t, result.ExistingTargets)
		assert.Empty(t, result.MissingTargets)
		assert.False(t, result.HasExistingTargets())
	})

	t.Run("missing target", func(t *testing.T) {
		result, err := FindTargetsForFile("lib/missing.rb", jobs, []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		}, tmpDir)
		require.NoError(t, err)

		require.Len(t, result.MatchedRules, 1)
		assert.Equal(t, []string{filepath.FromSlash("spec/missing_spec.rb")}, result.MissingTargets["rspec"])
		assert.False(t, result.HasExistingTargets())
		assert.True(t, result.HasMissingTargets())
	})

	t.Run("per-watch ignore", func(t *testing.T) {
		result, err := FindTargetsForFile("lib/ignored.rb", jobs, []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
				Ignore:  []string{"lib/ignored.rb"},
			},
		}, tmpDir)
		require.NoError(t, err)

		assert.Empty(t, result.MatchedRules)
		assert.Empty(t, result.ExistingTargets)
		assert.Empty(t, result.MissingTargets)
	})

	t.Run("source file target", func(t *testing.T) {
		result, err := FindTargetsForFile("spec/self_spec.rb", jobs, []WatchMapping{
			{
				Name:   "spec-files",
				Source: "spec/**/*_spec.rb",
				Jobs:   []string{"rspec"},
			},
		}, tmpDir)
		require.NoError(t, err)

		require.Len(t, result.MatchedRules, 1)
		assert.Equal(t, []string{filepath.FromSlash("spec/self_spec.rb")}, result.ExistingTargets["rspec"])
		assert.Empty(t, result.MissingTargets)
	})

	t.Run("multiple jobs", func(t *testing.T) {
		result, err := FindTargetsForFile("lib/user.rb", jobs, []WatchMapping{
			{
				Name:    "lib-to-tools",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec", "rubocop"},
			},
		}, tmpDir)
		require.NoError(t, err)

		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result.ExistingTargets["rspec"])
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result.ExistingTargets["rubocop"])
	})
}

func makeWatchTestProject(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Dir(cwd)
	if filepath.Base(cwd) != "watch" {
		root = cwd
	}
	tmpRoot := filepath.Join(root, "tmp")
	require.NoError(t, os.MkdirAll(tmpRoot, 0755))

	dir, err := os.MkdirTemp(tmpRoot, "watch-test-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(dir))
	})

	return dir
}

func writeWatchTestFile(t *testing.T, root, path string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
	require.NoError(t, os.WriteFile(fullPath, []byte("# test\n"), 0644))
}
