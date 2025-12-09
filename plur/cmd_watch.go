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

func printHelp() {
	cmdWidth := 20
	fmt.Println("Available commands")
	fmt.Printf("  %-*s %s\n", cmdWidth, "[Enter]", "Run all tests")
	fmt.Printf("  %-*s %s\n", cmdWidth, "debug", "Toggle debug mode")
	fmt.Printf("  %-*s %s\n", cmdWidth, "help", "Show this help")
	fmt.Printf("  %-*s %s\n", cmdWidth, "reload", "Reload plur")
	fmt.Printf("  %-*s %s\n", cmdWidth, "exit (Ctrl-C)", "Exit watch mode")
	fmt.Println()
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

// resetTerminal restores terminal to a known good state.
// This handles cases where jobs (like goreleaser with progress bars) may have
// left the terminal in raw mode with echo disabled.
func resetTerminal() {
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run() // Best effort, ignore errors
}

// reload performs an atomic process replacement (Unix/Linux/macOS only)
// and also maintains same args & env, including the debug state from previous process
func reload(manager *watch.WatcherManager) error {
	fmt.Println("Reloading plur...")
	fmt.Println()

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Must cleanup before exec - defers won't run
	manager.Stop()
	resetTerminal()

	args := os.Args
	hasDebugFlag := slices.Contains(args, "--debug") || slices.Contains(args, "-d")
	if logger.IsDebugEnabled() && !hasDebugFlag {
		args = append(args, "--debug")
	}
	if !logger.IsDebugEnabled() && hasDebugFlag {
		args = slices.DeleteFunc(args, func(arg string) bool {
			return arg == "--debug" || arg == "-d"
		})
	}

	env := os.Environ()
	err = syscall.Exec(execPath, args, env)
	if err != nil {
		return fmt.Errorf("failed to exec new process: %w", err)
	}
	os.Exit(1)
	return nil
}

func printWatchInfo(watchDirs []string) {
	absoluteWatchDirs := make([]string, len(watchDirs))
	for i, dir := range watchDirs {
		absoluteWatchDirs[i], _ = filepath.Abs(dir)
	}
	fmt.Printf("plur %s ready and watching %v\n", GetVersionInfo(), strings.Join(absoluteWatchDirs, ", "))
	fmt.Println()
}

func runWatchWithConfig(globalConfig *config.GlobalConfig, runCmd *WatchRunCmd, watchCmd *WatchCmd, cli *PlurCLI) error {
	logger.Logger.Info("plur watch starting!", "version", GetVersionInfo())

	resolvedJob, watches, err := loadWatchConfiguration(cli, watchCmd.Use)
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	// Build jobs map for FindTargetsForFile - include all user-defined jobs
	// plus the resolved job (for when auto-detection is used)
	jobs := make(map[string]job.Job)
	for name, j := range cli.Job {
		jobs[name] = j
	}
	jobs[resolvedJob.Name] = resolvedJob

	if len(watches) > 0 {
		logger.Logger.Info("Watch configuration loaded", "job", resolvedJob.Name, "watch_mappings", len(watches))
	} else {
		logger.Logger.Info("No watch mappings configured, file changes will not trigger tests")
	}

	debounceDelay := time.Duration(runCmd.Debounce) * time.Millisecond
	logger.Logger.Debug("Debounce delay", "ms", runCmd.Debounce)

	// Create debouncer and file event handler
	debouncer := watch.NewDebouncer(debounceDelay)

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

	globalIgnorePatterns := watchCmd.Ignore
	if len(globalIgnorePatterns) == 0 {
		globalIgnorePatterns = watch.DefaultIgnorePatterns
	}
	logger.Logger.Debug("Global watch ignore patterns", "patterns", globalIgnorePatterns)

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

	var timeoutChan <-chan time.Time
	if runCmd.Timeout > 0 {
		timeoutChan = time.After(time.Duration(runCmd.Timeout) * time.Second)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

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

	// Set up prompt channel - buffered to coalesce multiple prompt requests
	promptChan := make(chan struct{}, 1)
	showPrompt := func() {
		select {
		case promptChan <- struct{}{}:
		default: // already queued, skip
		}
	}

	printWatchInfo(watchDirs)
	printHelp()
	showPrompt()

	cwd, _ := os.Getwd()
	if resolvedCwd, err := filepath.EvalSymlinks(cwd); err == nil {
		if resolvedCwd != cwd {
			logger.Logger.Debug("watch", "cwd_symlink_resolved", true, "original", cwd, "resolved", resolvedCwd)
		}
		cwd = resolvedCwd
	}

	// Create file event handler
	handler := &watch.FileEventHandler{
		Jobs:    jobs,
		Watches: watches,
		CWD:     cwd,
	}

	for {
		select {
		case input := <-stdinChan:
			logger.Logger.Debug("received via stdin", "input", input)
			switch input {
			case "":
				fmt.Println("Running all tests...")
				cmd := job.BuildJobAllCmd(resolvedJob)
				watch.RunCommand(cmd)
				fmt.Println()
				showPrompt()
			case "help":
				printHelp()
				showPrompt()
			case "reload":
				logger.Logger.Debug("User requested process reload")
				if err := reload(manager); err != nil {
					logger.Logger.Error("Failed to reload", "error", err)
					fmt.Println("Failed to reload:", err)
					showPrompt()
				}
			case "debug":
				logger.ToggleDebug()
				if logger.IsDebugEnabled() {
					fmt.Println("Debug output enabled")
				} else {
					fmt.Println("Debug output disabled")
				}
				showPrompt()
			case "exit":
				fmt.Println("Exiting watch mode...")
				return nil
			default:
				fmt.Printf("Unknown command: '%s'\n", input)
				printHelp()
				showPrompt()
			}

		case event := <-manager.Events():
			// Early filtering - skip events we don't care about
			if event.PathType == "watcher" {
				logger.Logger.Debug("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))
				continue
			}

			path, err := filepath.Rel(cwd, event.PathName)
			if err != nil {
				logger.Logger.Warn("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "error", fmt.Sprintf("failed to get relative path: %v", err))
				continue
			}

			if watch.IsIgnored(path, globalIgnorePatterns) {
				continue
			}

			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)

			// Debounce and process
			debouncer.Debounce([]string{path}, func(paths []string) {
				result := handler.HandleBatch(paths)
				if result.ShouldReload {
					if err := reload(manager); err != nil {
						logger.Logger.Error("Failed to reload", "error", err)
						fmt.Println("Failed to reload:", err)
					}
				}
				if len(result.ExecutedJobs) > 0 {
					fmt.Println()
					showPrompt()
				}
			})

		case err := <-manager.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", runCmd.Timeout)
			fmt.Println("Timeout reached, exiting!")
			return nil

		case sig := <-sigChan:
			switch sig {
			case syscall.SIGINT:
				fmt.Println("Received SIGINT, shutting down gracefully...")
				return nil
			case syscall.SIGTERM:
				fmt.Println("Received SIGTERM, shutting down gracefully...")
				return nil
			case syscall.SIGHUP:
				fmt.Println("Received SIGHUP, reloading plur...")
				if err := reload(manager); err != nil {
					logger.Logger.Error("Failed to reload", "error", err)
					fmt.Println("Failed to reload:", err)
					showPrompt()
					continue
				}
				// reload() calls syscall.Exec which replaces process, so we never reach here on success
				return nil
			default:
				fmt.Printf("Received signal %v, shutting down gracefully...\n", sig)
				return nil
			}

		case <-promptChan:
			fmt.Print("[plur] > ")
		}
	}
}
