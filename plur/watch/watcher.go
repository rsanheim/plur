package watch

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
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
	done       chan struct{} // Signals cleanup is complete
	stopOnce   sync.Once
	started    bool
}

// NewWatcher creates a new watcher instance
func NewWatcher(config *WatcherConfig, binaryPath string) *Watcher {
	return &Watcher{
		config:     config,
		binaryPath: binaryPath,
		eventChan:  make(chan Event, 100),
		errorChan:  make(chan error, 10),
		stopChan:   make(chan struct{}),
		done:       make(chan struct{}),
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
		close(w.done) // Signal cleanup complete
	}()

	w.started = true
	return nil
}

// Stop stops the watcher and waits for cleanup to complete.
// Safe to call multiple times and safe to call without prior Start().
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopChan)
		if w.started {
			<-w.done // Wait for subprocess to be killed and reaped
		}
	})
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
	defer close(w.errorChan)

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

// DefaultIgnorePatterns are the default patterns to ignore from watch events
var DefaultIgnorePatterns = []string{".git/**", "node_modules/**"}

// RunCommand runs a command from a slice of arguments
func RunCommand(args []string) {
	if len(args) == 0 {
		return
	}

	fmt.Printf("\n[plur] %s\n", strings.Join(args, " "))

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
	}
}

// ExecuteJob runs a job with the given target files
// If the job doesn't use targets (no {{target}} in cmd), it runs once without targets
func ExecuteJob(j job.Job, targetFiles []string, cwd string) error {
	logger.Logger.Info("Executing job", "job", j.Name, "targets", fmt.Sprintf("%+v", targetFiles))

	// Jobs without {{target}} placeholder run once without targets
	if !j.UsesTargets() {
		cmd := j.Cmd
		fmt.Printf("\n[plur] %s\n", strings.Join(cmd, " "))

		execCmd := exec.Command(cmd[0], cmd[1:]...)
		execCmd.Dir = cwd
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Env = append(os.Environ(), j.Env...)

		if err := execCmd.Run(); err != nil {
			logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
			return err
		}
		return nil
	}

	// Jobs with {{target}} run once with all target files batched together
	if len(targetFiles) == 0 {
		return nil
	}

	cmd := job.BuildJobCmd(j, targetFiles)
	fmt.Printf("\n[plur] %s\n", strings.Join(cmd, " "))

	execCmd := exec.Command(cmd[0], cmd[1:]...)
	execCmd.Dir = cwd
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Env = append(os.Environ(), j.Env...)

	if err := execCmd.Run(); err != nil {
		logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
		return err
	}

	return nil
}

// IsIgnored checks if a path matches any of the ignore patterns
func IsIgnored(path string, patterns []string) bool {
	normalizedPath := filepath.ToSlash(path)
	for _, pattern := range patterns {
		if matched, _ := doublestar.Match(pattern, normalizedPath); matched {
			return true
		}
	}
	return false
}

// FilterDirectories validates and filters watch directories:
// 1. Security: Rejects paths that escape the project root (e.g., symlinks to "/")
// 2. Deduplication: Removes symlinks pointing to the same actual directory
// 3. Parent filtering: If dir A contains dir B, keeps only A
//
// Uses os.Root to safely confine operations to the working directory.
func FilterDirectories(dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return dirs, nil
	}

	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open root directory: %w", err)
	}
	defer root.Close()

	// Step 1: Validate all directories are within root
	type validDir struct {
		path string
		info os.FileInfo
	}
	valid := []validDir{}

	for _, dir := range dirs {
		info, err := root.Stat(dir)
		if err != nil {
			// Path escapes root or doesn't exist - skip with warning
			logger.Logger.Warn("Skipping watch directory (escapes project root or doesn't exist)",
				"dir", dir, "error", err)
			continue
		}
		if !info.IsDir() {
			logger.Logger.Warn("Skipping watch path (not a directory)", "path", dir)
			continue
		}
		valid = append(valid, validDir{path: dir, info: info})
	}

	if len(valid) == 0 {
		return []string{}, nil
	}

	// Step 2: Remove duplicates (symlinks to same location) using os.SameFile
	deduped := []validDir{}
	for _, v := range valid {
		isDupe := false
		for _, existing := range deduped {
			if os.SameFile(v.info, existing.info) {
				logger.Logger.Debug("Filtering duplicate watch directory",
					"dir", v.path, "same_as", existing.path)
				isDupe = true
				break
			}
		}
		if !isDupe {
			deduped = append(deduped, v)
		}
	}

	// Step 3: Filter subdirectories (if A contains B, keep only A)
	// Sort by path length (shorter paths = likely parents)
	sort.Slice(deduped, func(i, j int) bool {
		return len(deduped[i].path) < len(deduped[j].path)
	})

	result := []string{}
	for _, v := range deduped {
		isSubdir := false
		for _, parent := range result {
			rel, err := filepath.Rel(parent, v.path)
			// v is a subdirectory of parent if:
			// - Rel() succeeds
			// - result doesn't start with ".." (not escaping parent)
			// - result isn't "." (same directory)
			if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
				isSubdir = true
				break
			}
		}
		if !isSubdir {
			result = append(result, v.path)
		}
	}

	return result, nil
}
