package embedded

import (
	"io/fs"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedWatcherName intentionally duplicates the platform mapping in
// watch.getPlatformBinaryName: if a watcher binary is renamed or a build tag
// drifts, this test fails instead of `plur watch install` at runtime.
func expectedWatcherName() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "watcher-aarch64-apple-darwin"
	case "linux/amd64":
		return "watcher-x86_64-unknown-linux-gnu"
	case "linux/arm64":
		return "watcher-aarch64-unknown-linux-gnu"
	case "windows/amd64":
		return "watcher-x86_64-pc-windows-msvc"
	default:
		return ""
	}
}

func embeddedFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := fs.WalkDir(Watcher, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	return files
}

// TestWatcherEmbedsOnlyThisPlatformsBinary guards the per-platform embed
// split: each plur binary must carry exactly one watcher (its own), not all
// four (~570kB).
func TestWatcherEmbedsOnlyThisPlatformsBinary(t *testing.T) {
	files := embeddedFiles(t)
	expected := expectedWatcherName()

	if expected == "" {
		assert.Empty(t, files, "unsupported platforms must embed no watcher binaries")
		return
	}

	require.Equal(t, []string{"watcher/" + expected}, files,
		"exactly one watcher must be embedded for %s/%s", runtime.GOOS, runtime.GOARCH)
}

// TestWatcherContentsMatchSourceBinary catches a stale or truncated embed by
// comparing against the on-disk binary the directive points at.
func TestWatcherContentsMatchSourceBinary(t *testing.T) {
	expected := expectedWatcherName()
	if expected == "" {
		t.Skipf("no watcher binary for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	embedded, err := fs.ReadFile(Watcher, "watcher/"+expected)
	require.NoError(t, err)
	require.NotEmpty(t, embedded)

	onDisk, err := os.ReadFile("watcher/" + expected)
	require.NoError(t, err)
	assert.Equal(t, onDisk, embedded)
}
