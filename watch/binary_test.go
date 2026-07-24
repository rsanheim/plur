package watch

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rsanheim/plur/embedded"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if version, ok := os.LookupEnv("PLUR_TEST_WATCHER_VERSION"); ok {
		fmt.Println(version)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

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

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, t.TempDir(), embedded.WatcherVersion(), true))

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

func TestInstallBinarySkipsMatchingVersionWithoutForce(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()
	plurHome := t.TempDir()

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, plurHome, embedded.WatcherVersion(), true))

	existing := filepath.Join(binDir, binaryName)
	fixedTime := time.Unix(1_700_000_000, 0)
	require.NoError(t, os.Chtimes(existing, fixedTime, fixedTime))
	before, err := os.Stat(existing)
	require.NoError(t, err)

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, plurHome, embedded.WatcherVersion(), false))

	info, err := os.Stat(existing)
	require.NoError(t, err)
	assert.Equal(t, before.ModTime(), info.ModTime(), "matching watcher must not be overwritten")
}

func TestInstallBinaryReplacesMismatchedVersionWithoutForce(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()
	plurHome := t.TempDir()

	existing := filepath.Join(binDir, binaryName)
	testBinary, err := os.Executable()
	require.NoError(t, err)
	data, err := os.ReadFile(testBinary)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(existing, data, 0o755))
	t.Setenv("PLUR_TEST_WATCHER_VERSION", "0.13.8")

	fixedTime := time.Unix(1_700_000_000, 0)
	require.NoError(t, os.Chtimes(existing, fixedTime, fixedTime))

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, plurHome, embedded.WatcherVersion(), false))

	info, err := os.Stat(existing)
	require.NoError(t, err)
	assert.NotEqual(t, fixedTime, info.ModTime(), "mismatched watcher must be overwritten")
	version, err := installedWatcherVersion(existing)
	require.NoError(t, err)
	assert.Equal(t, embedded.WatcherVersion(), version)
}

func TestInstallBinaryReplacesUnreadableVersionWithoutForce(t *testing.T) {
	binaryName := skipUnlessWatcherSupported(t)
	binDir := t.TempDir()
	plurHome := t.TempDir()

	existing := filepath.Join(binDir, binaryName)
	require.NoError(t, os.WriteFile(existing, []byte("not a watcher"), 0o755))

	require.NoError(t, InstallBinary(embedded.Watcher, binDir, plurHome, embedded.WatcherVersion(), false))

	version, err := installedWatcherVersion(existing)
	require.NoError(t, err)
	assert.Equal(t, embedded.WatcherVersion(), version)
}

func TestInstallBinaryErrorsWhenNotEmbedded(t *testing.T) {
	skipUnlessWatcherSupported(t)

	var empty embed.FS
	err := InstallBinary(empty, t.TempDir(), t.TempDir(), embedded.WatcherVersion(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not embedded")
}
