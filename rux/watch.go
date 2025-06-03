package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
)

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
	fmt.Println("Watching ./spec directory for changes...")
	fmt.Println("Press Ctrl+C to stop")

	// Get the watcher binary path
	watcherPath, err := getWatcherBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

	// Start the watcher process
	cmd := exec.Command(watcherPath, "./spec")

	// Get stdout pipe for reading JSON events
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
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

	// Read JSON events from stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

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
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading watcher output: %v", err)
	}

	return cmd.Wait()
}

func getWatcherBinaryPath() (string, error) {
	// For now, hardcode the darwin-aarch64 binary
	// TODO: Add support for other platforms
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return "", fmt.Errorf("watcher binary not available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Try different possible locations
	possiblePaths := []string{
		// For development - using the downloaded watcher
		"/Users/rsanheim/Downloads/aarch64-apple-darwin/watcher",
	}

	// Also try relative to executable
	if exePath, err := os.Executable(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(filepath.Dir(exePath), "vendor", "watcher", "watcher-darwin-aarch64"))
		// Try in same directory as rux binary
		possiblePaths = append(possiblePaths,
			filepath.Join(filepath.Dir(exePath), "watcher-darwin-aarch64"))
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("watcher binary not found in any of: %v", possiblePaths)
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
