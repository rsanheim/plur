package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bmatcuk/doublestar/v4"
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

// executeJob runs a job with the given target files
func executeJob(j job.Job, targetFiles []string, cwd string) error {
	if len(targetFiles) == 0 {
		return nil
	}

	logger.Logger.Info("Executing job", "job", j.Name, "targets", fmt.Sprintf("%+v", targetFiles))

	// Build command for each target file
	for _, target := range targetFiles {
		cmd := job.BuildJobCmd(j, []string{target})
		logger.Logger.Info("Running command", "cmd", strings.Join(cmd, " "))

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
	debounceDelay := time.Duration(watchCmd.Debounce) * time.Millisecond
	logger.Logger.Debug("Debounce delay", "ms", watchCmd.Debounce)

	// Determine which directories to watch from watch mappings
	var watchDirs []string
	for _, mapping := range watches {
		dir := mapping.SourceDir()
		watchDirs = append(watchDirs, dir)
	}

	logger.Logger.Debug("Watch directories before filtering", "dirs", watchDirs)
	watchDirs, err = filterWatchDirectories(watchDirs)
	if err != nil {
		return fmt.Errorf("failed to filter watch directories: %w", err)
	}
	logger.Logger.Debug("Watch directories after filtering", "dirs", watchDirs)

	if len(watchDirs) == 0 {
		return fmt.Errorf("no directories to watch found in watch mappings")
	}

	// Set up global exclusion patterns (use defaults if not configured)
	globalExcludePatterns := cli.WatchExclude
	if len(globalExcludePatterns) == 0 {
		globalExcludePatterns = defaultWatchExcludePatterns
	}
	logger.Logger.Debug("Global watch exclusion patterns", "patterns", globalExcludePatterns)

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
			logger.Logger.Debug("watch stdin", "input", input)
			switch input {
			case "":
				logger.Logger.Info("Running all tests (manual trigger)")
				fmt.Println("Running all tests...")
				cmd := job.BuildJobAllCmd(resolvedJob)
				runCommandArgs(cmd)
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

			// Skip globally excluded paths (.git, node_modules, etc.)
			if isGloballyExcluded(path, globalExcludePatterns) {
				logger.Logger.Debug("Skipping globally excluded path", "path", path)
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
				logger.Logger.Info("File changed (no watch mapping configured)", "path", path)
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

// filterWatchDirectories validates and filters watch directories:
// 1. Security: Rejects paths that escape the project root (e.g., symlinks to "/")
// 2. Deduplication: Removes symlinks pointing to the same actual directory
// 3. Parent filtering: If dir A contains dir B, keeps only A
//
// Uses os.Root to safely confine operations to the working directory.
func filterWatchDirectories(dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return dirs, nil
	}

	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open root directory: %w", err)
	}
	defer root.Close()

	// Step 1: Validate all directories are within root
	type validDir struct {
		path string
		info os.FileInfo
	}
	valid := []validDir{}

	for _, dir := range dirs {
		info, err := root.Stat(dir)
		if err != nil {
			// Path escapes root or doesn't exist - skip with warning
			logger.Logger.Warn("Skipping watch directory (escapes project root or doesn't exist)",
				"dir", dir, "error", err)
			continue
		}
		if !info.IsDir() {
			logger.Logger.Warn("Skipping watch path (not a directory)", "path", dir)
			continue
		}
		valid = append(valid, validDir{path: dir, info: info})
	}

	if len(valid) == 0 {
		return []string{}, nil
	}

	// Step 2: Remove duplicates (symlinks to same location) using os.SameFile
	deduped := []validDir{}
	for _, v := range valid {
		isDupe := false
		for _, existing := range deduped {
			if os.SameFile(v.info, existing.info) {
				logger.Logger.Debug("Filtering duplicate watch directory",
					"dir", v.path, "same_as", existing.path)
				isDupe = true
				break
			}
		}
		if !isDupe {
			deduped = append(deduped, v)
		}
	}

	// Step 3: Filter subdirectories (if A contains B, keep only A)
	// Sort by path length (shorter paths = likely parents)
	sort.Slice(deduped, func(i, j int) bool {
		return len(deduped[i].path) < len(deduped[j].path)
	})

	result := []string{}
	for _, v := range deduped {
		isSubdir := false
		for _, parent := range result {
			rel, err := filepath.Rel(parent, v.path)
			// v is a subdirectory of parent if:
			// - Rel() succeeds
			// - result doesn't start with ".." (not escaping parent)
			// - result isn't "." (same directory)
			if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
				logger.Logger.Debug("Filtering subdirectory of existing watch",
					"subdir", v.path, "parent", parent)
				isSubdir = true
				break
			}
		}
		if !isSubdir {
			result = append(result, v.path)
		}
	}

	return result, nil
}

// defaultWatchExcludePatterns returns the default patterns to exclude from watch events
var defaultWatchExcludePatterns = []string{".git/**", "node_modules/**"}

// isGloballyExcluded checks if a path matches any of the global exclusion patterns
func isGloballyExcluded(path string, patterns []string) bool {
	normalizedPath := filepath.ToSlash(path)
	for _, pattern := range patterns {
		if matched, _ := doublestar.Match(pattern, normalizedPath); matched {
			return true
		}
	}
	return false
}
