package watch

import (
	"embed"
	"os"
	"os/exec"
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

	got, err := os.ReadFile(installedPath)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	if runtime.GOOS == "darwin" {
		assert.NoError(t, exec.Command("codesign", "--verify", installedPath).Run())
	}
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

// TestInstallBinaryOverwritesExistingWithForce covers the upgrade path: a stale
// binary already at the target must be replaced by the embedded one when forced.
func TestInstallBinaryOverwritesExistingWithForce(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()

	existing := filepath.Join(binDir, binaryName)
	require.NoError(t, os.WriteFile(existing, []byte("stale"), 0o755))

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, t.TempDir(), true))

	content, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.NotEqual(t, "stale", string(content), "forced install must overwrite the existing binary")
	assert.NotEmpty(t, content)
}

func TestInstallBinaryErrorsWhenNotEmbedded(t *testing.T) {
	skipUnlessWatcherSupported(t)

	var empty embed.FS
	err := InstallBinary(empty, t.TempDir(), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not embedded")
}
