package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/config"
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
func loadWatchConfiguration(cli *PlurCLI) (map[string]job.Job, []watch.WatchMapping, error) {
	// Start with user-configured jobs and watches
	jobs := make(map[string]job.Job)
	var watches []watch.WatchMapping

	// Load jobs from config
	for name, j := range cli.Job {
		j.Name = name
		jobs[name] = j
	}

	// Load watch mappings from config
	for i := range cli.WatchMappings {
		watches = append(watches, cli.WatchMappings[i])
	}

	// If no configuration provided, use autodetected defaults
	if len(jobs) == 0 && len(watches) == 0 {
		defaultJobs, defaultWatches := autodetect.GetAutodetectedDefaults()
		jobs = defaultJobs
		watches = defaultWatches

		if len(jobs) > 0 {
			logger.Logger.Info("Using autodetected default configuration", "profile", autodetect.AutodetectProfile())
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
func executeJob(j job.Job, targetFiles []string, cwd string) error {
	if len(targetFiles) == 0 {
		return nil
	}

	logger.Logger.Info("Executing job", "job", j.Name, "targets", fmt.Sprintf("%+v", targetFiles))

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
		logger.Logger.Info("Running command", "cmd", strings.Join(cmd, " "))

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

func runWatchWithConfig(globalConfig *config.GlobalConfig, watchCmd *WatchRunCmd, cli *PlurCLI) error {
	// Log startup info
	logger.Logger.Info("plur watch starting!", "version", GetVersionInfo())

	// Load watch configuration (jobs and watch mappings)
	jobs, watches, err := loadWatchConfiguration(cli)
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	// Validate job name if explicitly specified
	if watchCmd.Use != "" {
		if _, exists := jobs[watchCmd.Use]; !exists {
			// Build helpful error message
			availableJobs := make([]string, 0, len(jobs))
			for name := range jobs {
				availableJobs = append(availableJobs, name)
			}
			sort.Strings(availableJobs)
			return fmt.Errorf("job '%s' not found. Available jobs: %s", watchCmd.Use, strings.Join(availableJobs, ", "))
		}
	}

	// Log watch configuration
	if len(watches) > 0 {
		logger.Logger.Info("Watch configuration loaded",
			"jobs", len(jobs),
			"watch_mappings", len(watches))
	} else {
		logger.Logger.Info("No watch mappings configured, file changes will not trigger tests")
	}

	// Create debounce delay (for future use)
	debounceDelay := time.Duration(watchCmd.Debounce) * time.Millisecond
	logger.Logger.Debug("Debounce delay", "ms", watchCmd.Debounce)

	// Determine which directories to watch from watch mappings
	var watchDirs []string
	for _, mapping := range watches {
		dir := mapping.SourceDir()
		if _, err := os.Stat(dir); err == nil {
			watchDirs = append(watchDirs, dir)
		}
	}
	sort.Strings(watchDirs)
	watchDirs = slices.Compact(watchDirs)
	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found in watch mappings")
	}

	// Get project name from current directory
	projectName := "unknown"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
	}

	logger.Logger.Info("plur configuration info",
		"project", projectName,
		"directories", watchDirs,
		"jobs", len(jobs),
		"watch", fmt.Sprintf("%+v", watches),
		"debug", globalConfig.Debug,
		"verbose", globalConfig.Verbose,
		"debounce", watchCmd.Debounce,
		"timeout", watchCmd.Timeout)
	if watchCmd.Timeout > 0 {
		logger.Logger.Debug("plur in timeout mode - with auto exit after " + fmt.Sprintf("%d", watchCmd.Timeout) + " seconds")
	}

	// Get the watcher binary path
	watcherPath, err := watch.GetWatcherBinaryPath(globalConfig.ConfigPaths.BinDir)
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

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
	fmt.Println("  debug    - Toggle debug output")
	fmt.Println("  exit     - Exit watch mode")
	fmt.Println("  Ctrl-C   - Exit watch mode")
	fmt.Println()
	fmt.Print("plur> ")

	// Resolve the working directory in case we're in a symlinked directory
	cwd, _ := os.Getwd()
	if resolvedCwd, err := filepath.EvalSymlinks(cwd); err == nil {
		if resolvedCwd != cwd {
			logger.Logger.Debug("watch", "cwd_symlink_resolved", true,
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
				// User pressed Enter - run all tests using first job
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				// Find first test job (rspec or minitest) or use any job
				var firstJob job.Job
				var found bool
				if j, exists := jobs["rspec"]; exists {
					firstJob = j
					found = true
				} else if j, exists := jobs["minitest"]; exists {
					firstJob = j
					found = true
				} else {
					// Use first available job
					for _, j := range jobs {
						firstJob = j
						found = true
						break
					}
				}
				if found {
					cmd := job.BuildJobAllCmd(firstJob)
					runCommandArgs(cmd)
				} else {
					fmt.Println("No jobs configured")
				}
				fmt.Print("\nplur> ")
			case "reload":
				// User requested process reload
				logger.Logger.Debug("User requested process reload")
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
				if logger.IsDebugEnabled() {
					args = append(args, "--debug")
				}
				env := os.Environ()
				err = syscall.Exec(execPath, args, env)
				fmt.Fprintf(os.Stderr, "Failed to exec new process: %v\n", err)
				os.Exit(1)
			case "debug":
				// Toggle debug mode
				logger.ToggleDebug()
				if logger.IsDebugEnabled() {
					fmt.Println("Debug output enabled")
					logger.Logger.Debug("Debug mode activated")
				} else {
					fmt.Println("Debug output disabled")
				}
				fmt.Print("plur> ")
			case "exit":
				// User typed exit command
				logger.Logger.Info("User requested exit")
				fmt.Println("Exiting watch mode...")
				return nil
			default:
				// Unknown command
				fmt.Printf("Unknown command: '%s'\n", input)
				fmt.Println("Commands: [Enter] to run all tests, 'reload' to reload config, 'debug' to toggle debug, 'exit' to quit")
				fmt.Print("plur> ")
			}

		case event := <-manager.Events():
			// Compute relative path for debug logging
			relPath := event.PathName
			if rel, err := filepath.Rel(cwd, event.PathName); err == nil {
				relPath = "./" + rel
			}

			logger.Logger.Debug("watch", "event", event.EffectType, "type", event.PathType,
				"associated", fmt.Sprintf("%v", event.Associated), "path", relPath)

			// Only process file events (not directories)
			if event.PathType != "file" {
				continue
			}

			// Only process modify and create events
			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			// Convert absolute path to relative path for display
			relPath, err = filepath.Rel(cwd, event.PathName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get relative path: %v\n", err)
				continue
			}

			// Process the file change if we have watch mappings configured
			if len(watches) > 0 {
				// Find targets for this file, filtering out non-existent files
				result, err := watch.FindTargetsForFile(relPath, jobs, watches)
				if err != nil {
					logger.Logger.Warn("Error processing file change", "path", relPath, "error", err)
					continue
				}

				// Check if we have any valid targets
				if !result.HasExistingTargets() {
					logger.Logger.Debug("No matching watch rules for file", "path", "./"+relPath)
					continue
				}

				// Log warnings for missing target files
				if result.HasMissingTargets() {
					for jobName, targets := range result.MissingTargets {
						for _, target := range targets {
							logger.Logger.Info("Skipping non-existent target", "target", target, "job", jobName)
						}
					}
				}

				// Execute each job with its existing targets only
				for jobName, targets := range result.ExistingTargets {
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

// runCommandArgs runs a command from a slice of arguments
func runCommandArgs(args []string) {
	if len(args) == 0 {
		return
	}

	fmt.Println("running:", strings.Join(args, " "))

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
	}
}
