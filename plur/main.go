package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

type SpecCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	Use      string   `short:"u" help:"Job to use (overrides autodetection)" default:""`
	Auto     bool     `help:"Automatically run bundle install before tests" default:"false"`
}

func (r *SpecCmd) Run(parent *PlurCLI) error {
	cfg := parent.globalConfig

	// Determine which job to use
	// Priority: CLI --use, config use, auto-detect
	jobName := r.Use
	wasExplicit := r.Use != ""
	if jobName == "" && parent.Use != "" {
		jobName = parent.Use
		wasExplicit = true
	}

	// Get job from config or autodetection
	var currentJob *job.Job
	var autodetectedJobs map[string]*job.Job

	if jobName == "" {
		// Autodetect framework
		autodetectedJobs, _ = autodetect.GetAutodetectedDefaults()

		// Smart framework selection based on directory structure
		// If only spec/ exists, use RSpec
		// If only test/ exists, use Minitest
		// If both exist, prefer RSpec (more common in modern Ruby projects)
		hasSpecDir := dirExists("spec")
		hasTestDir := dirExists("test")

		if hasSpecDir && !hasTestDir {
			// Only spec/ directory - use RSpec
			if j, exists := autodetectedJobs["rspec"]; exists && j.TargetPattern != "" {
				jobName = "rspec"
				currentJob = j
			}
		} else if hasTestDir && !hasSpecDir {
			// Only test/ directory - use Minitest
			if j, exists := autodetectedJobs["minitest"]; exists && j.TargetPattern != "" {
				jobName = "minitest"
				currentJob = j
			}
		} else {
			// Both exist or neither exist - use priority order: rspec > minitest > other
			if j, exists := autodetectedJobs["rspec"]; exists && j.TargetPattern != "" {
				jobName = "rspec"
				currentJob = j
			} else if j, exists := autodetectedJobs["minitest"]; exists && j.TargetPattern != "" {
				jobName = "minitest"
				currentJob = j
			} else {
				// Fall back to any job with a target_pattern
				for name, j := range autodetectedJobs {
					if j.TargetPattern != "" {
						jobName = name
						currentJob = j
						break
					}
				}
			}
		}

		if currentJob == nil {
			return fmt.Errorf("no test framework detected. Please create a .plur.toml with a job configuration")
		}
		wasExplicit = false
	} else {
		// Use explicit job name
		if j, exists := parent.Job[jobName]; exists {
			jobCopy := j
			jobCopy.Name = jobName
			currentJob = &jobCopy
		} else {
			// Try autodetected jobs
			autodetectedJobs, _ = autodetect.GetAutodetectedDefaults()
			if j, exists := autodetectedJobs[jobName]; exists {
				currentJob = j
			} else {
				// Build helpful error message
				availableJobs := make([]string, 0, len(parent.Job)+len(autodetectedJobs))
				for name := range parent.Job {
					availableJobs = append(availableJobs, name)
				}
				for name := range autodetectedJobs {
					availableJobs = append(availableJobs, name)
				}
				sort.Strings(availableJobs)
				return fmt.Errorf("job '%s' not found. Available jobs: %s", jobName, strings.Join(availableJobs, ", "))
			}
		}
	}

	// Validate job has target_pattern for file discovery
	if currentJob.TargetPattern == "" {
		return fmt.Errorf("job '%s' cannot be used with plur spec because it has no target_pattern configured", jobName)
	}

	logger.Logger.Debug("SpecCmd.Run", "job", currentJob.Name, "patterns", r.Patterns, "target_pattern", currentJob.TargetPattern)

	// Show hint if both frameworks exist and we auto-detected
	if !wasExplicit && autodetectedJobs != nil && len(autodetectedJobs) > 1 {
		otherFramework := "minitest"
		if currentJob.Name == "minitest" {
			otherFramework = "rspec"
		}
		logger.LogVerbose(fmt.Sprintf("Both spec/ and test/ directories detected. Using %s. Specify --use=%s to run %s instead.",
			currentJob.Name, otherFramework, otherFramework))
	}

	// Discover test files
	var testFiles []string
	var err error
	if len(r.Patterns) > 0 {
		testFiles, err = ExpandPatternsFromJob(r.Patterns, currentJob)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			return fmt.Errorf("no test files found matching provided patterns")
		}
	} else {
		testFiles, err = FindFilesFromJob(currentJob)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			suffix := currentJob.GetTargetSuffix()
			// Determine directory from job's target pattern
			pattern := currentJob.TargetPattern
			dir := "spec"
			if strings.HasPrefix(pattern, "test/") {
				dir = "test"
			}
			return fmt.Errorf("no test files found (looking for *%s in %s/)", suffix, dir)
		}
	}
	msg := fmt.Sprintf("found %v test files", len(testFiles))
	logger.LogVerbose(msg, "testFiles", testFiles)

	// Run bundle install if --auto flag is set
	if r.Auto && !cfg.DryRun {
		depManager := NewDependencyManager()
		if err := depManager.InstallDependencies(); err != nil {
			return err
		}
	}

	// Create and run executor with Auto flag
	cfg.Auto = r.Auto
	executor := NewTestExecutor(cfg, testFiles, currentJob)
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
	Run     WatchRunCmd     `cmd:"" default:"withargs" help:"Run watch mode"`
	Install WatchInstallCmd `cmd:"" help:"Install the watcher binary"`
	Find    WatchFindCmd    `cmd:"" help:"Show what would be executed for a given file change"`
}

