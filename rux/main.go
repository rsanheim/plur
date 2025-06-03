package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rsanheim/rux/rspec"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

func createApp() *cli.App {
	return &cli.App{
		Name:    "rux",
		Usage:   "A fast Go-based test runner for Ruby/RSpec",
		Version: GetVersionInfo(),
		Commands: []*cli.Command{
			{
				Name:  "watch",
				Usage: "Watch for file changes and run tests automatically",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "timeout",
						Usage: "Exit after specified seconds (default: run until Ctrl-C)",
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
				Name:  "db:setup",
				Usage: "Setup test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "n",
						Aliases: []string{"workers"},
						Usage:   "Number of parallel workers",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
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
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
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
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
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
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
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
		},
		Action: func(ctx *cli.Context) error {
			// Initialize tracing if enabled
			if ctx.Bool("trace") {
				if err := InitTracer(true); err != nil {
					return fmt.Errorf("failed to initialize tracer: %v", err)
				}
				defer CloseTracer()
			}

			// Set custom runtime directory if provided
			if runtimeDir := ctx.String("runtime-dir"); runtimeDir != "" {
				customRuntimeDir = runtimeDir
			}

			defer TraceFunc("main.total_execution")()

			var specFiles []string
			var err error

			// Determine which spec files to run
			func() {
				defer TraceFunc("file_discovery")()

				if ctx.NArg() > 0 {
					// Expand glob patterns from provided arguments
					specFiles, err = ExpandGlobPatterns(ctx.Args().Slice())
					if err != nil {
						return
					}
					if len(specFiles) == 0 {
						err = fmt.Errorf("no spec files found matching provided patterns")
						return
					}
				} else {
					// Auto-discover spec files
					specFiles, err = FindSpecFiles()
					if err != nil {
						return
					}
					if len(specFiles) == 0 {
						err = fmt.Errorf("no spec files found")
						return
					}
				}
			}()

			if err != nil {
				return err
			}

			dryRun := ctx.Bool("dry-run")

			// Print version as first line (for both dry-run and normal)
			fmt.Printf("rux version %s\n", GetVersionInfo())

			if dryRun {
				if ctx.Bool("auto") {
					fmt.Fprintln(os.Stderr, "[dry-run] bundle install")
				}

				fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(specFiles))

				// Get formatter path for dry-run display
				cacheDir, err := getRuxCacheDir()
				var formatterPath string
				if err != nil {
					formatterPath = "~/.cache/rux/formatters/json_rows_formatter.rb"
				} else {
					formatterPath, err = rspec.GetFormatterPath(cacheDir)
					if err != nil {
						formatterPath = "~/.cache/rux/formatters/json_rows_formatter.rb"
					}
				}

				// Show grouped execution in dry-run
				workerCount := GetWorkerCount(ctx.Int("n"))

				// Load runtime data if available
				runtimeData, err := LoadRuntimeData()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Could not load runtime data: %v\n", err)
					runtimeData = make(map[string]float64)
				}

				// Always use grouping
				var groups []FileGroup
				if len(runtimeData) > 0 {
					groups = GroupSpecFilesByRuntime(specFiles, workerCount, runtimeData)
					fmt.Fprintf(os.Stderr, "[dry-run] Using runtime-based grouped execution: %d groups\n", len(groups))
				} else {
					groups = GroupSpecFilesBySize(specFiles, workerCount)
					fmt.Fprintf(os.Stderr, "[dry-run] Using size-based grouped execution: %d groups\n", len(groups))
				}
				for i, group := range groups {
					args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
					// Add color flags based on preference
					if !shouldUseColor(ctx) {
						args = append(args, "--no-color")
					} else {
						args = append(args, "--force-color", "--tty")
					}
					args = append(args, group.Files...)
					fmt.Fprintf(os.Stderr, "[dry-run] Worker %d: %s\n", i, strings.Join(args, " "))
				}
				return nil
			}

			// Run bundle install if --auto flag is set
			if ctx.Bool("auto") {
				defer TraceFunc("bundle_install")()

				fmt.Println("Installing dependencies...")
				bundleCmd := exec.Command("bundle", "install")
				bundleCmd.Stdout = os.Stdout
				bundleCmd.Stderr = os.Stderr

				if err := bundleCmd.Run(); err != nil {
					return fmt.Errorf("error running bundle install: %v", err)
				}
			}

			workerCount := GetWorkerCount(ctx.Int("n"))
			actualWorkers := workerCount
			if len(specFiles) < workerCount {
				actualWorkers = len(specFiles)
			}

			fmt.Printf("Running %d spec files in parallel using %d workers (%d cores available)...\n",
				len(specFiles), actualWorkers, runtime.NumCPU())

			// Determine color output settings
			colorOutput := shouldUseColor(ctx)

			// Always initialize runtime tracker
			runtimeTracker := NewRuntimeTracker()

			// Run specs in parallel with intelligent grouping
			results, wallTime := RunSpecsInParallel(specFiles, dryRun, colorOutput, workerCount, runtimeTracker)

			// Save runtime data
			if err := runtimeTracker.SaveToFile(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save runtime data: %v\n", err)
			} else {
				if runtimePath, err := GetRuntimeFilePath(); err == nil {
					fmt.Fprintf(os.Stderr, "Runtime data saved to: %s\n", runtimePath)
				}
			}

			// Build summary and print results
			summary := BuildTestSummary(results, wallTime)
			PrintResults(summary, colorOutput)

			// Exit with error if any tests failed
			if !summary.Success {
				os.Exit(1)
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
	app := createApp()
	// Reorder arguments to put flags before positional args
	args := reorderArgs(os.Args)
	if err := app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
