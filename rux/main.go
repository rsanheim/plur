package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/tracing"
)

type SpecCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	Command  string   `help:"Test command to run" default:"bundle exec rspec"`
	Type     string   `short:"t" help:"Test framework type (rspec|minitest)" default:""`
}

// GetFramework returns the TestFramework enum based on the Type field
func (s *SpecCmd) GetFramework() TestFramework {
	return ParseFrameworkType(s.Type)
}

func (r *SpecCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig

	framework := r.GetFramework()
	logger.Logger.Debug("SpecCmd.Run", "command", r.Command, "patterns", r.Patterns, "framework", framework)

	// Initialize tracing if enabled
	if config.TraceEnabled {
		if err := tracing.Init(true); err != nil {
			return fmt.Errorf("failed to initialize tracer: %v", err)
		}
		defer tracing.Close()
	}

	defer tracing.StartRegion(context.Background(), "main.total_execution")()

	// Discover test files
	var testFiles []string
	var err error
	if len(r.Patterns) > 0 {
		testFiles, err = ExpandGlobPatterns(r.Patterns, framework)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			return fmt.Errorf("no test files found matching provided patterns")
		}
	} else {
		testFiles, err = FindTestFiles(framework)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			suffix := getTestFileSuffix(framework)
			dir := "spec"
			if framework == FrameworkMinitest {
				dir = "test"
			}
			return fmt.Errorf("no test files found (looking for *%s in %s/)", suffix, dir)
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
	executor := NewTestExecutor(config, r, testFiles)
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
	Timeout  int    `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int    `help:"Debounce delay in milliseconds" default:"100"`
	Command  string `help:"Test command to run" default:"bundle exec rspec"`
	Type     string `short:"t" help:"Test framework type (rspec|minitest)" default:""`
}

// GetFramework returns the TestFramework enum based on the Type field
func (w *WatchRunCmd) GetFramework() TestFramework {
	return ParseFrameworkType(w.Type)
}

func (w *WatchRunCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig

	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *RuxCLI) error {
	return runWatchInstall(true)
}

type DoctorCmd struct{}

func (d *DoctorCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	return runDoctorWithConfig(parent.globalConfig)
}

type DBSetupCmd struct{}

func (d *DBSetupCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:setup", config.WorkerCount, config.DryRun)
}

type DBCreateCmd struct{}

func (d *DBCreateCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:create", config.WorkerCount, config.DryRun)
}

type DBMigrateCmd struct{}

func (d *DBMigrateCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:migrate", config.WorkerCount, config.DryRun)
}

type DBPrepareCmd struct{}

func (d *DBPrepareCmd) Run(parent *RuxCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:test:prepare", config.WorkerCount, config.DryRun)
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
	Debug      bool   `short:"d" help:"Enable debug output (includes verbose)" env:"RUX_DEBUG" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	Colour     bool   `help:"Force colorized output (British spelling)" negatable:"" hidden:""`
	RuntimeDir string `help:"Custom directory for runtime data" default:""`
	CacheDir   string `help:"Directory for caching runtime data" default:"${cache_dir}"`
	Trace      bool   `help:"Enable performance tracing (saves to ./rux_trace_*.json)" default:"false"`
	ChangeDir  string `short:"C" help:"Change to directory before running (like git -C)" default:""`
	Workers    int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" env:"PARALLEL_TEST_PROCESSORS" default:"0"`
	Version    bool   `help:"Show version information"`

	// Store the built global config
	globalConfig *GlobalConfig `kong:"-"`
}

func (r *RuxCLI) AfterApply() error {
	// Initialize logger early so we can use it
	// Kong has already resolved r.Debug from CLI flag, env var, or config file
	logger.InitLogger(r.Verbose, r.Debug)

	// Change directory if -C flag is provided - do this first
	if r.ChangeDir != "" {
		if err := os.Chdir(r.ChangeDir); err != nil {
			return fmt.Errorf("failed to change directory to %s: %v", r.ChangeDir, err)
		}
		logger.Logger.Debug("Changed directory", "dir", r.ChangeDir)
	}

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

	// Build global config once
	r.globalConfig = &GlobalConfig{
		Auto:         r.Auto,
		ColorOutput:  r.Color,
		ConfigPaths:  InitConfigPaths(),
		Debug:        r.Debug,
		Verbose:      r.Verbose,
		DryRun:       r.DryRun,
		TraceEnabled: r.Trace,
		WorkerCount:  GetWorkerCount(r.Workers),
		RuntimeDir:   r.RuntimeDir,
		JSON:         r.JSON,
	}

	return nil
}

func main() {
	var cli RuxCLI
	configPaths := InitConfigPaths()
	ctx := kong.Parse(&cli,
		kong.Name("rux"),
		kong.Description("A fast Go-based test runner for Ruby/RSpec"),
		kong.Configuration(kongtoml.Loader, ".rux.toml", "~/.rux.toml"),
		kong.Vars{
			"cache_dir": configPaths.CacheDir,
		})

	logger.Logger.Debug("running rux", "args", os.Args[1:], "command", ctx.Command())
	err := ctx.Run(ctx)
	if err != nil {
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
