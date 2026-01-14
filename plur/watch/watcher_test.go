package watch

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/rsanheim/plur/job"
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

func TestExecuteJob_BatchesMultipleTargets(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	// Job that writes all arguments to a file - verifies batching behavior
	j := job.Job{
		Name: "test-batch",
		Cmd:  []string{"sh", "-c", "echo \"$@\" > " + outputFile, "--", "{{target}}"},
	}

	err := ExecuteJob(j, []string{"file1.rb", "file2.rb", "file3.rb"}, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// All three files should appear in a single command invocation
	output := string(content)
	assert.Contains(t, output, "file1.rb")
	assert.Contains(t, output, "file2.rb")
	assert.Contains(t, output, "file3.rb")
}

func TestExecuteJob_SingleTarget(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	j := job.Job{
		Name: "test-single",
		Cmd:  []string{"sh", "-c", "echo \"$@\" > " + outputFile, "--", "{{target}}"},
	}

	err := ExecuteJob(j, []string{"only_file.rb"}, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "only_file.rb")
}

func TestExecuteJob_NoTargets(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "args.txt")

	j := job.Job{
		Name: "test-empty",
		Cmd:  []string{"sh", "-c", "echo ran > " + outputFile, "--", "{{target}}"},
	}

	err := ExecuteJob(j, []string{}, tmpDir)
	require.NoError(t, err)

	// Command should not run at all with empty targets
	_, err = os.ReadFile(outputFile)
	assert.True(t, os.IsNotExist(err), "Command should not execute with no targets")
}

func TestExecuteJob_WithoutTargetPlaceholder(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "ran.txt")

	// Job without {{target}} runs once regardless of targets
	j := job.Job{
		Name: "test-no-placeholder",
		Cmd:  []string{"sh", "-c", "echo executed > " + outputFile},
	}

	err := ExecuteJob(j, []string{"ignored1.rb", "ignored2.rb"}, tmpDir)
	require.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, "executed\n", string(content))
}

// Channel safety tests

func TestWatcher_StopIsIdempotent(t *testing.T) {
	config := &WatcherConfig{
		Directory: ".",
	}
	w := NewWatcher(config, "/nonexistent/binary")

	// Stop without Start should not panic
	w.Stop()

	// Calling Stop again should also not panic
	w.Stop()
	w.Stop()
}

func TestWatcher_StopBeforeStart(t *testing.T) {
	config := &WatcherConfig{
		Directory: ".",
	}
	w := NewWatcher(config, "/nonexistent/binary")

	// Stop before Start should complete immediately, not block
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good - Stop returned
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Stop() blocked forever when called before Start()")
	}
}

func TestWatcherManager_StopIsIdempotent(t *testing.T) {
	config := &ManagerConfig{
		Directories: []string{"."},
	}
	wm := NewWatcherManager(config, "/nonexistent/binary")

	// Stop without Start should not panic
	wm.Stop()

	// Calling Stop again should also not panic
	wm.Stop()
	wm.Stop()
}

func TestWatcherManager_AggregateEventsReturnsOnClosedWatcherChannels(t *testing.T) {
	wm := &WatcherManager{
		eventChan: make(chan Event, 1),
		errorChan: make(chan error, 1),
		stopChan:  make(chan struct{}),
	}
	defer close(wm.stopChan)

	w := &Watcher{
		eventChan: make(chan Event),
		errorChan: make(chan error),
	}
	close(w.eventChan)
	close(w.errorChan)

	wm.wg.Add(1)
	go wm.aggregateEvents(w)

	done := make(chan struct{})
	go func() {
		wm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("aggregateEvents did not return after watcher channels closed")
	}

	select {
	case event := <-wm.eventChan:
		t.Fatalf("unexpected event forwarded from closed watcher channel: %+v", event)
	default:
	}

	select {
	case err := <-wm.errorChan:
		t.Fatalf("unexpected error forwarded from closed watcher channel: %v", err)
	default:
	}
}
