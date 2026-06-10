package watch

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rsanheim/plur/embedded"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipUnlessWatcherSupported(t *testing.T) string {
	t.Helper()
	name, err := getPlatformBinaryName()
	if err != nil {
		t.Skipf("watch unsupported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
	}
	return name
}

// TestInstallBinaryFromEmbeddedFS exercises the real embedded.Watcher FS end
// to end: getEmbeddedBinaryPath must resolve to a key that actually exists in
// the embedded package's filesystem.
func TestInstallBinaryFromEmbeddedFS(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, t.TempDir(), true))

	installedPath := filepath.Join(binDir, binaryName)
	info, err := os.Stat(installedPath)
	require.NoError(t, err)
	require.NotZero(t, info.Size())
	if runtime.GOOS != "windows" {
		assert.NotZero(t, info.Mode()&0111, "installed watcher must be executable")
	}

	embeddedPath, err := getEmbeddedBinaryPath()
	require.NoError(t, err)
	want, err := fs.ReadFile(embedded.Watcher, embeddedPath)
	require.NoError(t, err)
	got, err := os.ReadFile(installedPath)
	require.NoError(t, err)
	assert.Equal(t, want, got, "installed watcher must match embedded bytes")
}

// TestInstallBinarySkipsExistingWithoutForce covers the idempotent path taken
// on every `plur watch` startup.
func TestInstallBinarySkipsExistingWithoutForce(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()

	existing := filepath.Join(binDir, binaryName)
	require.NoError(t, os.WriteFile(existing, []byte("sentinel"), 0o755))

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, t.TempDir(), false))

	content, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Equal(t, "sentinel", string(content), "existing binary must not be overwritten without force")
}

func TestInstallBinaryErrorsWhenNotEmbedded(t *testing.T) {
	skipUnlessWatcherSupported(t)

	var empty embed.FS
	err := InstallBinary(empty, t.TempDir(), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not embedded")
}
