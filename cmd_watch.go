package main

import (
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
	"github.com/rsanheim/plur/embedded"
	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

func runWatchInstall(force bool) error {
	configPaths := config.InitConfigPaths()
	return watch.InstallBinary(
		embedded.Watcher,
		configPaths.BinDir,
		configPaths.PlurHome,
		embedded.WatcherVersion(),
		force,
	)
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
		Directories: watchDirs,
	}
	manager := watch.NewWatcherManager(watcherConfig, watcherPath)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	ctrl := watch.NewController(watch.ControllerConfig{
		Planner:       planner,
		RunAllJob:     watch.JobRun{Job: selected.Job},
		DebounceDelay: debounceDelay,
		Timeout:       time.Duration(runCmd.Timeout) * time.Second,
		Watcher:       manager,
		Signals:       sigChan,
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		OnStarted:     func() { printWatchInfo(watchDirs) },
		Reload:        func() error { return reload(manager) },
	})
	return ctrl.Run()
}
