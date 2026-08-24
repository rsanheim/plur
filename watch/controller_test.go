package watch

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWatcher satisfies WatcherSource with plain channels, so controller
// tests never spawn the real watcher binary.
type fakeWatcher struct {
	events  chan Event
	errors  chan error
	started bool
	stopped bool
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{events: make(chan Event, 10), errors: make(chan error, 1)}
}

func (f *fakeWatcher) Start() error         { f.started = true; return nil }
func (f *fakeWatcher) Stop()                { f.stopped = true }
func (f *fakeWatcher) Events() <-chan Event { return f.events }
func (f *fakeWatcher) Errors() <-chan error { return f.errors }

// syncBuffer lets the test read output while the controller goroutine writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitForOutput(t *testing.T, buf *syncBuffer, substr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), substr) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in output:\n%s", substr, buf.String())
}

func waitForReturn(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("controller.Run did not return")
		return nil
	}
}

func echoJob() JobRun {
	return JobRun{Job: framework.Job{Name: "rspec", Cmd: []string{"echo", "ok"}}}
}

func TestControllerRun_ExitCommand(t *testing.T) {
	stdout := &syncBuffer{}
	stderr := &syncBuffer{}
	watcher := newFakeWatcher()
	stdinR, stdinW := io.Pipe()

	ctrl := NewController(ControllerConfig{
		Planner:       Planner{CWD: t.TempDir()},
		RunAllJob:     echoJob(),
		DebounceDelay: time.Millisecond,
		Watcher:       watcher,
		Signals:       make(chan os.Signal),
		Stdin:         stdinR,
		Stdout:        stdout,
		Stderr:        stderr,
		Reload:        func() error { return nil },
	})

	done := make(chan error, 1)
	go func() { done <- ctrl.Run() }()

	waitForOutput(t, stdout, "[plur] > ")
	_, err := stdinW.Write([]byte("exit\n"))
	require.NoError(t, err)

	require.NoError(t, waitForReturn(t, done))

	out := stdout.String()
	assert.Contains(t, out, "Available commands")
	assert.Contains(t, out, "[plur] > Exiting watch mode...")
	assert.True(t, watcher.started, "controller should start the watcher")
	assert.True(t, watcher.stopped, "controller should stop the watcher on exit")
	assert.Empty(t, stderr.String())
}