type WatchRunCmd struct {
	Timeout  int    `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int    `help:"Debounce delay in milliseconds" default:"100"`
	Use      string `short:"u" help:"Job to use (overrides autodetection)" default:""`
}

func (w *WatchRunCmd) Run(parent *PlurCLI) error {
	config := parent.globalConfig

	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w, parent)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *PlurCLI) error {
	return runWatchInstall(true)
}

type DoctorCmd struct{}

func (d *DoctorCmd) Run(parent *PlurCLI) error {
	return runDoctorWithConfig(parent.globalConfig)
}

type DBSetupCmd struct{}

func (d *DBSetupCmd) Run(parent *PlurCLI) error {
	return RunDatabaseTask("db:setup", parent.globalConfig)
}

type DBCreateCmd struct{}

func (d *DBCreateCmd) Run(parent *PlurCLI) error {
	return RunDatabaseTask("db:create", parent.globalConfig)
}

type DBMigrateCmd struct{}

func (d *DBMigrateCmd) Run(parent *PlurCLI) error {
	return RunDatabaseTask("db:migrate", parent.globalConfig)
}

type DBPrepareCmd struct{}

func (d *DBPrepareCmd) Run(parent *PlurCLI) error {
	return RunDatabaseTask("db:test:prepare", parent.globalConfig)
}

type PlurCLI struct {
	// Commands
	Spec       SpecCmd       `cmd:"" help:"Run tests" default:"withargs"`
	Watch      WatchCmd      `cmd:"" help:"Watch for file changes and run tests automatically"`
	Doctor     DoctorCmd     `cmd:"" help:"Diagnose Plur installation and environment"`
	ConfigInit ConfigInitCmd `cmd:"" name:"config:init" help:"Generate a starter configuration file"`
	DBSetup    DBSetupCmd    `cmd:"" name:"db:setup" help:"Setup test databases"`
	DBCreate   DBCreateCmd   `cmd:"" name:"db:create" help:"Create test databases"`
	DBMigrate  DBMigrateCmd  `cmd:"" name:"db:migrate" help:"Migrate test databases"`
	DBPrepare  DBPrepareCmd  `cmd:"" name:"db:test:prepare" help:"Prepare test databases"`

	// Global flags (alphabetically sorted for help display)
	// ChangeDir is kept for Kong's help text and CLI compatibility, but the actual
	// directory change is handled early in main() before config loading
	ChangeDir  string `short:"C" help:"Change to directory before running (like git -C)" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	Colour     bool   `help:"Force colorized output (British spelling)" negatable:"" default:"true" hidden:""`
	Debug      bool   `short:"d" help:"Enable debug output (includes verbose)" env:"PLUR_DEBUG" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	FirstIs1   bool   `help:"Start TEST_ENV_NUMBER at 1 instead of empty string (default: true)" negatable:"" default:"true"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	RuntimeDir string `help:"Custom directory for runtime data" default:""`
	Use        string `help:"Job to use (overrides autodetection)" default:"" hidden:""`
	Verbose    bool   `short:"v" help:"Enable verbose output for debugging" default:"false"`
	Version    bool   `help:"Show version information"`
	Workers    int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" env:"PARALLEL_TEST_PROCESSORS" default:"0"`

	// Job and watch configuration
	Job           map[string]job.Job   `help:"Job configurations (config file only)" hidden:""`
	WatchMappings []watch.WatchMapping `help:"Watch mappings (config file only)" hidden:"" toml:"watch"`

	// Store the built global config
	globalConfig *config.GlobalConfig `kong:"-"`

	// Store config files that were attempted (for tracking)
	configFiles []string `kong:"-"`
}

