package main

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapLogLevel_DefaultWarn(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	assert.Equal(t, slog.LevelWarn, bootstrapLogLevel([]string{"spec"}))
}

func TestBootstrapLogLevel_VerboseFlag(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	assert.Equal(t, slog.LevelInfo, bootstrapLogLevel([]string{"--verbose"}))
	assert.Equal(t, slog.LevelInfo, bootstrapLogLevel([]string{"-v"}))
}

func TestBootstrapLogLevel_DebugFlag(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	assert.Equal(t, slog.LevelDebug, bootstrapLogLevel([]string{"--debug"}))
	assert.Equal(t, slog.LevelDebug, bootstrapLogLevel([]string{"-d"}))
}

func TestBootstrapLogLevel_EnvDebug(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "1")
	assert.Equal(t, slog.LevelDebug, bootstrapLogLevel([]string{"spec"}))
}

func TestBootstrapLogLevel_DebugWinsOverVerbose(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	assert.Equal(t, slog.LevelDebug, bootstrapLogLevel([]string{"--verbose", "--debug"}))
}
