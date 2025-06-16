package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/tracing"
)

type RunCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
}

func (r *RunCmd) Run(parent *RuxCLI) error {
	// Build config from parent
	paths := InitConfigPaths()
	config := &Config{
		Auto:         parent.Auto,
		ColorOutput:  parent.Color,
		ConfigPaths:  paths,
		DryRun:       parent.DryRun,
		TraceEnabled: parent.Trace,
		WorkerCount:  GetWorkerCount(parent.Workers),
	}
	
	// Initialize tracing if enabled
	if config.TraceEnabled {
		if err := tracing.Init(true); err != nil {
			return fmt.Errorf("failed to initialize tracer: %v", err)
		}
		defer tracing.Close()
	}

	defer tracing.StartRegion(context.Background(), "main.total_execution")()

	// Discover spec files
	var specFiles []string
	var err error
	if len(r.Patterns) > 0 {
		specFiles, err = ExpandGlobPatterns(r.Patterns)
		if err != nil {
			return err
		}
		if len(specFiles) == 0 {
			return fmt.Errorf("no spec files found matching provided patterns")
		}
	} else {
		specFiles, err = FindSpecFiles()
		if err != nil {
			return err
		}
		if len(specFiles) == 0 {
			return fmt.Errorf("no spec files found")
		}
	}

	// Run bundle install if --auto flag is set
	if config.Auto && !config.DryRun {
		depManager := NewDependencyManager()
		if err := depManager.InstallDependencies(); err != nil {
			return err
		}
	}

	// Create and run executor
	executor := NewTestExecutor(config, specFiles)
	if err := executor.Execute(); err != nil {
		// Exit with error code 1 for test failures
		if strings.Contains(err.Error(), "test run failed") {
			os.Exit(1)
		}
		return err
	}

	return nil
}

type WatchCmd struct {
	Timeout  int  `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int  `help:"Debounce delay in milliseconds" default:"100"`
	Install  bool `help:"Install watcher binary and exit"`
}

func (w *WatchCmd) Run(parent *RuxCLI) error {
	// Build config from parent
	paths := InitConfigPaths()
	config := &Config{
		Auto:         parent.Auto,
		ColorOutput:  parent.Color,
		ConfigPaths:  paths,
		DryRun:       parent.DryRun,
		TraceEnabled: parent.Trace,
		WorkerCount:  GetWorkerCount(parent.Workers),
	}
	
	if w.Install {
		return runWatchInstall(true)
	}
	
	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}
	
	return runWatchWithConfig(config, w.Timeout, w.Debounce)
}

type DoctorCmd struct{}

func (d *DoctorCmd) Run(parent *RuxCLI) error {
	// Build config from parent
	paths := InitConfigPaths()
	config := &Config{
		Auto:         parent.Auto,
		ColorOutput:  parent.Color,
		ConfigPaths:  paths,
		DryRun:       parent.DryRun,
		TraceEnabled: parent.Trace,
		WorkerCount:  GetWorkerCount(parent.Workers),
	}
	return runDoctorWithConfig(config)
}

type DBCmd struct {
	Setup   DBSetupCmd   `cmd:"" help:"Setup test databases"`
	Create  DBCreateCmd  `cmd:"" help:"Create test databases"`
	Migrate DBMigrateCmd `cmd:"" help:"Migrate test databases"`
	Prepare DBPrepareCmd `cmd:"" help:"Prepare test databases"`
}

type DBSetupCmd struct {
	Workers int `short:"n" help:"Number of parallel workers"`
}

func (d *DBSetupCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	workerCount := GetWorkerCount(d.Workers)
	return RunDatabaseTask("db:setup", workerCount, config.DryRun)
}

type DBCreateCmd struct {
	Workers int `short:"n" help:"Number of parallel workers"`
}

func (d *DBCreateCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	workerCount := GetWorkerCount(d.Workers)
	return RunDatabaseTask("db:create", workerCount, config.DryRun)
}

type DBMigrateCmd struct {
	Workers int `short:"n" help:"Number of parallel workers"`
}

func (d *DBMigrateCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	workerCount := GetWorkerCount(d.Workers)
	return RunDatabaseTask("db:migrate", workerCount, config.DryRun)
}

type DBPrepareCmd struct {
	Workers int `short:"n" help:"Number of parallel workers"`
}

func (d *DBPrepareCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	workerCount := GetWorkerCount(d.Workers)
	return RunDatabaseTask("db:test:prepare", workerCount, config.DryRun)
}

type RuxCLI struct {
	// Commands
	Run    RunCmd    `cmd:"" help:"Run tests" default:"withargs"`
	Watch  WatchCmd  `cmd:"" help:"Watch for file changes and run tests automatically"`
	Doctor DoctorCmd `cmd:"" help:"Diagnose Rux installation and environment"`
	DB     DBCmd     `cmd:"" help:"Database management commands"`

	// Global flags
	Auto       bool   `help:"Automatically run bundle install before tests" default:"false"`
	Verbose    bool   `help:"Enable verbose output for debugging" default:"false"`
	Debug      bool   `help:"Enable debug output (includes verbose)" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	RuntimeDir string `help:"Custom directory for runtime data" default:""`
	CacheDir   string `help:"Directory for caching runtime data" default:"${cache_dir}"`
	Trace      bool   `help:"Enable performance tracing (saves to ./rux_trace_*.json)" default:"false"`
	Workers    int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" default:"0"`
	Version    bool   `help:"Show version information"`
}

func (r *RuxCLI) AfterApply() error {
	// Handle version flag
	if r.Version {
		fmt.Println(GetVersionInfo())
		os.Exit(0)
	}
	
	// Initialize logging
	debug := r.Debug || os.Getenv("RUX_DEBUG") == "1"
	logger.InitLogger(r.Verbose, debug)

	return nil
}

func runKongCLI() {
	var cli RuxCLI
	ctx := kong.Parse(&cli,
		kong.Name("rux"),
		kong.Description("A fast Go-based test runner for Ruby/RSpec"),
		kong.Vars{
			"cache_dir": configPaths.CacheDir,
		})

	err := ctx.Run()
	if err != nil {
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