func (r *PlurCLI) AfterApply() error {
	// Initialize logger early so we can use it
	// Kong has already resolved r.Debug from CLI flag, env var, or config file
	logger.InitLogger(r.Verbose, r.Debug)

	if r.Version {
		fmt.Println(GetVersionInfo())
		os.Exit(0)
	}

	if !r.Colour || !r.Color { // silly british spelling
		r.Color = false
	}

	configPaths := config.InitConfigPaths()

	var loadedConfigs []string
	for _, configFile := range r.configFiles {
		expandedPath := kong.ExpandPath(configFile)
		if _, err := os.Stat(expandedPath); err == nil {
			loadedConfigs = append(loadedConfigs, expandedPath)
		}
	}

	r.globalConfig = &config.GlobalConfig{
		ColorOutput:   r.Color,
		ConfigPaths:   configPaths,
		Debug:         r.Debug,
		Verbose:       r.Verbose,
		DryRun:        r.DryRun,
		WorkerCount:   GetWorkerCount(r.Workers),
		RuntimeDir:    r.RuntimeDir,
		JSON:          r.JSON,
		FirstIs1:      r.FirstIs1,
		LoadedConfigs: loadedConfigs,
	}

	return nil
}

// handleHelpCommand converts "help" command to "-h" flag for better UX.
// Kong doesn't have a built-in help command, so we intercept it early.
func handleHelpCommand(args []string) []string {
	if len(args) > 0 && args[0] == "help" {
		// Replace "help" with "-h"
		newArgs := []string{"-h"}
		// If there are additional args after "help", keep them for subcommand help
		if len(args) > 1 {
			newArgs = append(newArgs, args[1:]...)
		}
		return newArgs
	}
	return args
}

// handleEarlyChangeDir pre-parses command line arguments for the -C flag
// and changes the working directory before Kong configuration loading.
// This ensures config files are loaded from the target directory, not the current directory.
//
// Supports formats: -C dir, -C=dir, --change-dir dir, --change-dir=dir
func handleChangeDir(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		var dir string

		// Check for various -C formats
		switch {
		case arg == "-C" || arg == "--change-dir":
			if i+1 < len(args) {
				dir = args[i+1]
				i++ // Skip next arg since we consumed it
			} else {
				return fmt.Errorf("%s flag requires a directory argument", arg)
			}
		case strings.HasPrefix(arg, "-C="):
			dir = strings.TrimPrefix(arg, "-C=")
		case strings.HasPrefix(arg, "--change-dir="):
			dir = strings.TrimPrefix(arg, "--change-dir=")
		}

		if dir != "" {
			if err := os.Chdir(dir); err != nil {
				return fmt.Errorf("failed to change directory to %s: %v", dir, err)
			}
			// Only process the first -C flag
			return nil
		}
	}
	return nil
}

func main() {
	var cli PlurCLI

	// Handle "help" command by converting it to "-h" flag
	args := handleHelpCommand(os.Args[1:])

	if err := handleChangeDir(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var configFiles []string
	if configFile := os.Getenv("PLUR_CONFIG_FILE"); configFile != "" {
		if _, err := os.Stat(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Config file specified in PLUR_CONFIG_FILE does not exist or is not readable: %s\n", configFile)
			os.Exit(1)
		}
		configFiles = append(configFiles, configFile)
	}

	configFiles = append(configFiles, ".plur.toml", "~/.plur.toml")

	cli.configFiles = configFiles

	parser, err := kong.New(&cli,
		kong.Name("plur"),
		kong.Description("A fast Go-based test runner for Ruby/RSpec"),
		kong.Configuration(kongtoml.Loader, configFiles...))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)

	logger.Logger.Debug("running plur", "args", os.Args[1:], "command", ctx.Command())
	err = ctx.Run(ctx)
	if err != nil {
		// Check if it's a custom exit code (don't log as error)
		var exitErr ExitCode
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		// Regular error - log and exit with code 1
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

// ExitCode is an error type that specifies a custom exit code
type ExitCode struct {
	Code int
}

func (e ExitCode) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
