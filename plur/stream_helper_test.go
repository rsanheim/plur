package main

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParser is a minimal parser for testing that doesn't consume lines
type mockParser struct {
	linesReceived []string
}

func (m *mockParser) ParseLine(line string) ([]types.TestNotification, bool) {
	m.linesReceived = append(m.linesReceived, line)
	return nil, false // Don't consume - lines will be added as raw output
}

func (m *mockParser) NotificationToProgress(n types.TestNotification) (string, bool) {
	return "", false
}

func (m *mockParser) FormatSummary(suite *types.SuiteNotification, totalExamples, totalFailures, totalPending int, wallTime, loadTime float64) string {
	return ""
}

func (m *mockParser) FormatFailuresList(failures []types.TestCaseNotification) string {
	return ""
}

func (m *mockParser) ColorizeSummary(summary string, hasFailures bool) string {
	return summary
}

func (m *mockParser) CurrentFile() string {
	return ""
}

func TestStreamTestOutput_LongLineDoesNotHang(t *testing.T) {
	// Generate a line larger than ScannerBufferSize (256KB)
	// Using 300KB to be safely over the limit
	longLineSize := 300 * 1024
	longLine := strings.Repeat("x", longLineSize)

	// Create a subprocess that outputs the long line
	// Using printf to avoid shell argument length limits
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", "printf '%s\\n' \"$LONG_LINE\"")
	cmd.Env = append(cmd.Environ(), "LONG_LINE="+longLine)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	parser := &mockParser{}
	collector := NewTestCollector()

	// Run streamTestOutput - this should NOT hang
	done := make(chan string)
	go func() {
		stderrOutput := streamTestOutput(stdout, stderr, parser, collector, nil, 0, false)
		done <- stderrOutput
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Success - streamTestOutput completed
	case <-ctx.Done():
		t.Fatal("streamTestOutput hung on long line - pipe was not fully drained")
	}

	// Verify the subprocess exited cleanly
	err = cmd.Wait()
	require.NoError(t, err, "subprocess should exit cleanly when pipe is drained")

	// Verify the long line was received by the parser
	require.Len(t, parser.linesReceived, 1, "should receive exactly one line")
	assert.Len(t, parser.linesReceived[0], longLineSize, "long line should be preserved in full")
}

func TestStreamTestOutput_MultipleLines(t *testing.T) {
	// Test normal operation with multiple lines
	stdout := bytes.NewBufferString("line1\nline2\nline3\n")
	stderr := bytes.NewBufferString("err1\nerr2\n")

	parser := &mockParser{}
	collector := NewTestCollector()

	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, nil, 0, false)

	assert.Equal(t, "err1\nerr2\n", stderrOutput)
	assert.Len(t, parser.linesReceived, 3)
	assert.Equal(t, []string{"line1", "line2", "line3"}, parser.linesReceived)
}

func TestStreamTestOutput_LineWithoutTrailingNewline(t *testing.T) {
	// Test that final line without newline is still processed
	stdout := bytes.NewBufferString("line1\nfinal")
	stderr := bytes.NewBufferString("")

	parser := &mockParser{}
	collector := NewTestCollector()

	streamTestOutput(stdout, stderr, parser, collector, nil, 0, false)

	assert.Len(t, parser.linesReceived, 2)
	assert.Equal(t, []string{"line1", "final"}, parser.linesReceived)
}

func TestStreamTestOutput_EmptyInput(t *testing.T) {
	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")

	parser := &mockParser{}
	collector := NewTestCollector()

	stderrOutput := streamTestOutput(stdout, stderr, parser, collector, nil, 0, false)

	assert.Empty(t, stderrOutput)
	assert.Empty(t, parser.linesReceived)
}
