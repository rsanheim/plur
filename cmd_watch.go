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

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/runtime"
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

// buildWatchPlanner resolves the inputs both watch commands share: the
// symlink-resolved cwd, global ignore patterns, and the planner that maps
// changed files to job runs. Job selection is deliberately separate so
// watch find can report missing mappings even when no job is selectable.
func buildWatchPlanner(globals *PlurCLI, watchCmd *WatchCmd) (watch.Planner, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return watch.Planner{}, fmt.Errorf("failed to get current directory: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}

	ignorePatterns := watchCmd.Ignore
	if len(ignorePatterns) == 0 {
		ignorePatterns = watch.DefaultIgnorePatterns
	}
	for _, pattern := range ignorePatterns {
		if !watch.ValidatePattern(pattern) {
			return watch.Planner{}, fmt.Errorf("invalid --ignore pattern %q", pattern)
		}
	}

	return watch.Planner{
		Jobs:           globals.runtimeConfig.Jobs,
		Watches:        globals.runtimeConfig.Watches,
		IgnorePatterns: ignorePatterns,
		CWD:            cwd,
	}, nil
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
	fmt.Printf("plur %s ready and watching %v\n", buildinfo.GetVersionInfo(), strings.Join(absoluteWatchDirs, ", "))
	fmt.Println()
}

func runWatchWithConfig(globalConfig *config.GlobalConfig, runCmd *WatchRunCmd, watchCmd *WatchCmd, cli *PlurCLI) error {
	logger.Logger.Info("plur watch starting!", "version", buildinfo.GetVersionInfo())

	planner, err := buildWatchPlanner(cli, watchCmd)
	if err != nil {
		return err
	}

	selected, err := runtime.SelectJobFromRuntimeConfig(cli.runtimeConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to select watch job: %w", err)
	}
	runtime.LogInheritedFields(selected.Name, selected.Inherited)

	if len(planner.Watches) > 0 {
		logger.Logger.Info("Watch configuration loaded", "job", selected.Job.Name, "watch_mappings", len(planner.Watches))
	} else {
		logger.Logger.Info("No watch mappings configured, file changes will not trigger tests")
	}

	debounceDelay := time.Duration(runCmd.Debounce) * time.Millisecond
	debouncer := watch.NewDebouncer(debounceDelay)
	logger.Logger.Debug("Debounce delay", "ms", runCmd.Debounce)

	var watchDirs []string
	for _, mapping := range planner.Watches {
		watchDirs = append(watchDirs, mapping.SourceDir())
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

	logger.Logger.Debug("Global watch ignore patterns", "patterns", planner.IgnorePatterns)

	projectName := "unknown"
	if cwd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(cwd)
	}

	logger.Logger.Info("plur configuration info",
		"project", projectName,
		"directories", watchDirs,
		"job", selected.Job.Name,
		"reason", selected.Reason,
		"watch", fmt.Sprintf("%+v", planner.Watches),
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

	manager := watch.NewWatcherManager(watcherConfig, watcherPath)
	if err := manager.Start(); err != nil {
		return err
	}
	defer manager.Stop()

	var timeoutChan <-chan time.Time
	if runCmd.Timeout > 0 {
		timeoutChan = time.After(time.Duration(runCmd.Timeout) * time.Second)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	stdinChan := make(chan string, 10)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			stdinChan <- line
		}
	}()

	promptChan := make(chan struct{}, 1)
	showPrompt := func() {
		select {
		case promptChan <- struct{}{}:
		default: // already queued, skip
		}
	}

	reloadChan := make(chan struct{}, 1)
	triggerReload := func() {
		select {
		case reloadChan <- struct{}{}:
		default: // already queued, skip
		}
	}

	printWatchInfo(watchDirs)
	printHelp()
	showPrompt()

	for {
		select {
		case input := <-stdinChan:
			logger.Logger.Debug("received via stdin", "input", input)
			switch input {
			case "":
				fmt.Println("Running all tests...")
				if err := watch.ExecuteJob(watch.JobRun{Job: selected.Job}, planner.CWD); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to run: %v\n", err)
				}
				fmt.Println()
				showPrompt()
			case "help":
				printHelp()
				showPrompt()
			case "reload":
				logger.Logger.Debug("User requested process reload")
				triggerReload()
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
			if event.PathType == "watcher" {
				logger.Logger.Debug("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))
				continue
			}

			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			path, ok := planner.Admit(event.PathName)
			if !ok {
				continue
			}

			logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)

			debouncer.Debounce([]string{path}, func(paths []string) {
				plan := planner.Plan(paths)
				for _, run := range plan.Runs {
					if err := watch.ExecuteJob(run, planner.CWD); err != nil {
						logger.Logger.Warn("Job execution error", "job", run.Job.Name, "error", err)
					}
				}
				if plan.Reload {
					triggerReload()
				}
				if len(plan.Runs) > 0 {
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

		case <-reloadChan:
			if err := reload(manager); err != nil {
				logger.Logger.Error("Failed to reload", "error", err)
				fmt.Println("Failed to reload:", err)
				showPrompt()
			}
		}
	}
}
