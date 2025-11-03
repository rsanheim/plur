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

	// Create debounce delay (for future use)
	debounceDelay := time.Duration(watchCmd.Debounce) * time.Millisecond
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
	fmt.Println("  reload   - Reload configuration")
	fmt.Println("  exit     - Exit watch mode")
	fmt.Println("  Ctrl-C   - Exit watch mode")
	fmt.Println()
	fmt.Print("plur> ")

	// Resolve the working directory in case we're in a symlinked directory
	cwd, _ := os.Getwd()
	if resolvedCwd, err := filepath.EvalSymlinks(cwd); err == nil {
		if resolvedCwd != cwd {
			logger.LogDebug("watch", "cwd_symlink_resolved", true,
				"original", cwd, "resolved", resolvedCwd)
		}
		cwd = resolvedCwd
	}

	// Process events with watchCmd.Timeout
	for {
		select {
		case input := <-stdinChan:
			switch input {
			case "":
				// User pressed Enter - run all specs
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				runCommand("spec", currentTask.Run)
				fmt.Print("\nplur> ")
			case "reload":
				// User requested process reload
				logger.LogDebug("User requested process reload")
				fmt.Println("Reloading plur...")

				// Get current executable path
				execPath, err := os.Executable()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to reload: %v\n", err)
					fmt.Print("plur> ")
					continue
				}

				// Must cleanup before exec - defers won't run
				manager.Stop()

				// Atomic process replacement (Unix/Linux/macOS only)
				// Process is replaced in-place, maintaining the same PID
				args := os.Args
				env := os.Environ()
				err = syscall.Exec(execPath, args, env)

				// Only reached if exec fails
				fmt.Fprintf(os.Stderr, "Failed to exec new process: %v\n", err)
				os.Exit(1)
			case "exit":
				// User typed exit command
				logger.Logger.Info("User requested exit")
				fmt.Println("Exiting watch mode...")
				return nil
			default:
				// Unknown command
				fmt.Printf("Unknown command: '%s'\n", input)
				fmt.Println("Commands: [Enter] to run all tests, 'reload' to reload config, 'exit' to quit")
				fmt.Print("plur> ")
			}

		case event := <-manager.Events():
			// Compute relative path for debug logging
			relPath := event.PathName
			if rel, err := filepath.Rel(cwd, event.PathName); err == nil {
				relPath = "./" + rel
			}

			logger.LogDebug("watch", "event", event.EffectType, "type", event.PathType,
				"associated", fmt.Sprintf("%v", event.Associated), "path", relPath)

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

			// Convert absolute path to relative path for display
			relPath, err = filepath.Rel(cwd, event.PathName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get relative path: %v\n", err)
				continue
			}

			// Just report the file change - no mapping or test execution
			logger.Logger.Info("Running: [handler for file]", "path", "./"+relPath)
			fmt.Print("plur> ")

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

// Simple implementation using direct command call for now
func runCommand(targetPath string, command string) {
	var cmd *exec.Cmd

	cmdParts := strings.Fields(command)
	args := append(cmdParts, targetPath)
	cmd_string := strings.Join(args, " ")

	fmt.Println("running:", cmd_string)

	if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("file not found: %s\n", targetPath)
		return
	}

	cmd = exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
	}
}
