package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
)

// TaskConfig defines the structure for task configurations that Kong can parse from TOML
type TaskConfig struct {
	Description    string             `toml:"description"`     // Human-readable description
	Run            string             `toml:"run"`             // Command to run (e.g., "bundle exec rspec")
	SourceDirs     []string           `toml:"source_dirs"`     // Directories to watch/search
	Mappings       []task.MappingRule `toml:"mappings"`        // File mapping rules
	IgnorePatterns []string           `toml:"ignore_patterns"` // Patterns to ignore (for watch)
}

type SpecCmd struct {
	Patterns []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	Command  string   `help:"Test command to run" default:"bundle exec rspec"`
	Type     string   `short:"t" help:"Test framework type (rspec|minitest)" default:""`
}

// GetFramework returns the TestFramework enum based on the Type field
func (s *SpecCmd) GetFramework() config.TestFramework {
	return ParseFrameworkType(s.Type)
}

func (r *SpecCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	cfg := parent.globalConfig

	// Get the appropriate task with overrides applied
	// Use Type field directly as task name, fall back to framework detection
	taskName := r.Type
	if taskName == "" {
		framework := config.DetectTestFramework()
		taskName = string(framework)
	}

	// Only override command if it's not the default value
	commandOverride := ""
	if r.Command != "bundle exec rspec" { // Default value from struct tag
		commandOverride = r.Command
	}
	currentTask := parent.getTaskWithOverrides(taskName, commandOverride)
	framework := r.GetFramework() // Still needed for legacy file discovery
	logger.Logger.Debug("SpecCmd.Run", "command", currentTask.Run, "patterns", r.Patterns, "task", currentTask.Name)

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
			if framework == config.FrameworkMinitest {
				dir = "test"
			}
			return fmt.Errorf("no test files found (looking for *%s in %s/)", suffix, dir)
		}
	}
	msg := fmt.Sprintf("found %v test files", len(testFiles))
	logger.LogVerbose(msg, "testFiles", testFiles)

	// Run bundle install if --auto flag is set
	if cfg.Auto && !cfg.DryRun {
		depManager := NewDependencyManager()
		if err := depManager.InstallDependencies(); err != nil {
			return err
		}
	}

	// Update SpecCmd to use task command
	r.Command = currentTask.Run

	// Create and run executor
	executor := NewTestExecutor(cfg, r, testFiles)
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
	Find    WatchFindCmd    `cmd:"" help:"Find and suggest mappings for files"`
}

type WatchRunCmd struct {
	// Flags for watch command
	Timeout  int    `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int    `help:"Debounce delay in milliseconds" default:"100"`
	Command  string `help:"Test command to run" default:"bundle exec rspec"`
	Type     string `short:"t" help:"Test framework type (rspec|minitest)" default:""`
}

// GetFramework returns the TestFramework enum based on the Type field
func (w *WatchRunCmd) GetFramework() config.TestFramework {
	return ParseFrameworkType(w.Type)
}

func (w *WatchRunCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig

	// Auto-install watcher binary if needed
	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *PlurCLI) error {
	return runWatchInstall(true)
}

type DoctorCmd struct{}

func (d *DoctorCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	return runDoctorWithConfig(parent.globalConfig)
}

type DBSetupCmd struct{}

func (d *DBSetupCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:setup", config)
}

type DBCreateCmd struct{}

func (d *DBCreateCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:create", config)
}

type DBMigrateCmd struct{}

func (d *DBMigrateCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:migrate", config)
}

type DBPrepareCmd struct{}

func (d *DBPrepareCmd) Run(parent *PlurCLI) error {
	// Use the pre-built global config
	config := parent.globalConfig
	return RunDatabaseTask("db:test:prepare", config)
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

	// Global flags
	Auto       bool   `help:"Automatically run bundle install before tests" default:"false"`
	Verbose    bool   `help:"Enable verbose output for debugging" default:"false"`
	Debug      bool   `short:"d" help:"Enable debug output (includes verbose)" env:"PLUR_DEBUG" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	Colour     bool   `help:"Force colorized output (British spelling)" negatable:"" default:"true" hidden:""`
	RuntimeDir string `help:"Custom directory for runtime data" default:""`
	// ChangeDir is kept for Kong's help text and CLI compatibility, but the actual
	// directory change is handled early in main() before config loading
	ChangeDir string `short:"C" help:"Change to directory before running (like git -C)" default:""`
	Workers   int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" env:"PARALLEL_TEST_PROCESSORS" default:"0"`
	FirstIs1  bool   `help:"Start TEST_ENV_NUMBER at 1 instead of empty string (default: true)" negatable:"" default:"true"`
	Version   bool   `help:"Show version information"`

	// Task configurations from [task.NAME] sections in TOML - parsed by Kong
	Task map[string]TaskConfig `help:"Task configurations (config file only)"`

	// Processed task configurations (converted from TaskConfig)
	Tasks map[string]*task.Task `kong:"-"`

	// Store the built global config
	globalConfig *config.GlobalConfig `kong:"-"`
}

