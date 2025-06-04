package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rsanheim/rux/watch"
	"github.com/urfave/cli/v2"
)

// Embed the watcher binaries at compile time
//
//go:embed vendor/watcher/*
var watcherBinaries embed.FS

func runWatch(ctx *cli.Context) error {
	currentLogLevel := slog.SetLogLoggerLevel(slog.LevelDebug)
	defer slog.SetLogLoggerLevel(currentLogLevel) // revert chang

	fmt.Println("Starting rux watch mode...")

	// Create file mapper
	fileMapper := watch.NewFileMapper()

	// Create debouncer with configurable delay
	debounceMs := ctx.Int("debounce")
	debounceDelay := time.Duration(debounceMs) * time.Millisecond
	debouncer := watch.NewDebouncer(debounceDelay)
	slog.Debug("Debounce delay", "ms", debounceMs)

	// Determine which directories to watch
	watchDirs := watch.GetWatchDirectories()
	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found (tried: spec, lib, app)")
	}

	fmt.Printf("Watching directories: %s\n", strings.Join(watchDirs, ", "))

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

	// Create watcher configuration
	watcherConfig := &watch.Config{
		Directories:    watchDirs,
		DebounceDelay:  debounceDelay,
		TimeoutSeconds: timeout,
	}

	// Create and start the watcher
	watcher := watch.NewWatcher(watcherConfig, watcherPath)
	if err := watcher.Start(); err != nil {
		return err
	}
	defer watcher.Stop()

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(time.Duration(timeout) * time.Second)
	}

	// Process events with timeout
	for {
		select {
		case event := <-watcher.Events():
			slog.Debug("Event", "event", event)

			// Only process file events (not directories)
			if event.PathType != "file" {
				continue
			}

			// Only process modify and create events
			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			// Check if we should watch this file
			if !fileMapper.ShouldWatchFile(event.PathName) {
				continue
			}

			// Convert absolute path to relative path for mapping
			cwd, _ := os.Getwd()
			relPath, err := filepath.Rel(cwd, event.PathName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get relative path: %v\n", err)
				continue
			}

			// Map the file to specs
			specsToRun := fileMapper.MapFileToSpecs(relPath)
			if len(specsToRun) == 0 {
				fmt.Printf("Changed: %s (no spec mapping found)\n", relPath)
				continue
			}

			slog.Debug("Changed", "path", relPath)

			// Debounce the spec runs
			debouncer.Debounce(specsToRun, func(specs []string) {
				// Remove duplicates
				uniqueSpecs := make(map[string]bool)
				for _, spec := range specs {
					uniqueSpecs[spec] = true
				}

				// Run each unique spec
				for spec := range uniqueSpecs {
					slog.Debug("Running", "spec", spec)
					runSpecsOrDirectory(spec)
				}
			})

		case err := <-watcher.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			fmt.Println("\nTimeout reached, exiting watch mode")
			return nil
		}
	}
}

func getWatcherBinaryPath() (string, error) {
	// Get cache directory
	cacheDir, err := getRuxCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %v", err)
	}

	// Get expected binary path
	binaryPath, err := watch.GetBinaryPath(cacheDir)
	if err != nil {
		return "", err
	}

	// Check if binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Need to extract the binary
	binDir := filepath.Dir(binaryPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %v", err)
	}

	// Get the binary name from the path
	binaryName := filepath.Base(binaryPath)

	// Extract binary from embedded files
	embeddedPath := filepath.Join("vendor/watcher", binaryName)
	data, err := watcherBinaries.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("watcher binary not embedded: %v", err)
	}

	// Write binary to cache
	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to write watcher binary: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Extracted watcher binary to: %s\n", binaryPath)
	return binaryPath, nil
}

// Simple implementation using direct rspec call for now
// We'll integrate with rux runner properly later
func runSpecsOrDirectory(specPath string) {
	var cmd *exec.Cmd

	if _, err := os.Stat(specPath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Spec file not found: %s\n", specPath)
		return
	}

	args := []string{"bundle", "exec", "rspec", "--format", "progress", specPath}
	cmd_string := strings.Join(args, " ")

	fmt.Println("running:", cmd_string)

	cmd = exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run spec: %v\n", err)
	}
	slog.Debug("watching...")
}
