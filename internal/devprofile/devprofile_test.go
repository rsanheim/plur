package devprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartThenStopWritesAllProfiles(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, Start(root))
	Stop()

	dir := filepath.Join(root, fmt.Sprintf("plur-%d", os.Getpid()))
	for _, name := range []string{"cpu.pprof", "heap.pprof", "goroutine.txt", "goroutineleak.txt"} {
		info, err := os.Stat(filepath.Join(dir, name))
		require.NoError(t, err, name)
		assert.NotZero(t, info.Size(), name)
	}

	leaks, err := os.ReadFile(filepath.Join(dir, "goroutineleak.txt"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(leaks), "goroutineleak profile: total 0"), string(leaks))
}

func TestStartFailsWhenDirectoryCannotBeCreated(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(file, nil, 0o644))

	require.ErrorContains(t, Start(filepath.Join(file, "profiles")), "dev-profile")
}
