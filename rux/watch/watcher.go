package watch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// Config holds configuration for the watcher
type Config struct {
	Directories    []string
	DebounceDelay  time.Duration
	TimeoutSeconds int
}

// Watcher manages the file watching process
type Watcher struct {
	config     *Config
	binaryPath string
	process    *exec.Cmd
	eventChan  chan Event
	errorChan  chan error
	stopChan   chan struct{}
}

// NewWatcher creates a new watcher instance
func NewWatcher(config *Config, binaryPath string) *Watcher {
	return &Watcher{
		config:     config,
		binaryPath: binaryPath,
		eventChan:  make(chan Event, 100),
		errorChan:  make(chan error, 10),
		stopChan:   make(chan struct{}),
	}
}

// Start begins watching the configured directories
func (w *Watcher) Start() error {
	// Convert directories to absolute paths
	absPaths := make([]string, len(w.config.Directories))
	for i, dir := range w.config.Directories {
		absPath, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
		}
		absPaths[i] = absPath
	}

	// Start the watcher process - logging handled by caller
	w.process = exec.Command(w.binaryPath, absPaths...)

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

// GetBinaryPath determines the platform-specific watcher binary path
func GetBinaryPath(cacheDir string) (string, error) {
	// Determine platform-specific binary name
	var binaryName string
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "arm64", "aarch64":
			binaryName = "watcher-aarch64-apple-darwin"
		case "amd64":
			return "", fmt.Errorf("Intel Mac (x86_64) is not supported. Please use an Apple Silicon Mac")
		default:
			return "", fmt.Errorf("unsupported macOS architecture: %s", runtime.GOARCH)
		}
	case "linux":
		switch runtime.GOARCH {
		case "arm64", "aarch64":
			binaryName = "watcher-aarch64-unknown-linux-gnu"
		case "amd64":
			binaryName = "watcher-x86_64-unknown-linux-gnu"
		default:
			return "", fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Return the expected binary path
	return filepath.Join(cacheDir, "bin", binaryName), nil
}
