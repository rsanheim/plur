package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindTargetsForFile_DeduplicatesExistingTargetsForSameJob(t *testing.T) {
	tmpDir := projectTmpDir(t)

	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))

	jobs := map[string]framework.Job{
		"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
	}
	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
		{
			Name:    "lib-to-same-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
	}

	result, err := FindTargetsForFile("lib/user.rb", jobs, watches, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result.ExistingTargetFiles())
}

func TestFindTargetsForFile_NoTargetsDoesNotWipeExistingTargetsForSameJob(t *testing.T) {
	tmpDir := projectTmpDir(t)

	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))

	jobs := map[string]framework.Job{
		"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
	}
	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
		{
			Name:      "lib-rspec-no-targets",
			Source:    "lib/**/*.rb",
			NoTargets: true,
			Jobs:      []string{"rspec"},
		},
	}

	result, err := FindTargetsForFile("lib/user.rb", jobs, watches, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result.ExistingTargetFiles())
}
