package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
)

// TaskConfig defines the structure for task configurations that Kong can parse from TOML
type TaskConfig struct {
	Description string   `toml:"description"` // Human-readable description
	Run         string   `toml:"run"`         // Command to run (e.g., "bundle exec rspec")
	SourceDirs  []string `toml:"source_dirs"` // Directories to watch/search
	TestGlob    string   `toml:"test_glob"`   // Glob pattern for test files
}

type SpecCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	Use      string   `short:"t" help:"Task to run (rspec/minitest/custom)" default:""`
	Auto     bool     `help:"Automatically run bundle install before tests" default:"false"`
}

func (r *SpecCmd) Run(parent *PlurCLI) error {
	cfg := parent.globalConfig

	// Get the appropriate task with overrides applied
	// Priority: CLI --use, config use, auto-detect
	taskName := r.Use
	wasExplicit := r.Use != ""
	if taskName == "" && parent.Use != "" {
		taskName = parent.Use
		wasExplicit = true
	}
	if taskName == "" {
		detectedTask := task.DetectFramework()
		taskName = detectedTask.Name
		wasExplicit = false
	}

	// Validate task exists if explicitly requested
	if err := parent.validateTaskExists(taskName, wasExplicit); err != nil {
		return err
	}

	currentTask := parent.getTaskWithOverrides(taskName)
	logger.Logger.Debug("SpecCmd.Run", "command", currentTask.Run, "patterns", r.Patterns, "task", currentTask.Name)

	// Show hint if both frameworks exist and we auto-detected
	if !wasExplicit && task.BothFrameworksExist() {
		otherFramework := "minitest"
		if currentTask.Name == "minitest" {
			otherFramework = "rspec"
		}
		logger.LogVerbose(fmt.Sprintf("Both spec/ and test/ directories detected. Using %s. Use -t %s to run %s instead.",
			currentTask.Name, otherFramework, otherFramework))
	}

	// Discover test files
	var testFiles []string
	var err error
	if len(r.Patterns) > 0 {
		testFiles, err = ExpandGlobPatterns(r.Patterns, currentTask)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			return fmt.Errorf("no test files found matching provided patterns")
		}
	} else {
		testFiles, err = FindTestFiles(currentTask)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			suffix := currentTask.GetTestSuffix()
			// Determine directory from task's test pattern
			pattern := currentTask.GetTestPattern()
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
	executor := NewTestExecutor(cfg, testFiles, currentTask)
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
	Find    WatchFindCmd    `cmd:"" help:"Placeholder - functionality being rebuilt"`
}

type WatchRunCmd struct {
	Timeout  int    `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int    `help:"Debounce delay in milliseconds" default:"100"`
	Use      string `short:"t" help:"Task to run (rspec/minitest/custom)" default:""`
}

func (w *WatchRunCmd) Run(parent *PlurCLI) error {
	config := parent.globalConfig

	// Get the appropriate task with overrides applied
	// Priority: CLI --use, config use, auto-detect
	taskName := w.Use
	wasExplicit := w.Use != ""
	if taskName == "" && parent.Use != "" {
		taskName = parent.Use
		wasExplicit = true
	}
	if taskName == "" {
		detectedTask := task.DetectFramework()
		taskName = detectedTask.Name
		wasExplicit = false
	}

	// Validate task exists if explicitly requested
	if err := parent.validateTaskExists(taskName, wasExplicit); err != nil {
		return err
	}

	currentTask := parent.getTaskWithOverrides(taskName)

	// Show hint if both frameworks exist and we auto-detected
	if !wasExplicit && task.BothFrameworksExist() {
		otherFramework := "minitest"
		if currentTask.Name == "minitest" {
			otherFramework = "rspec"
		}
		logger.LogVerbose(fmt.Sprintf("Both spec/ and test/ directories detected. Using %s. Use -t %s to run %s instead.",
			currentTask.Name, otherFramework, otherFramework))
	}

	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w, currentTask)
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
	Use        string `help:"Default task configuration to use" default:"" hidden:""`
	Verbose    bool   `short:"v" help:"Enable verbose output for debugging" default:"false"`
	Version    bool   `help:"Show version information"`
	Workers    int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" env:"PARALLEL_TEST_PROCESSORS" default:"0"`

	// Task configurations from [task.NAME] sections in TOML - parsed by Kong
	Task map[string]TaskConfig `help:"Task configurations (config file only)" hidden:""`

	// Processed task configurations (converted from TaskConfig)
	Tasks map[string]*task.Task `kong:"-"`

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

	// Convert TaskConfig map to task.Task map
	r.Tasks = make(map[string]*task.Task)
	for taskName, taskConfig := range r.Task {
		// Convert TaskConfig to task.Task
		taskObj := &task.Task{
			Name:        taskName,
			Description: taskConfig.Description,
			Run:         taskConfig.Run,
			SourceDirs:  taskConfig.SourceDirs,
			TestGlob:    taskConfig.TestGlob,
		}
		r.Tasks[taskName] = taskObj
	}

	return nil
}

// validateTaskExists checks if a task exists when explicitly requested
// Returns nil if task exists or was auto-detected
// Returns error with available tasks if explicitly requested task doesn't exist
func (r *PlurCLI) validateTaskExists(taskName string, wasExplicit bool) error {
	if !wasExplicit {
		return nil // Auto-detected tasks are always valid
	}

	// Check built-in tasks
	if taskName == "rspec" || taskName == "minitest" {
		return nil
	}

	// Check custom tasks from config
	if _, exists := r.Tasks[taskName]; exists {
		return nil
	}

	// Task not found - build helpful error message with deduplication
	availableMap := make(map[string]bool)
	availableMap["rspec"] = true
	availableMap["minitest"] = true
	for name := range r.Tasks {
		availableMap[name] = true
	}

	// Convert to sorted slice for consistent output
	available := make([]string, 0, len(availableMap))
	for name := range availableMap {
		available = append(available, name)
	}
	sort.Strings(available)

	return fmt.Errorf("task '%s' not found. Available tasks: %s",
		taskName, strings.Join(available, ", "))
}

// mergeTaskConfig merges non-empty fields from override into base task
func (r *PlurCLI) mergeTaskConfig(base *task.Task, override *task.Task) {
	// Always preserve Name from override (custom task name, not detected framework name)
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.Run != "" {
		base.Run = override.Run
	}
	if len(override.SourceDirs) > 0 {
		base.SourceDirs = override.SourceDirs
	}
	if override.TestGlob != "" {
		base.TestGlob = override.TestGlob
	}
}

// getTaskWithOverrides returns the appropriate task with CLI/config overrides applied
func (r *PlurCLI) getTaskWithOverrides(taskName string) *task.Task {
	var baseTask *task.Task

	// Start with appropriate default task
	switch taskName {
	case "rspec":
		baseTask = task.NewRSpecTask()
	case "minitest":
		baseTask = task.NewMinitestTask()
	default:
		// Custom tasks inherit from auto-detected framework
		baseTask = task.DetectFramework()
	}

	// Merge TOML config overrides if they exist (for both built-in and custom tasks)
	if configTask, exists := r.Tasks[taskName]; exists {
		r.mergeTaskConfig(baseTask, configTask)
	}

	return baseTask
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
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
