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
					args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter", "--no-color"}
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
			PrintResults(summary)

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

func main() {
	app := createApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
