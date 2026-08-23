package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitConfigPathsCreatesConfiguredDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "plur-home")
	t.Setenv("PLUR_HOME", home)

	paths, err := InitConfigPaths()
	require.NoError(t, err)
	require.NotNil(t, paths)
	assert.Equal(t, home, paths.PlurHome)
	assert.DirExists(t, paths.BinDir)
	assert.DirExists(t, paths.CacheDir)
	assert.DirExists(t, paths.RuntimeDir)
	assert.DirExists(t, paths.FormatterDir)
	assert.DirExists(t, paths.RubyLibDir)
}

func TestInitConfigPathsReturnsAnErrorForAFilePLURHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(home, []byte("file"), 0644))
	t.Setenv("PLUR_HOME", home)

	paths, err := InitConfigPaths()
	require.Error(t, err)
	assert.Nil(t, paths)
	assert.Contains(t, err.Error(), "create PLUR_HOME directory")
}
