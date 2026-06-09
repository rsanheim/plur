package watch

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectories
	for _, d := range []string{"lib", "lib/foo", "lib/bar", "spec", "app", "app/models"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, d), 0755))
	}

	t.Chdir(tmpDir)

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
	t.Chdir(tmpDir)

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

	t.Chdir(tmpDir)

	// "escape" symlink should be rejected, "valid" should remain
	result, err := FilterDirectories([]string{"escape", "valid"})
	require.NoError(t, err)
	assert.Equal(t, []string{"valid"}, result)
}

func TestFilterDirectories_NonexistentDirSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only one valid directory
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "exists"), 0755))

	t.Chdir(tmpDir)

	// "nonexistent" should be skipped, "exists" should remain
	result, err := FilterDirectories([]string{"nonexistent", "exists"})
	require.NoError(t, err)
	assert.Equal(t, []string{"exists"}, result)
}

func TestDefaultIgnorePatterns(t *testing.T) {
	assert.Contains(t, DefaultIgnorePatterns, ".git/**")
	assert.Contains(t, DefaultIgnorePatterns, "node_modules/**")
	assert.Len(t, DefaultIgnorePatterns, 2)
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

	wm.wg.Go(func() {
		wm.aggregateEvents(w)
	})

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
