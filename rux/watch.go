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
	PathType    string      `json:"path_type"`
	PathName    string      `json:"path_name"`
	EffectType  string      `json:"effect_type"`
	EffectTime  int64       `json:"effect_time"`
	Associated  interface{} `json:"associated"`
}

func runWatch(ctx *cli.Context) error {
	fmt.Println("Starting rux watch mode...")
	fmt.Println("Watching ./spec directory for changes...")
	
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
	defer cmd.Process.Kill()
	
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
		   event.EffectType == "modify" && 
		   strings.HasSuffix(event.PathName, "_spec.rb") {
			fmt.Printf("TODO: Run spec file: %s\n", event.PathName)
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
	
	// Look for the binary in our vendor directory
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	
	vendorPath := filepath.Join(filepath.Dir(exePath), "vendor", "watcher", "watcher-darwin-aarch64")
	if _, err := os.Stat(vendorPath); err == nil {
		return vendorPath, nil
	}
	
	// Try relative to source directory (for development)
	devPath := filepath.Join("vendor", "watcher", "watcher-darwin-aarch64") 
	if _, err := os.Stat(devPath); err == nil {
		return devPath, nil
	}
	
	return "", fmt.Errorf("watcher binary not found")
}