func (r *PlurCLI) AfterApply() error {
	// Initialize logger early so we can use it
	// Kong has already resolved r.Debug from CLI flag, env var, or config file
	logger.InitLogger(r.Verbose, r.Debug)

	// Note: Directory change is now handled before Kong initialization in main()

	// Handle version flag
	if r.Version {
		fmt.Println(GetVersionInfo())
		os.Exit(0)
	}

	// Sync British spelling to American spelling
	// Both default to true, so if either is false, use false
	// This handles --no-color, --no-colour, or both
	if !r.Colour || !r.Color {
		r.Color = false
	}

	// Initialize config paths
	configPaths := config.InitConfigPaths()

	// Build global config once
	r.globalConfig = &config.GlobalConfig{
		Auto:        r.Auto,
		ColorOutput: r.Color,
		ConfigPaths: configPaths,
		Debug:       r.Debug,
		Verbose:     r.Verbose,
		DryRun:      r.DryRun,
		WorkerCount: GetWorkerCount(r.Workers),
		RuntimeDir:  r.RuntimeDir,
		JSON:        r.JSON,
		FirstIs1:    r.FirstIs1,
	}

	// Convert TaskConfig map to task.Task map
	r.Tasks = make(map[string]*task.Task)
	for taskName, taskConfig := range r.Task {
		// Convert TaskConfig to task.Task
		taskObj := &task.Task{
			Name:           taskName,
			Description:    taskConfig.Description,
			Run:            taskConfig.Run,
			SourceDirs:     taskConfig.SourceDirs,
			Mappings:       taskConfig.Mappings,
			IgnorePatterns: taskConfig.IgnorePatterns,
		}
		r.Tasks[taskName] = taskObj
	}

	return nil
}

// getTaskWithOverrides returns the appropriate task with CLI/config overrides applied
func (r *PlurCLI) getTaskWithOverrides(taskName string, commandOverride string) *task.Task {
	var baseTask *task.Task

	// Try to get task from TOML config first
	if configTask, exists := r.Tasks[taskName]; exists {
		// Clone the config task to avoid modifying the original
		clonedTask := *configTask
		baseTask = &clonedTask
	} else {
		// Fall back to default task based on framework detection
		switch taskName {
		case "rspec":
			baseTask = task.NewRSpecTask()
		case "minitest":
			baseTask = task.NewMinitestTask()
		default:
			// Auto-detect framework and fall back to RSpec
			framework := config.DetectTestFramework()
			if framework == config.FrameworkMinitest {
				baseTask = task.NewMinitestTask()
			} else {
				baseTask = task.NewRSpecTask() // Default fallback
			}
		}
	}

	// Apply command override if provided
	if commandOverride != "" {
		baseTask.Run = commandOverride
	}

	return baseTask
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

	// Handle -C flag early to ensure config files are loaded from the correct directory
	if err := handleChangeDir(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build config file list
	var configFiles []string

	// Check for PLUR_CONFIG_FILE environment variable
	if configFile := os.Getenv("PLUR_CONFIG_FILE"); configFile != "" {
		// Verify the file exists and is readable
		if _, err := os.Stat(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Config file specified in PLUR_CONFIG_FILE does not exist or is not readable: %s\n", configFile)
			os.Exit(1)
		}
		// Add it first for highest precedence
		configFiles = append(configFiles, configFile)
	}

	// Always append default locations after
	configFiles = append(configFiles, ".plur.toml", "~/.plur.toml")

	// Create parser with configuration
	parser, err := kong.New(&cli,
		kong.Name("plur"),
		kong.Description("A fast Go-based test runner for Ruby/RSpec"),
		kong.Configuration(kongtoml.Loader, configFiles...))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Parse command line arguments
	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	logger.Logger.Debug("running plur", "args", os.Args[1:], "command", ctx.Command())
	err = ctx.Run(ctx)
	if err != nil {
		logger.Logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}
