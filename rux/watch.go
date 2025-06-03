package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// Embed the watcher binaries at compile time
//
//go:embed vendor/watcher/*
var watcherBinaries embed.FS

// WatcherEvent represents a file system event from the watcher
type WatcherEvent struct {
	PathType   string      `json:"path_type"`
	PathName   string      `json:"path_name"`
	EffectType string      `json:"effect_type"`
	EffectTime int64       `json:"effect_time"`
	Associated interface{} `json:"associated"`
}

func runWatch(ctx *cli.Context) error {
	fmt.Println("Starting rux watch mode...")
	fmt.Println("Watching spec directory for changes...")

	timeout := ctx.Int("timeout")
	if timeout > 0 {
		fmt.Printf("Will exit after %d seconds\n", timeout)
	} else {
		fmt.Println("Press Ctrl+C to stop")
	}

	// Get the watcher binary path
	watcherPath, err := getWatcherBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

	// Get absolute path to avoid watcher issues
	watchPath := "spec"
	if _, err := os.Stat(watchPath); os.IsNotExist(err) {
		return fmt.Errorf("spec directory not found in current directory")
	}

	// Convert to absolute path
	absWatchPath, err := filepath.Abs(watchPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	fmt.Printf("Absolute watch path: %s\n", absWatchPath)

	// Start the watcher process with absolute path
	cmd := exec.Command(watcherPath, absWatchPath)

	// Create a pipe for stdin to keep the watcher alive
	// The watcher blocks on stdin.get() and exits when stdin is closed
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}
	// We'll close this pipe when we want the watcher to exit
	defer stdinPipe.Close()

	// Get stdout pipe for reading JSON events
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	// Get stderr pipe for error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %v", err)
	}

	// Ensure we kill the process on exit
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(time.Duration(timeout) * time.Second)
	}

	// Read JSON events from stdout
	scanner := bufio.NewScanner(stdout)
	eventChan := make(chan string)
	errorChan := make(chan error)

	// Start goroutine to read events
	go func() {
		for scanner.Scan() {
			eventChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errorChan <- err
		}
		close(eventChan)
	}()

	// Start goroutine to read stderr
	go func() {
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			fmt.Fprintf(os.Stderr, "watcher stderr: %s\n", stderrScanner.Text())
		}
	}()

	// Process events with timeout
	for {
		select {
		case line, ok := <-eventChan:
			if !ok {
				// Channel closed, exit normally
				return nil
			}

			// Parse JSON event
			var event WatcherEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse event: %v\n", err)
				continue
			}

			// For now, just print the event
			fmt.Printf("Event: %s %s %s\n", event.EffectType, event.PathType, event.PathName)

			// Check if this is a spec file modification
			if event.PathType == "file" &&
				(event.EffectType == "modify" || event.EffectType == "create") &&
				strings.HasSuffix(event.PathName, "_spec.rb") {
				fmt.Printf("\nRunning spec: %s\n", event.PathName)
				runSingleSpec(event.PathName)
			}

		case err := <-errorChan:
			return fmt.Errorf("error reading watcher output: %v", err)

		case <-timeoutChan:
			fmt.Println("\nTimeout reached, exiting watch mode")
			return nil
		}
	}
}

func getWatcherBinaryPath() (string, error) {
	// Determine platform-specific binary name
	var binaryName string
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "arm64", "aarch64":
			binaryName = "watcher-aarch64-apple-darwin"
		case "amd64":
			binaryName = "watcher-x86_64-apple-darwin"
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

	// Get cache directory for extracted binaries
	cacheDir, err := getRuxCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %v", err)
	}

	// Create bin directory in cache
	binDir := filepath.Join(cacheDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %v", err)
	}

	// Target path for the extracted binary
	targetPath := filepath.Join(binDir, binaryName)

	// Check if binary already exists in cache
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil
	}

	// Extract binary from embedded files
	embeddedPath := filepath.Join("vendor/watcher", binaryName)
	data, err := watcherBinaries.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("watcher binary not embedded for %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
	}

	// Write binary to cache
	if err := os.WriteFile(targetPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to write watcher binary: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Extracted watcher binary to: %s\n", targetPath)
	return targetPath, nil
}

func runSingleSpec(specPath string) {
	// Run the spec using the existing rux infrastructure
	// For now, we'll just shell out to rspec directly to keep it simple
	cmd := exec.Command("bundle", "exec", "rspec", "--format", "progress", specPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run spec: %v\n", err)
	}

	fmt.Printf(strings.Repeat("=", 80) + "\n\n")
	fmt.Println("Watching for changes...")
}
