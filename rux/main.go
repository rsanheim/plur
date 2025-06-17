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

type SpecCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
}

func (r *SpecCmd) Run(parent *RuxCLI) error {
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
	Run     WatchRunCmd     `cmd:"" default:"" help:"Run watch mode"`
	Install WatchInstallCmd `cmd:"" help:"Install the watcher binary"`
}

type WatchRunCmd struct {
	// Flags for watch command
	Timeout  int `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int `help:"Debounce delay in milliseconds" default:"100"`
}

func (w *WatchRunCmd) Run(parent *RuxCLI) error {
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

	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w.Timeout, w.Debounce)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *RuxCLI) error {
	return runWatchInstall(true)
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

type DBSetupCmd struct{}

func (d *DBSetupCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	// Use parent.Workers since Kong parses -n as a global flag
	workerCount := GetWorkerCount(parent.Workers)
	return RunDatabaseTask("db:setup", workerCount, config.DryRun)
}

type DBCreateCmd struct{}

func (d *DBCreateCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	// Use parent.Workers since Kong parses -n as a global flag
	workerCount := GetWorkerCount(parent.Workers)
	return RunDatabaseTask("db:create", workerCount, config.DryRun)
}

type DBMigrateCmd struct{}

func (d *DBMigrateCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	// Use parent.Workers since Kong parses -n as a global flag
	workerCount := GetWorkerCount(parent.Workers)
	return RunDatabaseTask("db:migrate", workerCount, config.DryRun)
}

type DBPrepareCmd struct{}

func (d *DBPrepareCmd) Run(parent *RuxCLI) error {
	config := &Config{
		DryRun: parent.DryRun,
	}
	// Use parent.Workers since Kong parses -n as a global flag
	workerCount := GetWorkerCount(parent.Workers)
	return RunDatabaseTask("db:test:prepare", workerCount, config.DryRun)
}

type RuxCLI struct {
	// Commands
	Spec      SpecCmd      `cmd:"" help:"Run tests" default:"withargs"`
	Watch     WatchCmd     `cmd:"" help:"Watch for file changes and run tests automatically"`
	Doctor    DoctorCmd    `cmd:"" help:"Diagnose Rux installation and environment"`
	DBSetup   DBSetupCmd   `cmd:"" name:"db:setup" help:"Setup test databases"`
	DBCreate  DBCreateCmd  `cmd:"" name:"db:create" help:"Create test databases"`
	DBMigrate DBMigrateCmd `cmd:"" name:"db:migrate" help:"Migrate test databases"`
	DBPrepare DBPrepareCmd `cmd:"" name:"db:test:prepare" help:"Prepare test databases"`

	// Global flags
	Auto       bool   `help:"Automatically run bundle install before tests" default:"false"`
	Verbose    bool   `help:"Enable verbose output for debugging" default:"false"`
	Debug      bool   `help:"Enable debug output (includes verbose)" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	Colour     bool   `help:"Force colorized output (British spelling)" negatable:"" hidden:""`
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

	// Sync British spelling to American spelling
	// If --no-colour is used, r.Colour is false and we need to set r.Color to false
	// If --colour is used, r.Colour is true and we need to set r.Color to true
	// The issue is that Kong sets the flag based on what's explicitly provided
	// TODO: This is a limitation of Kong - we can't distinguish between
	// "not set" vs "explicitly set to false"

	// For now, we'll check if the args contain --no-colour
	for _, arg := range os.Args {
		if arg == "--no-colour" {
			r.Color = false
			break
		}
	}

	debug := r.Debug || os.Getenv("RUX_DEBUG") == "1"
	logger.InitLogger(r.Verbose, debug)

	return nil
}

func main() {
	var cli RuxCLI
	configPaths := InitConfigPaths()
	ctx := kong.Parse(&cli,
		kong.Name("rux"),
		kong.Description("A fast Go-based test runner for Ruby/RSpec"),
		kong.Vars{
			"cache_dir": configPaths.CacheDir,
		})

	logger.Logger.Debug("running kong CLI", "args", os.Args, "ctx", ctx)
	err := ctx.Run(ctx)
	if err != nil {
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
