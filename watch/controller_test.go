package watch

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWatcher satisfies WatcherSource with plain channels, so controller
// tests never spawn the real watcher binary.
type fakeWatcher struct {
	events   chan Event
	errors   chan error
	startErr error
	started  bool
	stopped  bool
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{events: make(chan Event, 10), errors: make(chan error, 1)}
}

func (f *fakeWatcher) Start() error         { f.started = true; return f.startErr }
func (f *fakeWatcher) Stop()                { f.stopped = true }
func (f *fakeWatcher) Events() <-chan Event { return f.events }
func (f *fakeWatcher) Errors() <-chan error { return f.errors }

// syncBuffer lets the test read output while the controller goroutine writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

// blockingStdin yields one command, then blocks until the Controller closes it.
type blockingStdin struct {
	mu       sync.Mutex
	sent     bool
	closed   chan struct{}
	readDone chan struct{}
	closeMu  sync.Once
}

func newBlockingStdin() *blockingStdin {
	return &blockingStdin{closed: make(chan struct{}), readDone: make(chan struct{})}
}

func (s *blockingStdin) Read(p []byte) (int, error) {
	s.mu.Lock()
	if !s.sent {
		s.sent = true
		s.mu.Unlock()
		return copy(p, "exit\n"), nil
	}
	s.mu.Unlock()

	<-s.closed
	close(s.readDone)
	return 0, io.EOF
}

func (s *blockingStdin) Close() error {
	s.closeMu.Do(func() { close(s.closed) })
	return nil
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

// controllerHarness runs a Controller in a goroutine with fakes on every
// seam and blocks until the first prompt, so tests type against a settled
// session.
type controllerHarness struct {
	stdout  *syncBuffer
	stderr  *syncBuffer
	watcher *fakeWatcher
	stdinW  *io.PipeWriter
	signals chan os.Signal
	done    chan error
}

func startController(t *testing.T, mutate func(*ControllerConfig)) *controllerHarness {
	t.Helper()
	h := &controllerHarness{
		stdout:  &syncBuffer{},
		stderr:  &syncBuffer{},
		watcher: newFakeWatcher(),
		signals: make(chan os.Signal, 1),
		done:    make(chan error, 1),
	}
	stdinR, stdinW := io.Pipe()
	h.stdinW = stdinW

	cfg := ControllerConfig{
		Planner:       Planner{CWD: t.TempDir()},
		RunAllJob:     echoJob(),
		DebounceDelay: time.Millisecond,
		Watcher:       h.watcher,
		Signals:       h.signals,
		Stdin:         stdinR,
		Stdout:        h.stdout,
		Stderr:        h.stderr,
		Reload:        func() error { return nil },
	}
	if mutate != nil {
		mutate(&cfg)
	}

	ctrl := NewController(cfg)
	go func() { h.done <- ctrl.Run() }()
	waitForOutput(t, h.stdout, "[plur] > ")
	return h
}

func (h *controllerHarness) typeLine(t *testing.T, line string) {
	t.Helper()
	_, err := h.stdinW.Write([]byte(line + "\n"))
	require.NoError(t, err)
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for file %s", path)
}

func TestControllerRun_ExitCommand(t *testing.T) {
	h := startController(t, nil)

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))

	out := h.stdout.String()
	assert.Contains(t, out, "Available commands")
	assert.Contains(t, out, "[plur] > Exiting watch mode...")
	assert.True(t, h.watcher.started, "controller should start the watcher")
	assert.True(t, h.watcher.stopped, "controller should stop the watcher on exit")
	assert.Empty(t, h.stderr.String())
}

func TestControllerRun_ClosesAndJoinsStdinOnExit(t *testing.T) {
	stdin := newBlockingStdin()
	watcher := newFakeWatcher()
	ctrl := NewController(ControllerConfig{
		Planner: Planner{CWD: t.TempDir()},
		Watcher: watcher,
		Signals: make(chan os.Signal),
		Stdin:   stdin,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		Reload:  func() error { return nil },
	})

	require.NoError(t, ctrl.Run())
	select {
	case <-stdin.closed:
	default:
		t.Fatal("Controller.Run returned without closing stdin")
	}
	select {
	case <-stdin.readDone:
	default:
		t.Fatal("Controller.Run returned before the stdin reader stopped")
	}
}

func TestControllerRun_AnnouncesStartedOnlyAfterWatcherStarts(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		var announced atomic.Bool
		h := startController(t, func(cfg *ControllerConfig) {
			cfg.OnStarted = func() { announced.Store(true) }
		})

		assert.True(t, announced.Load())
		h.typeLine(t, "exit")
		require.NoError(t, waitForReturn(t, h.done))
	})

	t.Run("failed start", func(t *testing.T) {
		watcher := newFakeWatcher()
		watcher.startErr = errors.New("watcher failed to start")
		stdin := newBlockingStdin()
		var announced atomic.Bool
		ctrl := NewController(ControllerConfig{
			Planner:   Planner{CWD: t.TempDir()},
			Watcher:   watcher,
			Signals:   make(chan os.Signal),
			Stdin:     stdin,
			Stdout:    io.Discard,
			Stderr:    io.Discard,
			Reload:    func() error { return nil },
			OnStarted: func() { announced.Store(true) },
		})

		err := ctrl.Run()

		require.EqualError(t, err, "watcher failed to start")
		assert.False(t, announced.Load())
		assert.False(t, watcher.stopped)
		select {
		case <-stdin.closed:
		default:
			t.Fatal("Controller.Run returned without closing stdin")
		}
	})
}

