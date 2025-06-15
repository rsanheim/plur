package watch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Event represents a file system event from the watcher
type Event struct {
	PathType   string      `json:"path_type"`
	PathName   string      `json:"path_name"`
	EffectType string      `json:"effect_type"`
	EffectTime int64       `json:"effect_time"`
	Associated interface{} `json:"associated"`
}

// WatcherConfig holds configuration for a single watcher
type WatcherConfig struct {
	Directory      string // Single directory to watch
	DebounceDelay  time.Duration
	TimeoutSeconds int
}

// Watcher manages the file watching process
type Watcher struct {
	config     *WatcherConfig
	binaryPath string
	process    *exec.Cmd
	eventChan  chan Event
	errorChan  chan error
	stopChan   chan struct{}
}

// NewWatcher creates a new watcher instance
func NewWatcher(config *WatcherConfig, binaryPath string) *Watcher {
	return &Watcher{
		config:     config,
		binaryPath: binaryPath,
		eventChan:  make(chan Event, 100),
		errorChan:  make(chan error, 10),
		stopChan:   make(chan struct{}),
	}
}

// Start begins watching the configured directory
func (w *Watcher) Start() error {
	// Convert directory to absolute path
	absPath, err := filepath.Abs(w.config.Directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", w.config.Directory, err)
	}

	// Start the watcher process - logging handled by caller
	w.process = exec.Command(w.binaryPath, absPath)

	// Create stdin pipe to keep watcher alive
	stdinPipe, err := w.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Get stdout for events
	stdout, err := w.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Get stderr for errors
	stderr, err := w.process.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the process
	if err := w.process.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Start goroutines to handle output
	go w.readEvents(stdout)
	go w.readErrors(stderr)

	// Handle process lifecycle
	go func() {
		<-w.stopChan
		stdinPipe.Close()
		w.process.Process.Kill()
		w.process.Wait()
	}()

	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopChan)
}

// Events returns the event channel
func (w *Watcher) Events() <-chan Event {
	return w.eventChan
}

// Errors returns the error channel
func (w *Watcher) Errors() <-chan error {
	return w.errorChan
}

// readEvents reads JSON events from stdout
func (w *Watcher) readEvents(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	defer close(w.eventChan)

	for scanner.Scan() {
		line := scanner.Text()

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip non-JSON lines
			continue
		}

		select {
		case w.eventChan <- event:
		case <-w.stopChan:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case w.errorChan <- fmt.Errorf("error reading watcher output: %w", err):
		case <-w.stopChan:
		}
	}
}

// readErrors reads error messages from stderr
func (w *Watcher) readErrors(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(os.Stderr, "watcher stderr: %s\n", line)
	}
}
