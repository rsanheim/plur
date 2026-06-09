package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor records job executions for testing
type mockExecutor struct {
	calls []executorCall
}

type executorCall struct {
	jobName string
	targets []string
}

func (m *mockExecutor) execute(j framework.Job, targets []string, cwd string) error {
	m.calls = append(m.calls, executorCall{jobName: j.Name, targets: targets})
	return nil
}

func projectTmpDir(t *testing.T) string {
	t.Helper()

	rootTmp := filepath.Join("..", "tmp")
	require.NoError(t, os.MkdirAll(rootTmp, 0755))

	tmpDir, err := os.MkdirTemp(rootTmp, "watch-plan-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	})

	return tmpDir
}

func TestFileEventHandler_HandleBatch_EmptyWatches(t *testing.T) {
	handler := &FileEventHandler{
		Jobs:    map[string]framework.Job{},
		Watches: []WatchMapping{},
		CWD:     "/tmp",
	}

	result := handler.HandleBatch([]string{"foo.rb"})

	assert.Empty(t, result.ExecutedJobs)
	assert.False(t, result.ShouldReload)
}

func TestFileEventHandler_HandleBatch_SingleFile(t *testing.T) {
	// Create temp directory with real files
	tmpDir := t.TempDir()

	// Create source file
	srcDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "user.rb"), []byte("# user"), 0644))

	// Create target file (spec)
	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"lib/user.rb"})

	assert.Equal(t, []string{"rspec"}, result.ExecutedJobs)
	assert.False(t, result.ShouldReload)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "rspec", mock.calls[0].jobName)
	assert.Contains(t, mock.calls[0].targets[0], "user_spec.rb")
}

func TestFileEventHandler_HandleBatch_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source files
	srcDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "user.rb"), []byte("# user"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "post.rb"), []byte("# post"), 0644))

	// Create target files (specs)
	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "post_spec.rb"), []byte("# spec"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	// Handle both files in one batch
	result := handler.HandleBatch([]string{"lib/user.rb", "lib/post.rb"})

	assert.Equal(t, []string{"rspec"}, result.ExecutedJobs, "Job should only be executed once")
	require.Len(t, mock.calls, 1)
	assert.Len(t, mock.calls[0].targets, 2, "Both targets should be passed to single job execution")
}

func TestFileEventHandler_HandleBatch_TargetDeduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a single target file
	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "shared_spec.rb"), []byte("# spec"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "both-to-same-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/shared_spec.rb"}, // Same target for all
				Jobs:    []string{"rspec"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	// Two source files that map to the same target
	result := handler.HandleBatch([]string{"lib/a.rb", "lib/b.rb"})

	assert.Equal(t, []string{"rspec"}, result.ExecutedJobs)
	require.Len(t, mock.calls, 1)
	assert.Len(t, mock.calls[0].targets, 1, "Duplicate target should be deduplicated")
}

func TestFileEventHandler_HandleBatch_ShouldReload(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "settings.yml"), []byte("key: val"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{},
		Watches: []WatchMapping{
			{
				Name:   "config-reload",
				Source: "config/**/*.yml",
				Reload: true,
				Jobs:   []string{},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"config/settings.yml"})

	assert.True(t, result.ShouldReload)
}

func TestFileEventHandler_HandleBatch_NoTargetsExecutesJob(t *testing.T) {
	tmpDir := t.TempDir()

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"build": {Name: "build", Cmd: []string{"bin/rake", "install"}},
		},
		Watches: []WatchMapping{
			{
				Name:      "go-build",
				Source:    "**/*.go",
				NoTargets: true,
				Jobs:      []string{"build"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"runner.go"})

	assert.Equal(t, []string{"build"}, result.ExecutedJobs)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "build", mock.calls[0].jobName)
	assert.Empty(t, mock.calls[0].targets)
}

func TestFileEventHandler_HandleBatch_NoTargetsExecutesWhenOtherTargetsAreMissing(t *testing.T) {
	tmpDir := t.TempDir()

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"build": {Name: "build", Cmd: []string{"bin/rake", "install"}},
		},
		Watches: []WatchMapping{
			{
				Name:      "go-build",
				Source:    "**/*.go",
				NoTargets: true,
				Jobs:      []string{"build"},
			},
			{
				Name:    "generated-target",
				Source:  "**/*.go",
				Targets: []string{"generated/missing.txt"},
				Jobs:    []string{"build"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"runner.go"})

	assert.Equal(t, []string{"build"}, result.ExecutedJobs)
	require.Len(t, mock.calls, 1)
	assert.Empty(t, mock.calls[0].targets)
}

func TestFileEventHandler_HandleBatch_MissingOnlyJobDoesNotRunWhenAnotherJobHasTargets(t *testing.T) {
	tmpDir := projectTmpDir(t)

	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec":    {Name: "rspec", Cmd: []string{"rspec"}},
			"minitest": {Name: "minitest", Cmd: []string{"ruby"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "lib-to-rspec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
			{
				Name:    "lib-to-minitest",
				Source:  "lib/**/*.rb",
				Targets: []string{"test/{{match}}_test.rb"},
				Jobs:    []string{"minitest"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"lib/user.rb"})

	assert.Equal(t, []string{"rspec"}, result.ExecutedJobs)
	require.Len(t, mock.calls, 1)
	assert.Equal(t, "rspec", mock.calls[0].jobName)
	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, mock.calls[0].targets)
}

func TestFileEventHandler_HandleBatch_MultipleJobs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source and target files
	srcDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "user.rb"), []byte("# user"), 0644))

	specDir := filepath.Join(tmpDir, "spec")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "user_spec.rb"), []byte("# spec"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec":   {Name: "rspec", Cmd: []string{"rspec"}},
			"rubocop": {Name: "rubocop", Cmd: []string{"rubocop"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec", "rubocop"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"lib/user.rb"})

	assert.Equal(t, []string{"rspec", "rubocop"}, result.ExecutedJobs)
	assert.Len(t, mock.calls, 2)
}

func TestFileEventHandler_HandleBatch_NoMatchingTargets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file but NOT the target
	srcDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "user.rb"), []byte("# user"), 0644))

	mock := &mockExecutor{}
	handler := &FileEventHandler{
		Jobs: map[string]framework.Job{
			"rspec": {Name: "rspec", Cmd: []string{"rspec"}},
		},
		Watches: []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
		CWD:      tmpDir,
		Executor: mock.execute,
	}

	result := handler.HandleBatch([]string{"lib/user.rb"})

	assert.Empty(t, result.ExecutedJobs, "No jobs should run when target doesn't exist")
	assert.Len(t, mock.calls, 0)
}
