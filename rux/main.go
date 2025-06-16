package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/tracing"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

var ruxConfig *Config

func createApp() *cli.App {
	return &cli.App{
		Name:    "rux",
		Usage:   "A fast Go-based test runner for Ruby/RSpec",
		Version: GetVersionInfo(),
		Before: func(ctx *cli.Context) error {
			// Initialize logging globally before any command runs
			debug := ctx.Bool("debug") || os.Getenv("RUX_DEBUG") == "1"
			logger.InitLogger(ctx.Bool("verbose"), debug)

			configPaths := InitConfigPaths()
			ruxConfig = BuildConfig(ctx, configPaths)
			logger.Logger.Debug("initial config", "config", ruxConfig)

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "watch",
				Usage: "Watch for file changes and run tests automatically",
				Before: func(ctx *cli.Context) error {
					return runWatchInstall(false)
				},
				Subcommands: []*cli.Command{
					{
						Name:  "install",
						Usage: "Install the watcher binary",
						Action: func(ctx *cli.Context) error {
							return runWatchInstall(true)
						},
					},
				},
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "timeout",
						Usage: "Exit after specified seconds (default: run until Ctrl-C)",
					},
					&cli.IntFlag{
						Name:  "debounce",
						Usage: "Debounce delay in milliseconds (default: 100)",
						Value: 100,
					},
				},
				Action: func(ctx *cli.Context) error {
					return runWatch(ctx)
				},
			},
			{
				Name:  "doctor",
				Usage: "Show diagnostic information about rux installation",
				Action: func(ctx *cli.Context) error {
					return runDoctor(ctx)
				},
			},
			{
				Name:      "dev:file_mapper",
				Usage:     "Test file mapping - shows which specs would run for given files",
				ArgsUsage: "[files...]",
				Action: func(ctx *cli.Context) error {
					return runFileMapper(ctx)
				},
			},
			{
				Name:  "db:setup",
				Usage: "Setup test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "n",
						Aliases: []string{"workers"},
						Usage:   "Number of parallel workers",
					},
				},
				Action: func(ctx *cli.Context) error {
					return runDatabaseTask("db:setup", ctx)
				},
			},
			{
				Name:  "db:create",
				Usage: "Create test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "n",
						Aliases: []string{"workers"},
						Usage:   "Number of parallel workers",
					},
				},
				Action: func(ctx *cli.Context) error {
					return runDatabaseTask("db:create", ctx)
				},
			},
			{
				Name:  "db:migrate",
				Usage: "Migrate test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "n",
						Aliases: []string{"workers"},
						Usage:   "Number of parallel workers",
					},
				},
				Action: func(ctx *cli.Context) error {
					return runDatabaseTask("db:migrate", ctx)
				},
			},
			{
				Name:  "db:test:prepare",
				Usage: "Prepare test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "n",
						Aliases: []string{"workers"},
						Usage:   "Number of parallel workers",
					},
				},
				Action: func(ctx *cli.Context) error {
					return runDatabaseTask("db:test:prepare", ctx)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Print what would be executed without running",
			},
			&cli.BoolFlag{
				Name:  "auto",
				Usage: "Run bundle install if necessary before running tests",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Save detailed test results to JSON files",
			},
			&cli.BoolFlag{
				Name:    "color",
				Aliases: []string{"colour"},
				Usage:   "Force colorized output (default: auto-detect TTY)",
				Value:   true,
			},
			&cli.BoolFlag{
				Name:    "no-color",
				Aliases: []string{"no-colour"},
				Usage:   "Disable colorized output",
			},
			&cli.IntFlag{
				Name:    "n",
				Aliases: []string{"workers"},
				Usage:   "Number of parallel workers (default: cores-2, env: PARALLEL_TEST_PROCESSORS)",
			},
			&cli.BoolFlag{
				Name:  "trace",
				Usage: "Enable performance tracing to analyze execution time",
			},
			&cli.StringFlag{
				Name:  "runtime-dir",
				Usage: "Directory to store runtime data (default: ~/.cache/rux/runtimes)",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose output for debugging",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug output (includes verbose)",
			},
		},
		Action: func(ctx *cli.Context) error {
			// Initialize tracing if enabled
			if ctx.Bool("trace") {
				if err := tracing.Init(true); err != nil {
					return fmt.Errorf("failed to initialize tracer: %v", err)
				}
				defer tracing.Close()
			}

			defer tracing.StartRegion(context.Background(), "main.total_execution")()

			// Discover spec files for the main command
			specFiles, err := discoverSpecFiles(ctx)
			if err != nil {
				return err
			}

			// Run bundle install if --auto flag is set
			if ruxConfig.Auto && !ruxConfig.DryRun {
				depManager := NewDependencyManager()
				if err := depManager.InstallDependencies(); err != nil {
					return err
				}
			}

			// Create and run executor
			executor := NewTestExecutor(ruxConfig, specFiles)
			if err := executor.Execute(); err != nil {
				// Exit with error code 1 for test failures
				if strings.Contains(err.Error(), "test run failed") {
					os.Exit(1)
				}
				return err
			}

			return nil
		},
	}
}