func TestControllerRun_EnterRunsAllTests(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	h := startController(t, func(cfg *ControllerConfig) {
		cfg.RunAllJob = JobRun{Job: framework.Job{Name: "rspec", Cmd: []string{"touch", marker}}}
	})

	h.typeLine(t, "")
	waitForFile(t, marker)
	// The blank line and fresh prompt after the run are pinned output.
	waitForOutput(t, h.stdout, "[plur] > Running all tests...\n\n[plur] > ")

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))
	assert.Empty(t, h.stderr.String())
}

func TestControllerRun_EnterReportsRunFailureOnStderr(t *testing.T) {
	h := startController(t, func(cfg *ControllerConfig) {
		cfg.RunAllJob = JobRun{Job: framework.Job{Name: "rspec", Cmd: []string{"false"}}}
	})

	h.typeLine(t, "")
	waitForOutput(t, h.stderr, "Failed to run:")
	waitForOutput(t, h.stdout, "Running all tests...\n\n[plur] > ")

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))
}

func TestControllerRun_HelpAndUnknownCommands(t *testing.T) {
	h := startController(t, nil)

	h.typeLine(t, "help")
	waitForOutput(t, h.stdout, "[plur] > Available commands")

	h.typeLine(t, "wat")
	waitForOutput(t, h.stdout, "Unknown command: 'wat'")

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))
	assert.Equal(t, 3, strings.Count(h.stdout.String(), "Available commands"),
		"startup, help, and unknown-command should each print the table")
}

func TestControllerRun_DebugToggle(t *testing.T) {
	h := startController(t, nil)

	h.typeLine(t, "debug")
	h.typeLine(t, "debug") // toggle back so global logger state is restored
	waitForOutput(t, h.stdout, "Debug output enabled")
	waitForOutput(t, h.stdout, "Debug output disabled")

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))
}

func TestControllerRun_StdinEOFThenTimeout(t *testing.T) {
	h := startController(t, func(cfg *ControllerConfig) {
		cfg.Timeout = 200 * time.Millisecond
	})

	// Closing stdin must idle the REPL, not end or spin the session.
	require.NoError(t, h.stdinW.Close())

	require.NoError(t, waitForReturn(t, h.done))
	assert.Contains(t, h.stdout.String(), "Timeout reached, exiting!")
}

func TestControllerRun_SignalsEndSession(t *testing.T) {
	cases := []struct {
		sig     os.Signal
		message string
	}{
		{syscall.SIGINT, "Received SIGINT, shutting down gracefully..."},
		{syscall.SIGTERM, "Received SIGTERM, shutting down gracefully..."},
	}
	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			h := startController(t, nil)

			h.signals <- tc.sig

			require.NoError(t, waitForReturn(t, h.done))
			assert.Contains(t, h.stdout.String(), tc.message)
		})
	}
}

func TestControllerRun_ReloadFailureEndsSession(t *testing.T) {
	var reloads atomic.Int32
	h := startController(t, func(cfg *ControllerConfig) {
		cfg.Reload = func() error {
			reloads.Add(1)
			return errors.New("exec failed")
		}
	})

	h.typeLine(t, "reload")
	err := waitForReturn(t, h.done)

	require.EqualError(t, err, "exec failed")
	assert.Contains(t, h.stdout.String(), "Failed to reload: exec failed")
	assert.Equal(t, int32(1), reloads.Load())
	assert.True(t, h.watcher.stopped)
}

func TestControllerRun_SighupReloadFailureEndsSession(t *testing.T) {
	h := startController(t, func(cfg *ControllerConfig) {
		cfg.Reload = func() error { return errors.New("exec failed") }
	})

	h.signals <- syscall.SIGHUP
	err := waitForReturn(t, h.done)

	require.EqualError(t, err, "exec failed")
	assert.Contains(t, h.stdout.String(), "Received SIGHUP, reloading plur...")
	assert.Contains(t, h.stdout.String(), "Failed to reload: exec failed")
	assert.True(t, h.watcher.stopped)
}

func TestControllerRun_WatcherErrorEndsSession(t *testing.T) {
	h := startController(t, nil)

	h.watcher.errors <- errors.New("watcher died")

	err := waitForReturn(t, h.done)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "watcher error: watcher died")
	assert.True(t, h.watcher.stopped)
}

func TestControllerRun_FileEventTriggersPlannedRun(t *testing.T) {
	projectDir := t.TempDir()
	writeFileTree(t, projectDir, "spec/user_spec.rb")
	marker := filepath.Join(projectDir, "ran")

	h := startController(t, func(cfg *ControllerConfig) {
		cfg.Planner = Planner{
			Jobs: map[string]framework.Job{
				"rspec": {Name: "rspec", Cmd: []string{"touch", marker}},
			},
			Watches: []WatchMapping{libToSpec()},
			CWD:     projectDir,
		}
		cfg.DebounceDelay = 5 * time.Millisecond
	})

	h.watcher.events <- Event{PathType: "file", PathName: "lib/user.rb", EffectType: "modify"}

	waitForFile(t, marker)
	// A batch that ran jobs re-shows the prompt after a blank line.
	waitForOutput(t, h.stdout, "\n[plur] > ")

	h.typeLine(t, "exit")
	require.NoError(t, waitForReturn(t, h.done))
}
