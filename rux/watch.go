package main

import (
	"bufio"
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

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/watch"
	"github.com/urfave/cli/v2"
)

// Embed the watcher binaries at compile time
//
//go:embed vendor/watcher/*
var watcherBinaries embed.FS

func runWatchInstall(force bool) error {
	// For Kong CLI, we need to initialize paths if globals are not set
	if configPaths == nil {
		configPaths = InitConfigPaths()
	}
	return watch.InstallBinary(watcherBinaries, configPaths.BinDir, configPaths.RuxHome, force)
}

func runWatch(ctx *cli.Context) error {
	timeout := ctx.Int("timeout")
	debounceMs := ctx.Int("debounce")
	return runWatchWithConfig(ruxConfig, timeout, debounceMs)
}

func runWatchWithConfig(config *Config, timeout int, debounceMs int) error {
	// Log startup info
	logger.Logger.Info("rux watch starting!", "version", GetVersionInfo())

	// Create file mapper
	fileMapper := watch.NewFileMapper()

	// Create debouncer with configurable delay
	debounceDelay := time.Duration(debounceMs) * time.Millisecond
	debouncer := watch.NewDebouncer(debounceDelay)
	logger.LogDebug("Debounce delay", "ms", debounceMs)

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

	logger.Logger.Info("rux configuration info",
		"project", projectName,
		"directories", watchDirs,
		"debounce", debounceMs,
		"timeout", timeout)
	if timeout > 0 {
		logger.LogDebug("rux in timeout mode - with auto exit after " + fmt.Sprintf("%d", timeout) + " seconds")
	}

	// Get the watcher binary path
	watcherPath, err := watch.GetWatcherBinaryPath(config.ConfigPaths.BinDir)
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

	// Log binary path in debug mode
	logger.LogDebug("rux using e-dant/watcher", "path", watcherPath)

	// Create watcher configuration
	watcherConfig := &watch.ManagerConfig{
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

	// Set up stdin monitoring for commands
	stdinChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			select {
			case stdinChan <- line:
			default:
				// Channel is full, skip this input
			}
		}
	}()

	// Show instructions to user
	fmt.Println("Watching for file changes.")
	fmt.Println("Commands:")
	fmt.Println("  [Enter]  - Run all tests")
	fmt.Println("  exit     - Exit watch mode")
	fmt.Println("  Ctrl-C   - Exit watch mode")
	fmt.Println()
	fmt.Print("rux> ")

	// Process events with timeout
	for {
		select {
		case input := <-stdinChan:
			switch input {
			case "":
				// User pressed Enter - run all specs
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				runSpecsOrDirectory("spec")
				fmt.Print("\nrux> ")
			case "exit":
				// User typed exit command
				logger.Logger.Info("User requested exit")
				fmt.Println("Exiting watch mode...")
				return nil
			default:
				// Unknown command
				fmt.Printf("Unknown command: '%s'\n", input)
				fmt.Println("Commands: [Enter] to run all tests, 'exit' to quit")
				fmt.Print("rux> ")
			}

		case event := <-manager.Events():
			// Convert absolute path to relative for cleaner logs
			cwd, _ := os.Getwd()
			relPath := event.PathName
			if rel, err := filepath.Rel(cwd, event.PathName); err == nil {
				relPath = "./" + rel
			}

			logger.LogDebug("watch",
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
				logger.LogDebug("rux", "event", "mapping_not_found", "path", "./"+relPath, "specs", []string{})
				continue
			}
			logger.LogDebug("rux", "event", "mapping_found", "path", "./"+relPath, "specs", specsToRun)

			// Debounce the spec runs
			debouncer.Debounce(specsToRun, func(specs []string) {
				// Remove duplicates
				uniqueSpecs := make(map[string]bool)
				for _, spec := range specs {
					uniqueSpecs[spec] = true
				}

				// Run each unique spec
				for spec := range uniqueSpecs {
					logger.LogDebug("rux", "event", "run_spec", "path", "./"+spec)
					runSpecsOrDirectory(spec)
				}

				go func() {
					time.Sleep(50 * time.Millisecond)
					fmt.Print("rux> ")
				}()
			})

		case err := <-manager.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("rux timeout reached, exiting!", "event", "timeout", "timeout", timeout)
			fmt.Println("\nTimeout reached, exiting!")
			return nil

		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
			return nil
		}
	}
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
