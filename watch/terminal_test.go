package watch

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminal_ShowPromptWritesPromptOnce(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()
	terminal.ShowPrompt()

	assert.Equal(t, "[plur] > ", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestTerminal_PrintLineSuspendsPromptBeforeWriting(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()
	terminal.PrintLine("bundle exec rspec spec/foo_spec.rb")

	assert.Equal(t, "[plur] > \nbundle exec rspec spec/foo_spec.rb\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestTerminal_StderrWriterSuspendsPromptBeforeWrite(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	terminal := NewTerminal(&stdout, &stderr, "[plur] > ")
	terminal.ShowPrompt()

	_, err := io.WriteString(terminal.Stderr(), "04:33:19 - DEBUG - watch type=\"watcher\"\n")
	require.NoError(t, err)

	assert.Equal(t, "[plur] > \n", stdout.String())
	assert.Equal(t, "04:33:19 - DEBUG - watch type=\"watcher\"\n", stderr.String())
}
