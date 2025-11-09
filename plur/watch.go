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
	"github.com/rsanheim/plur/job"
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

// loadWatchConfiguration loads job and watch mappings from config or defaults
func loadWatchConfiguration(cli *PlurCLI, currentTask *task.Task) (map[string]*job.Job, []*watch.WatchMapping, error) {
	// Start with user-configured jobs and watches
	jobs := make(map[string]*job.Job)
	var watches []*watch.WatchMapping

	// Load jobs from config
	for name, j := range cli.Job {
		jobCopy := j
		jobCopy.Name = name
		jobs[name] = &jobCopy
	}

	// Load watch mappings from config
	for i := range cli.WatchMappings {
		watches = append(watches, &cli.WatchMappings[i])
	}

	// If no configuration provided, use autodetected defaults
	if len(jobs) == 0 && len(watches) == 0 {
		defaultJobs, defaultWatches := watch.GetAutodetectedDefaults()
		jobs = defaultJobs
		watches = defaultWatches

		if len(jobs) > 0 {
			logger.LogVerbose("Using autodetected default configuration",
				"profile", watch.AutodetectProfile())
		}
	}

	// Validate configuration
	if len(watches) > 0 {
		if err := watch.ValidateConfig(jobs, watches); err != nil {
			return nil, nil, fmt.Errorf("invalid watch configuration: %w", err)
		}
	}

	return jobs, watches, nil
}

// executeJob runs a job with the given target files
func executeJob(j *job.Job, targetFiles []string, cwd string) error {
	if len(targetFiles) == 0 {
		return nil
	}

	logger.Logger.Info("Executing job", "job", j.Name, "targets", len(targetFiles))

	// Build command for each target file
	for _, target := range targetFiles {
		// Make target relative to cwd if it's absolute
		relTarget := target
		if filepath.IsAbs(target) {
			if rel, err := filepath.Rel(cwd, target); err == nil {
				relTarget = rel
			}
		}

		cmd := job.BuildJobCmd(j, []string{relTarget})
		logger.LogVerbose("Running command", "cmd", strings.Join(cmd, " "))

		// Execute command
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		execCmd.Dir = cwd
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Env = append(os.Environ(), j.Env...)

		if err := execCmd.Run(); err != nil {
			// Log error but don't fail - continue watching
			logger.Logger.Warn("Job execution failed", "job", j.Name, "error", err)
		}
	}

	return nil
}

func runWatchWithConfig(globalConfig *config.GlobalConfig, watchCmd *WatchRunCmd, currentTask *task.Task, cli *PlurCLI) error {
	// Log startup info
	logger.Logger.Info("plur watch starting!", "version", GetVersionInfo())

	// Load watch configuration (jobs and watch mappings)
	jobs, watches, err := loadWatchConfiguration(cli, currentTask)
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	// Create event processor if we have watch mappings
	var processor *watch.EventProcessor
	if len(watches) > 0 {
		processor = watch.NewEventProcessor(jobs, watches)
		logger.LogVerbose("Watch configuration loaded",
			"jobs", len(jobs),
			"watch_mappings", len(watches))
	} else {
		logger.LogVerbose("No watch mappings configured, file changes will not trigger tests")
	}

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
		"debug", globalConfig.Debug,
		"verbose", globalConfig.Verbose,
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

			// Process the file change through EventProcessor if configured
			if processor != nil {
				logger.Logger.Info("File changed", "path", "./"+relPath)

				// Map file to jobs and target files
				jobTargets, err := processor.ProcessPath(relPath)
				if err != nil {
					logger.Logger.Warn("Error processing file change", "path", relPath, "error", err)
					fmt.Print("plur> ")
					continue
				}

				if len(jobTargets) == 0 {
					logger.LogVerbose("No matching watch rules for file", "path", "./"+relPath)
					fmt.Print("plur> ")
					continue
				}

				// Execute each job with its targets
				for jobName, targets := range jobTargets {
					job, exists := jobs[jobName]
					if !exists {
						logger.Logger.Warn("Job not found", "job", jobName)
						continue
					}

					if err := executeJob(job, targets, cwd); err != nil {
						logger.Logger.Warn("Job execution error", "job", jobName, "error", err)
					}
				}

				fmt.Print("\nplur> ")
			} else {
				// No processor - just report the file change
				logger.Logger.Info("File changed (no watch mapping configured)", "path", "./"+relPath)
				fmt.Print("plur> ")
			}

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