func runDatabaseTask(task string, ctx *cli.Context) error {
	workerCount := GetWorkerCount(ctx.Int("n"))
	dryRun := ctx.Bool("dry-run")

	return RunDatabaseTask(task, workerCount, dryRun)
}

// shouldUseColor determines if colorized output should be used
func shouldUseColor(ctx *cli.Context) bool {
	// If --no-color or --no-colour is set, disable color
	if ctx.Bool("no-color") {
		return false
	}

	// If --color or --colour is set, enable color
	if ctx.Bool("color") {
		return true
	}

	// Auto-detect: use color if output is a TTY and FORCE_COLOR is set or TTY is detected
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}

// reorderArgs moves flags before positional arguments to work around urfave/cli v2 limitation
// This allows both `rux --no-color spec/` and `rux spec/ --no-color` to work
func reorderArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	// Check if we have a subcommand (watch, doctor, etc)
	hasSubcommand := false
	subcommandIndex := -1
	for i := 1; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			// This might be a subcommand
			for _, cmd := range []string{"watch", "doctor", "db:setup", "db:create", "db:migrate", "db:test:prepare"} {
				if args[i] == cmd {
					hasSubcommand = true
					subcommandIndex = i
					break
				}
			}
			if hasSubcommand {
				break
			}
		}
	}

	// If we have a subcommand, don't reorder after it
	if hasSubcommand {
		// Only reorder global flags before the subcommand
		result := []string{args[0]}
		var globalFlags []string
		var beforeSubcommand []string

		for i := 1; i < subcommandIndex; i++ {
			if strings.HasPrefix(args[i], "-") {
				globalFlags = append(globalFlags, args[i])
				// Handle flag values
				if i+1 < subcommandIndex && !strings.HasPrefix(args[i+1], "-") {
					i++
					globalFlags = append(globalFlags, args[i])
				}
			} else {
				beforeSubcommand = append(beforeSubcommand, args[i])
			}
		}

		result = append(result, globalFlags...)
		result = append(result, beforeSubcommand...)
		// Add subcommand and everything after it unchanged
		result = append(result, args[subcommandIndex:]...)
		return result
	}

	// Original logic for when there's no subcommand
	cmd := args[0]
	var flags []string
	var positional []string

	for i := 1; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if this flag takes a value
			if (arg == "-n" || arg == "--workers" || arg == "--runtime-dir") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}

	result := []string{cmd}
	result = append(result, flags...)
	result = append(result, positional...)
	return result
}

func main() {
	// Use urfave/cli only if explicitly requested
	if os.Getenv("URFAVE") == "1" {
		app := createApp()
		// Reorder arguments to put flags before positional args
		args := reorderArgs(os.Args)
		if err := app.Run(args); err != nil {
			logger.Logger.Error("Application error", "error", err)
			os.Exit(1)
		}
		return
	}

	// Default to Kong CLI
	runKongCLI()
}
