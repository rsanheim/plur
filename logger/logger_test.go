package logger

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	// StdoutLogger should always be at info level (even when main logger is debug)
	Init(slog.LevelDebug)

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

	// Initialize with info level
	Init(slog.LevelInfo)

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

	// Initialize with debug level
	Init(slog.LevelDebug)

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

	Init(slog.LevelInfo)

	// Spawn multiple goroutines that toggle debug concurrently
	var wg sync.WaitGroup
	iterations := 100
	goroutines := 10

	for i := 0; i < goroutines; i++ {
		wg.Go(func() {
			for j := 0; j < iterations; j++ {
				ToggleDebug()
				IsDebugEnabled()
			}
		})
	}

	// Should not panic or race
	wg.Wait()
}

func TestSetLogLevel_ChangesLevel(t *testing.T) {
	originalLogger := Logger
	defer func() {
		Logger = originalLogger
	}()

	// Initialize first
	Init(slog.LevelWarn)

	// Change to debug with SetLogLevel
	SetLogLevel(slog.LevelDebug)
	assert.True(t, IsDebugEnabled())

	// Change to info with SetLogLevel
	SetLogLevel(slog.LevelInfo)
	assert.False(t, IsDebugEnabled())

	// Change to warn with SetLogLevel
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

func TestCustomTextHandler_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	handler := NewCustomTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	var wg sync.WaitGroup
	goroutines := 10
	iterations := 100

	for i := 0; i < goroutines; i++ {
		workerID := i
		wg.Go(func() {
			for j := 0; j < iterations; j++ {
				logger.Info("message", "worker", workerID, "iteration", j)
			}
		})
	}

	wg.Wait()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly goroutines * iterations lines
	assert.Equal(t, goroutines*iterations, len(lines), "Expected %d log lines", goroutines*iterations)

	// Each line should be complete (contains expected format)
	for i, line := range lines {
		assert.Contains(t, line, "INFO", "Line %d should contain INFO level", i)
		assert.Contains(t, line, "message", "Line %d should contain message", i)
		assert.Contains(t, line, "worker=", "Line %d should contain worker attribute", i)
		assert.Contains(t, line, "iteration=", "Line %d should contain iteration attribute", i)
	}
}

func TestIsVerboseEnabled(t *testing.T) {
	// Test various log levels
	testCases := []struct {
		level   slog.Level
		verbose bool
	}{
		{slog.LevelDebug, true},  // Debug is verbose
		{slog.LevelInfo, true},   // Info is verbose
		{slog.LevelWarn, false},  // Warn is not verbose
		{slog.LevelError, false}, // Error is not verbose
	}

	for _, tc := range testCases {
		Init(tc.level)
		result := IsVerboseEnabled()
		assert.Equal(t, tc.verbose, result, "Expected IsVerboseEnabled()=%v for level %v", tc.verbose, tc.level)
	}
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

func TestInitFromArgs_DefaultWarn(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	InitFromArgs([]string{"spec"})
	assert.False(t, IsDebugEnabled())
	assert.False(t, IsVerboseEnabled())
}

func TestInitFromArgs_VerboseFlag(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	InitFromArgs([]string{"--verbose"})
	assert.True(t, IsVerboseEnabled())
	assert.False(t, IsDebugEnabled())

	InitFromArgs([]string{"-v"})
	assert.True(t, IsVerboseEnabled())
	assert.False(t, IsDebugEnabled())
}

func TestInitFromArgs_DebugFlag(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	InitFromArgs([]string{"--debug"})
	assert.True(t, IsDebugEnabled())

	InitFromArgs([]string{"-d"})
	assert.True(t, IsDebugEnabled())
}

func TestInitFromArgs_EnvDebug(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "1")
	InitFromArgs([]string{"spec"})
	assert.True(t, IsDebugEnabled())
}

func TestInitFromArgs_DebugWinsOverVerbose(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "")
	InitFromArgs([]string{"--verbose", "--debug"})
	assert.True(t, IsDebugEnabled())
}

func TestInitFromArgs_CLIFlagWinsOverEnv(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "0")
	InitFromArgs([]string{"-d"})
	assert.True(t, IsDebugEnabled())
}

func TestInitFromArgs_EnvDebugEscalatesVerbose(t *testing.T) {
	t.Setenv("PLUR_DEBUG", "1")
	InitFromArgs([]string{"-v"})
	assert.True(t, IsDebugEnabled())
}
