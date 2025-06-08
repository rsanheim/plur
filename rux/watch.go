package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rsanheim/rux/watch"
	"github.com/urfave/cli/v2"
)

// Embed the watcher binaries at compile time
//
//go:embed vendor/watcher/*
var watcherBinaries embed.FS

func runWatch(ctx *cli.Context) error {
	// Initialize logging if not already done (watch can be called directly)
	if Logger == nil {
		debug := os.Getenv("RUX_DEBUG") == "1"
		// Try to get verbose flag from context
		verbose := ctx.Bool("verbose")
		InitLogger(verbose, debug)
	}

	// Log startup info
	Logger.Info("rux watch starting!", "version", GetVersionInfo())

	// Create file mapper
	fileMapper := watch.NewFileMapper()

	// Create debouncer with configurable delay
	debounceMs := ctx.Int("debounce")
	debounceDelay := time.Duration(debounceMs) * time.Millisecond
	debouncer := watch.NewDebouncer(debounceDelay)
	LogDebug("Debounce delay", "ms", debounceMs)

	// Determine which directories to watch
	watchDirs := watch.GetWatchDirectories()
	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found (tried: spec, lib, app)")
	}

	// Get project name from current directory
	projectName := "unknown"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
	}
	
	timeout := ctx.Int("timeout")
	
	Logger.Info("rux configuration info", 
		"project", projectName,
		"directories", watchDirs,
		"debounce", debounceMs,
		"timeout", timeout)
	if timeout > 0 {
		LogDebug("rux in timeout mode - with auto exit after "+fmt.Sprintf("%d", timeout)+" seconds")
	}

	// Get the watcher binary path
	watcherPath, err := getWatcherBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}
	
	// Log binary path in debug mode
	LogDebug("rux using e-dant/watcher", "path", watcherPath)

	// Create watcher configuration
	watcherConfig := &watch.Config{
		Directories:    watchDirs,
		DebounceDelay:  debounceDelay,
		TimeoutSeconds: timeout,
	}

	// Create and start the watcher manager
	manager := watch.NewWatcherManager(watcherConfig, watcherPath)
	if err := manager.Start(); err != nil {
		return err
	}
	defer manager.Stop()

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(time.Duration(timeout) * time.Second)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Process events with timeout
	for {
		select {
		case event := <-manager.Events():
			// Convert absolute path to relative for cleaner logs
			cwd, _ := os.Getwd()
			relPath := event.PathName
			if rel, err := filepath.Rel(cwd, event.PathName); err == nil {
				relPath = "./" + rel
			}
			
			LogDebug("watch", 
				"event", event.EffectType,
				"type", event.PathType,
				"associated", fmt.Sprintf("%v", event.Associated),
				"path", relPath)

			// Only process file events (not directories)
			if event.PathType != "file" {
				// Skip logging for directory events
				continue
			}

			// Only process modify and create events
			if event.EffectType != "modify" && event.EffectType != "create" {
				// Skip logging for non-modify/create events
				continue
			}

			// Check if we should watch this file
			if !fileMapper.ShouldWatchFile(event.PathName) {
				// Skip logging for files not in watch list
				continue
			}

			// Convert absolute path to relative path for mapping
			relPath, err = filepath.Rel(cwd, event.PathName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get relative path: %v\n", err)
				continue
			}

			// Map the file to specs
			specsToRun := fileMapper.MapFileToSpecs(relPath)
			if len(specsToRun) == 0 {
				LogDebug("rux", "event", "mapping_not_found", "path", "./"+relPath, "specs", []string{})
				continue
			}
			LogDebug("rux", "event", "mapping_found", "path", "./"+relPath, "specs", specsToRun)

			// File change notification removed - info is in debug logs

			// Debounce the spec runs
			debouncer.Debounce(specsToRun, func(specs []string) {
				// Remove duplicates
				uniqueSpecs := make(map[string]bool)
				for _, spec := range specs {
					uniqueSpecs[spec] = true
				}

				// Run each unique spec
				for spec := range uniqueSpecs {
					LogDebug("rux", "event", "run_spec", "path", "./"+spec)
					runSpecsOrDirectory(spec)
				}
			})

		case err := <-manager.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			Logger.Info("rux timeout reached, exiting!", "event", "timeout", "timeout", timeout)
			fmt.Println("\nTimeout reached, exiting!")
			return nil

		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
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

	LogDebug("Extracted watcher binary", "path", binaryPath)
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
}
