package logger

import (
	"bytes"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger_WithDebug(t *testing.T) {
	// Capture original state
	originalLogger := Logger
	originalStdoutLogger := StdoutLogger
	originalVerboseMode := VerboseMode
	defer func() {
		Logger = originalLogger
		StdoutLogger = originalStdoutLogger
		VerboseMode = originalVerboseMode
	}()

	InitLogger(false, true)

	assert.True(t, VerboseMode, "VerboseMode should be true when debug is enabled")
	require.NotNil(t, Logger, "Logger should be initialized")
	require.NotNil(t, StdoutLogger, "StdoutLogger should be initialized")
}

func TestInitLogger_WithVerbose(t *testing.T) {
	originalLogger := Logger
	originalStdoutLogger := StdoutLogger
	originalVerboseMode := VerboseMode
	defer func() {
		Logger = originalLogger
		StdoutLogger = originalStdoutLogger
		VerboseMode = originalVerboseMode
	}()

	InitLogger(true, false)

	assert.True(t, VerboseMode, "VerboseMode should be true when verbose is enabled")
	require.NotNil(t, Logger, "Logger should be initialized")
	require.NotNil(t, StdoutLogger, "StdoutLogger should be initialized")
}

func TestInitLogger_NoFlags(t *testing.T) {
	originalLogger := Logger
	originalStdoutLogger := StdoutLogger
	originalVerboseMode := VerboseMode
	defer func() {
		Logger = originalLogger
		StdoutLogger = originalStdoutLogger
		VerboseMode = originalVerboseMode
	}()

	InitLogger(false, false)

	assert.False(t, VerboseMode, "VerboseMode should be false")
	require.NotNil(t, Logger, "Logger should be initialized")
	require.NotNil(t, StdoutLogger, "StdoutLogger should be initialized")
}

func TestStderrLogger_RespectsDebugLevel(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Capture stderr output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := NewCustomTextHandler(&buf, opts)
	Logger = slog.New(handler)

	// Debug message should appear
	Logger.Debug("test debug message")
	output := buf.String()
	assert.Contains(t, output, "test debug message")
	assert.Contains(t, output, "DEBUG")
}

func TestStderrLogger_FiltersDebugWhenInfo(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Capture stderr output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := NewCustomTextHandler(&buf, opts)
	Logger = slog.New(handler)

	// Debug message should NOT appear
	Logger.Debug("test debug message")
	output := buf.String()
	assert.NotContains(t, output, "test debug message")

	// Info message should appear
	buf.Reset()
	Logger.Info("test info message")
	output = buf.String()
	assert.Contains(t, output, "test info message")
}

func TestStdoutLogger_AlwaysLevelInfo(t *testing.T) {
	originalStdoutLogger := StdoutLogger
	defer func() {
		StdoutLogger = originalStdoutLogger
	}()

	// Initialize with debug - stdout should still be info
	InitLogger(false, true)

	// Create a test stdout logger with buffer to verify behavior
	var buf bytes.Buffer
	stdoutHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	testLogger := slog.New(stdoutHandler)

	// Debug message should NOT appear
	testLogger.Debug("debug message")
	assert.Empty(t, buf.String())

	// Info message should appear
	testLogger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
}

// Tests for dynamic level changes (these will initially fail until implementation)

func TestToggleDebug_EnablesDebugLevel(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Initialize without debug
	InitLogger(false, false)

	// Verify debug is initially disabled
	assert.False(t, IsDebugEnabled(), "Debug should be initially disabled")

	// Toggle debug on
	ToggleDebug()

	// Verify debug is now enabled
	assert.True(t, IsDebugEnabled(), "Debug should be enabled after toggle")
}

func TestToggleDebug_DisablesDebugLevel(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Initialize with debug
	InitLogger(false, true)

	// Verify debug is initially enabled
	assert.True(t, IsDebugEnabled(), "Debug should be initially enabled")

	// Toggle debug off
	ToggleDebug()

	// Verify debug is now disabled
	assert.False(t, IsDebugEnabled(), "Debug should be disabled after toggle")
}

func TestToggleDebug_ConcurrentAccess(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	InitLogger(false, false)

	// Spawn multiple goroutines that toggle debug concurrently
	var wg sync.WaitGroup
	iterations := 100
	goroutines := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ToggleDebug()
				IsDebugEnabled()
			}
		}()
	}

	// Should not panic or race
	wg.Wait()
}

func TestSetLogLevel_ChangesLevel(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	InitLogger(false, false)

	// Set to debug
	SetLogLevel(slog.LevelDebug)
	assert.True(t, IsDebugEnabled())

	// Set to info
	SetLogLevel(slog.LevelInfo)
	assert.False(t, IsDebugEnabled())

	// Set to warn
	SetLogLevel(slog.LevelWarn)
	assert.False(t, IsDebugEnabled())
}

func TestDynamicLevel_TakesEffectImmediately(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Capture stderr output - need to redirect to buffer
	var buf bytes.Buffer

	// Create logger with LevelVar
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelInfo)

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	handler := NewCustomTextHandler(&buf, opts)
	testLogger := slog.New(handler)

	// Debug message should NOT appear at info level
	testLogger.Debug("debug 1")
	assert.NotContains(t, buf.String(), "debug 1")

	// Change level to debug
	logLevel.Set(slog.LevelDebug)

	// Debug message should now appear
	testLogger.Debug("debug 2")
	assert.Contains(t, buf.String(), "debug 2")

	// Change back to info
	logLevel.Set(slog.LevelInfo)
	buf.Reset()

	// Debug message should NOT appear again
	testLogger.Debug("debug 3")
	assert.NotContains(t, buf.String(), "debug 3")
}

// Helper function to redirect stderr for testing
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}
