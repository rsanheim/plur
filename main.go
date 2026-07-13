package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/cmd"
	"github.com/rsanheim/plur/config"
	clihelp "github.com/rsanheim/plur/internal/cli"
	"github.com/rsanheim/plur/internal/framework"
	kongtoml "github.com/rsanheim/plur/internal/kongtoml"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/internal/term"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

type SpecCmd struct {
	Patterns        []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	Tags            []string `help:"Filter RSpec by tag (repeatable)" name:"tag"`
	ExcludePatterns []string `help:"Exclude test files matching glob (repeatable)" name:"exclude-pattern"`
	Auto            bool     `help:"Automatically run bundle install before tests" default:"false"`
	RspecTrace      bool     `help:"Prefix stdout/stderr with source file path (RSpec only)" default:"false" name:"rspec-trace"`
	RspecSplit      bool     `help:"EXPERIMENTAL: split long-running RSpec files into focused file:line runs" name:"rspec-split" env:"PLUR_RSPEC_SPLIT" default:"false"`
}

type WorkerCount int

func (w WorkerCount) Validate() error {
	if w < 1 {
		return errors.New("workers must be at least 1")
	}
	return nil
}

type WatchCmd struct {
	Run     WatchRunCmd     `cmd:"" default:"withargs" group:"daily" help:"Run watch mode"`
	Install WatchInstallCmd `cmd:"" group:"advanced" help:"Install the watcher binary"`
	Find    WatchFindCmd    `cmd:"" group:"daily" help:"Show what would be executed for a given file change"`

	Ignore []string `help:"Patterns to ignore from watch events (default: .git/**, node_modules/**)" name:"ignore"`
}

