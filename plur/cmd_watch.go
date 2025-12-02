package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

// loadWatchConfiguration resolves job and watch mappings
func loadWatchConfiguration(cli *PlurCLI, explicitJobName string) (job.Job, []watch.WatchMapping, error) {
	result, err := autodetect.ResolveJob(explicitJobName, cli.Job, nil)
	if err != nil {
		return job.Job{}, nil, err
	}

	// Use user's watches if provided, else from resolved result
	watches := cli.WatchMappings
	if len(watches) == 0 {
		watches = result.Watches
	}

	return result.Job, watches, nil
}

func runWatchWithConfig(globalConfig *config.GlobalConfig, runCmd *WatchRunCmd, watchCmd *WatchCmd, cli *PlurCLI) error {
	logger.Logger.Info("plur watch starting!", "version", GetVersionInfo())

	resolvedJob, watches, err := loadWatchConfiguration(cli, watchCmd.Use)
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	// Build jobs map for FindTargetsForFile (expects map)
	jobs := map[string]job.Job{resolvedJob.Name: resolvedJob}

	// Log watch configuration
	if len(watches) > 0 {
		logger.Logger.Info("Watch configuration loaded",
			"job", resolvedJob.Name,
			"watch_mappings", len(watches))
	} else {
		logger.Logger.Info("No watch mappings configured, file changes will not trigger tests")
	}

	// Create debounce delay (for future use)
	debounceDelay := time.Duration(runCmd.Debounce) * time.Millisecond
	logger.Logger.Debug("Debounce delay", "ms", runCmd.Debounce)

	// Determine which directories to watch from watch mappings
	var watchDirs []string
	for _, mapping := range watches {
		dir := mapping.SourceDir()
		watchDirs = append(watchDirs, dir)
	}

	logger.Logger.Debug("Watch directories before filtering", "dirs", watchDirs)
	watchDirs, err = watch.FilterDirectories(watchDirs)
	if err != nil {
		return fmt.Errorf("failed to filter watch directories: %w", err)
	}
	logger.Logger.Debug("Watch directories after filtering", "dirs", watchDirs)

	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found in watch mappings")
	}

	// Set up global ignore patterns (use defaults if not configured)
	globalIgnorePatterns := watchCmd.Ignore
	if len(globalIgnorePatterns) == 0 {
		globalIgnorePatterns = watch.DefaultIgnorePatterns
	}
	logger.Logger.Debug("Global watch ignore patterns", "patterns", globalIgnorePatterns)

	// Get project name from current directory
	projectName := "unknown"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
	}

	logger.Logger.Info("plur configuration info",
		"project", projectName,
		"directories", watchDirs,
		"job", resolvedJob.Name,
		"watch", fmt.Sprintf("%+v", watches),
		"debug", globalConfig.Debug,
		"verbose", globalConfig.Verbose,
		"debounce", runCmd.Debounce,
		"timeout", runCmd.Timeout)
	if runCmd.Timeout > 0 {
		logger.Logger.Debug("plur in timeout mode - with auto exit after " + fmt.Sprintf("%d", runCmd.Timeout) + " seconds")
	}

	// Get the watcher binary path
	watcherPath, err := watch.GetWatcherBinaryPath(globalConfig.ConfigPaths.BinDir)
	if err != nil {
		return fmt.Errorf("failed to find watcher binary: %v", err)
	}

	watcherConfig := &watch.ManagerConfig{
		Directories:    watchDirs,
		DebounceDelay:  debounceDelay,
		TimeoutSeconds: runCmd.Timeout,
	}

	// Create and start the watcher manager
	manager := watch.NewWatcherManager(watcherConfig, watcherPath)
	if err := manager.Start(); err != nil {
		return err
	}
	defer manager.Stop()

	// Set up timeout if specified
	var timeoutChan <-chan time.Time
	if runCmd.Timeout > 0 {
		timeoutChan = time.After(time.Duration(runCmd.Timeout) * time.Second)
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

	// Process events with runCmd.Timeout
	for {
		select {
		case input := <-stdinChan:
			logger.Logger.Debug("watch stdin", "input", input)
			switch input {
			case "":
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				cmd := job.BuildJobAllCmd(resolvedJob)
				watch.RunCommand(cmd)
				fmt.Print("\nplur> ")
			case "reload":
				logger.Logger.Debug("User requested process reload")
				fmt.Println("Reloading plur...")

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
				logger.ToggleDebug()
				if logger.IsDebugEnabled() {
					fmt.Println("Debug output enabled")
					logger.Logger.Debug("Debug mode activated")
				} else {
					fmt.Println("Debug output disabled")
				}
				fmt.Print("plur> ")
			case "exit":
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
			if event.PathType == "watcher" { // log watcher lifecycle events and continue
				logger.Logger.Debug("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))
				continue
			}

			path, err := filepath.Rel(cwd, event.PathName)
			if err != nil {
				logger.Logger.Warn("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "error", fmt.Sprintf("failed to get relative path: %v", err))
				continue
			}

			logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))

			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			// Skip globally ignored paths (.git, node_modules, etc.)
			if watch.IsIgnored(path, globalIgnorePatterns) {
				logger.Logger.Debug("Skipping globally ignored path", "path", path)
				continue
			}

			if len(watches) > 0 {
				result, err := watch.FindTargetsForFile(path, jobs, watches)
				if err != nil {
					logger.Logger.Warn("Error processing file change", "path", path, "error", err)
					continue
				}

				if !result.HasExistingTargets() {
					logger.Logger.Debug("No matching watch rules for file", "path", path)
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
					j, exists := jobs[jobName]
					if !exists {
						logger.Logger.Warn("Job not found", "job", jobName)
						continue
					}

					if err := watch.ExecuteJob(j, targets, cwd); err != nil {
						logger.Logger.Warn("Job execution error", "job", jobName, "error", err)
					}
				}

				fmt.Print("\nplur> ")
			} else {
				logger.Logger.Info("File changed (no watch mapping configured)", "path", path)
				fmt.Print("plur> ")
			}

		case err := <-manager.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", runCmd.Timeout)
			fmt.Println("\nTimeout reached, exiting!")
			return nil

		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
			return nil
		}
	}
}
