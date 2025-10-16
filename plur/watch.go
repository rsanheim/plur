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

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

// Embed the watcher binaries at compile time
//
//go:embed embedded/watcher/*
var watcherBinaries embed.FS

func runWatchInstall(force bool) error {
	configPaths := config.InitConfigPaths()
	return watch.InstallBinary(watcherBinaries, configPaths.BinDir, configPaths.PlurHome, force)
}

func runWatchWithConfig(globalConfig *config.GlobalConfig, watchCmd *WatchRunCmd, currentTask *task.Task) error {
	// Log startup info
	logger.Logger.Info("plur watch starting!", "version", GetVersionInfo())

	// Create debouncer with configurable delay
	debounceDelay := time.Duration(watchCmd.Debounce) * time.Millisecond
	debouncer := watch.NewDebouncer(debounceDelay)
	logger.LogDebug("Debounce delay", "ms", watchCmd.Debounce)

	// Determine which directories to watch from Task's SourceDirs
	watchDirs := []string{}
	for _, dir := range currentTask.SourceDirs {
		if _, err := os.Stat(dir); err == nil {
			watchDirs = append(watchDirs, dir)
		}
	}
	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found (tried: %v)", currentTask.SourceDirs)
	}

	// Get project name from current directory
	projectName := "unknown"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
	}

	logger.Logger.Info("plur configuration info",
		"project", projectName,
		"directories", watchDirs,
		"task", currentTask.Name,
		"mappings", currentTask.Mappings,
		"debounce", watchCmd.Debounce,
		"timeout", watchCmd.Timeout)
	if watchCmd.Timeout > 0 {
		logger.LogDebug("plur in timeout mode - with auto exit after " + fmt.Sprintf("%d", watchCmd.Timeout) + " seconds")
	}

	// Get the watcher binary path
	watcherPath, err := watch.GetWatcherBinaryPath(globalConfig.ConfigPaths.BinDir)
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

	// Log binary path in debug mode
	logger.LogDebug("plur using e-dant/watcher", "path", watcherPath)

	// Create watcher configuration
	watcherConfig := &watch.ManagerConfig{
		Directories:    watchDirs,
		DebounceDelay:  debounceDelay,
		TimeoutSeconds: watchCmd.Timeout,
	}

	// Create and start the watcher manager
	manager := watch.NewWatcherManager(watcherConfig, watcherPath)
	if err := manager.Start(); err != nil {
		return err
	}
	defer manager.Stop()

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if watchCmd.Timeout > 0 {
		timeoutChan = time.After(time.Duration(watchCmd.Timeout) * time.Second)
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
	fmt.Print("plur> ")

	// Process events with watchCmd.Timeout
	for {
		select {
		case input := <-stdinChan:
			switch input {
			case "":
				// User pressed Enter - run all specs
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				runSpecsOrDirectory("spec", currentTask.Run)
				fmt.Print("\nplur> ")
			case "exit":
				// User typed exit command
				logger.Logger.Info("User requested exit")
				fmt.Println("Exiting watch mode...")
				return nil
			default:
				// Unknown command
				fmt.Printf("Unknown command: '%s'\n", input)
				fmt.Println("Commands: [Enter] to run all tests, 'exit' to quit")
				fmt.Print("plur> ")
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

			// Convert absolute path to relative path for mapping first
			relPath, err = filepath.Rel(cwd, event.PathName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get relative path: %v\n", err)
				continue
			}

			// Check if we should watch this file by seeing if it matches any mapping pattern
			if !shouldWatchFile(relPath, currentTask) {
				// Skip logging for files not in watch list
				continue
			}

			// Map the file to specs using Task
			specsToRun := currentTask.MapFilesToTarget([]string{relPath})
			if len(specsToRun) == 0 {
				// Still log the mapping_not_found for tests
				logger.LogDebug("plur", "event", "mapping_not_found", "path", "./"+relPath, "specs", []string{})
				continue
			}
			logger.LogDebug("plur", "event", "mapping_found", "path", "./"+relPath, "specs", specsToRun)

			// Debounce the spec runs
			debouncer.Debounce(specsToRun, func(specs []string) {
				// Remove duplicates
				uniqueSpecs := make(map[string]bool)
				for _, spec := range specs {
					uniqueSpecs[spec] = true
				}

				// Run each unique spec
				for spec := range uniqueSpecs {
					logger.LogDebug("plur", "event", "run_spec", "path", "./"+spec)
					runSpecsOrDirectory(spec, currentTask.Run)
				}

				go func() {
					time.Sleep(50 * time.Millisecond)
					fmt.Print("plur> ")
				}()
			})

		case err := <-manager.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", watchCmd.Timeout)
			fmt.Println("\nTimeout reached, exiting!")
			return nil

		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
			return nil
		}
	}
}

// Simple implementation using direct rspec call for now
// We'll integrate with plur runner properly later
func runSpecsOrDirectory(specPath string, command string) {
	var cmd *exec.Cmd

	if _, err := os.Stat(specPath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Spec file not found: %s\n", specPath)
		return
	}

	// Split the command string into parts
	cmdParts := strings.Fields(command)
	args := append(cmdParts, "--format", "progress", specPath)
	cmd_string := strings.Join(args, " ")

	fmt.Println("running:", cmd_string)

	cmd = exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run spec: %v\n", err)
	}
}

// shouldWatchFile determines if a file should trigger spec runs by checking if it matches any mapping pattern
func shouldWatchFile(filePath string, currentTask *task.Task) bool {
	for _, mapping := range currentTask.Mappings {
		if matched, err := doublestar.Match(mapping.Pattern, filePath); err == nil && matched {
			return true
		}
	}
	return false
}