type WatchRunCmd struct {
	Timeout  int `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int `help:"Debounce delay in milliseconds" default:"30"`
}

func (w *WatchRunCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	config := globals.globalConfig

	if err := runWatchInstall(false); err != nil {
		return err
	}

	return runWatchWithConfig(config, w, parent, globals)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *PlurCLI) error {
	return runWatchInstall(true)
}

type DoctorCmd struct{}

func (d *DoctorCmd) Run(parent *PlurCLI) error {
	return runDoctorWithConfig(parent.globalConfig, parent.runtimeConfig)
}

type ConfigCmd struct {
	Init ConfigInitCmd `cmd:"" group:"advanced" help:"Generate a starter configuration file"`
}

type PlurCLI struct {
	Spec       SpecCmd        `cmd:"" group:"daily" help:"Run tests" default:"withargs"`
	Watch      WatchCmd       `cmd:"" help:"Watch for file changes and run tests automatically"`
	Rails      RailsCmd       `cmd:"" name:"rails" aliases:"rake" group:"advanced" help:"Run a Rails or Rake command once per worker"`
	Doctor     DoctorCmd      `cmd:"" group:"advanced" help:"Diagnose Plur installation and environment"`
	Config     ConfigCmd      `cmd:"" help:"Configuration commands"`
	RailsInit  RailsInitCmd   `cmd:"" name:"rails:init" group:"advanced" help:"Configure a Rails project for parallel testing"`
	VersionCmd cmd.VersionCmd `cmd:"" name:"version" group:"advanced" help:"Show version information"`

	// ChangeDir is kept for Kong's help text and CLI compatibility, but the actual
	// directory change is handled early in main() before config loading
	ChangeDir string      `short:"C" help:"Change to directory before running (like git -C)" default:""`
	Color     string      `help:"When to color output: auto (detect terminal), always, or never" enum:"auto,always,never,on,off" env:"PLUR_COLOR" default:"auto"`
	Debug     bool        `short:"d" help:"Enable debug output (includes verbose)" env:"PLUR_DEBUG" default:"false"`
	DryRun    bool        `help:"Print what would be executed without running" default:"false"`
	FirstIs1  bool        `help:"Start TEST_ENV_NUMBER at 1 instead of empty string (default: true)" negatable:"" default:"true"`
	JSON      string      `help:"Save detailed test results as JSON to the specified file" default:"" hidden:""`
	Use       string      `short:"u" help:"Job to use (overrides autodetection)" default:""`
	Verbose   bool        `short:"v" help:"Enable verbose output for debugging" default:"false"`
	Version   bool        `help:"Show version information"`
	Workers   WorkerCount `short:"n" help:"Number of parallel workers" env:"PLUR_WORKERS,PARALLEL_TEST_PROCESSORS" default:"4"`

	// Job and watch configuration
	Job           map[string]framework.Job `help:"Job configurations (config file only)" hidden:""`
	WatchMappings []watch.WatchMapping     `help:"Watch mappings (config file only)" hidden:"" name:"watch" toml:"watch"`

	// Store the built global config
	globalConfig  *config.GlobalConfig   `kong:"-"`
	runtimeConfig *runtime.RuntimeConfig `kong:"-"`

	// Store config files that were attempted (for tracking)
	configFiles []string `kong:"-"`

	// RSpec passthrough args from -- delimiter
	passthroughArgs []string `kong:"-"`
}

func (cli *PlurCLI) Validate() error {
	if err := cli.Workers.Validate(); err != nil {
		return fmt.Errorf("--workers: %w", err)
	}
	return nil
}

// Initialize logger with appropriate level
// At this point, Kong has already resolved r.Debug and r.Verbose
func (cli *PlurCLI) AfterApply() error {
	level := slog.LevelWarn
	if cli.Debug {
		level = slog.LevelDebug
	} else if cli.Verbose {
		level = slog.LevelInfo
	}
	logger.Init(level)

	if cli.Version {
		err := (&cmd.VersionCmd{}).Run()
		if err != nil {
			return err
		}
	}

	configPaths := config.InitConfigPaths()

	var loadedConfigs []string
	for _, configFile := range cli.configFiles {
		expandedPath := kong.ExpandPath(configFile)
		if _, err := os.Stat(expandedPath); err == nil {
			loadedConfigs = append(loadedConfigs, expandedPath)
		}
	}

	colorOn, colorSource := term.ResolveColor(cli.Color, os.LookupEnv, term.IsStdoutTTY())
	slog.Info("color output resolved", "mode", cli.Color, "enabled", colorOn, "source", colorSource)

	cli.globalConfig = &config.GlobalConfig{
		ColorOutput:   colorOn,
		ColorSource:   colorSource,
		ConfigPaths:   configPaths,
		Debug:         cli.Debug,
		Verbose:       cli.Verbose,
		DryRun:        cli.DryRun,
		WorkerCount:   int(cli.Workers),
		RuntimeDir:    configPaths.RuntimeDir,
		JSON:          cli.JSON,
		FirstIs1:      cli.FirstIs1,
		RspecSplit:    cli.Spec.RspecSplit,
		LoadedConfigs: loadedConfigs,
	}

	if err := validateUniqueWatchNames(cli.WatchMappings, loadedConfigs); err != nil {
		return err
	}

	cliInput := &runtime.CLIInput{
		Use:           cli.Use,
		Jobs:          cli.Job,
		WatchMappings: cli.WatchMappings,
		ConfigFiles:   cli.configFiles,
	}
	runtimeConfig, err := runtime.BuildRuntimeConfig(cliInput)
	if err != nil {
		return err
	}
	cli.runtimeConfig = runtimeConfig

	return nil
}

func validateUniqueWatchNames(watches []watch.WatchMapping, sources []string) error {
	seenNames := make(map[string]struct{})
	for _, w := range watches {
		if w.Name == "" {
			continue
		}
		if _, exists := seenNames[w.Name]; exists {
			return fmt.Errorf("configuration error in %v: duplicate watch name %q; named [[watch]] entries must be unique", sources, w.Name)
		}
		seenNames[w.Name] = struct{}{}
	}
	return nil
}

// handleHelpCommand converts "help" command to "-h" flag for better UX.
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

func splitArgsAtDoubleDash(args []string) ([]string, []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
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
				return fmt.Errorf("failed to change directory to %s: %w", dir, err)
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
	args, cli.passthroughArgs = splitArgsAtDoubleDash(args)
	logger.InitFromArgs(args)

	if err := handleChangeDir(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	configFiles := []string{"~/.plur.toml", ".plur.toml"}
	if configFile := os.Getenv("PLUR_CONFIG_FILE"); configFile != "" {
		if _, err := os.Stat(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Config file specified in PLUR_CONFIG_FILE does not exist or is not readable: %s\n", configFile)
			os.Exit(1)
		}
		configFiles = append(configFiles, configFile)
	}

	cli.configFiles = configFiles

	parser, err := kong.New(&cli,
		kong.Name("plur"),
		kong.Description("A fast, parallel test runner and watcher for Ruby/RSpec"),
		kong.ExplicitGroups([]kong.Group{
			{Key: "daily", Title: "Daily commands"},
			{Key: "advanced", Title: "Advanced and setup commands"},
		}),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true, FlagsLast: true}),
		clihelp.ConfigureHelpDetails(),
		kong.Help(clihelp.HelpPrinter),
		kong.Configuration(colorAwareLoader, configFiles...))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(args)
	parser.FatalIfErrorf(retiredColorFlagHint(err))

	if len(cli.passthroughArgs) > 0 && !commandSupportsPassthrough(ctx.Command()) {
		fmt.Fprintln(os.Stderr, "Error: passthrough args via -- are only supported for the spec, rails, and rake commands")
		os.Exit(1)
	}

	err = ctx.Run(ctx)
	if err != nil {
		// Check if it's a custom exit code (don't log as error)
		var exitErr ExitCode
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func commandSupportsPassthrough(command string) bool {
	return strings.HasPrefix(command, "spec") || strings.HasPrefix(command, "rails")
}

// colorAwareLoader wraps the TOML config loader to keep the color key's
// precedence correct: NO_COLOR outranks config files, and the retired boolean
// form fails with a useful error instead of kong's raw type-mismatch message.
func colorAwareLoader(r io.Reader) (kong.Resolver, error) {
	resolver, err := kongtoml.Loader(r)
	if err != nil {
		return nil, err
	}
	return colorConfigResolver{resolver}, nil
}

type colorConfigResolver struct{ kong.Resolver }

func (r colorConfigResolver) Resolve(kctx *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
	value, err := r.Resolver.Resolve(kctx, parent, flag)
	if err != nil || value == nil || flag.Name != "color" {
		return value, err
	}
	if _, isBool := value.(bool); isBool {
		return nil, errors.New(`booleans are no longer supported; use "auto", "always", or "never"`)
	}
	if term.EnvDecidesColor(os.LookupEnv) {
		return nil, nil // env outranks the config file for color
	}
	return value, nil
}

// retiredColorFlagHint rewrites kong's generic parse errors for the removed
// bare --color and --no-color forms into messages that point at the new
// --color=auto|always|never syntax. Any other error passes through unchanged.
// (An explicit bad value like --color=purple keeps kong's enum error, which
// already lists the valid choices.)
func retiredColorFlagHint(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown flag --no-color"):
		return usageError("--no-color is no longer supported; use --color=never")
	case strings.Contains(msg, "--color") && strings.Contains(msg, "expected string value"):
		return usageError("--color needs a value; use --color=auto, --color=always, or --color=never")
	}
	return err
}

// usageError is a CLI usage error that reports kong's usage-error exit status
// (80, per https://github.com/square/exit) via kong's ExitCoder interface, so
// rewritten flag errors exit consistently with kong's own parse errors.
type usageError string

func (e usageError) Error() string { return string(e) }
func (e usageError) ExitCode() int { return 80 }

type ExitCode struct {
	Code int
}

func (e ExitCode) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
