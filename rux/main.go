package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
)

func createApp() *cli.App {
	return &cli.App{
		Name:  "rux",
		Usage: "A fast Go-based test runner for Ruby/RSpec",
		Commands: []*cli.Command{
			{
				Name:  "db:setup",
				Usage: "Setup test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"j"},
						Usage:   "Number of parallel workers",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
					},
				},
				Action: func(c *cli.Context) error {
					return runDatabaseTask("db:setup", c)
				},
			},
			{
				Name:  "db:create",
				Usage: "Create test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"j"},
						Usage:   "Number of parallel workers",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
					},
				},
				Action: func(c *cli.Context) error {
					return runDatabaseTask("db:create", c)
				},
			},
			{
				Name:  "db:migrate",
				Usage: "Migrate test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"j"},
						Usage:   "Number of parallel workers",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
					},
				},
				Action: func(c *cli.Context) error {
					return runDatabaseTask("db:migrate", c)
				},
			},
			{
				Name:  "db:test:prepare",
				Usage: "Prepare test databases in parallel",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "workers",
						Aliases: []string{"j"},
						Usage:   "Number of parallel workers",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Show what would be executed without running",
					},
				},
				Action: func(c *cli.Context) error {
					return runDatabaseTask("db:test:prepare", c)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Print what would be executed without running",
			},
			&cli.BoolFlag{
				Name:  "auto",
				Usage: "Run bundle install if necessary before running tests",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Save detailed test results to JSON files",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"j"},
				Usage:   "Number of parallel workers (default: cores-2, env: PARALLEL_TEST_PROCESSORS)",
			},
		},
		Action: func(c *cli.Context) error {
			var specFiles []string
			var err error

			// Determine which spec files to run
			if c.NArg() > 0 {
				// Use provided arguments as spec files
				specFiles = c.Args().Slice()
			} else {
				// Auto-discover spec files
				specFiles, err = FindSpecFiles()
				if err != nil {
					return fmt.Errorf("error finding spec files: %v", err)
				}
				if len(specFiles) == 0 {
					return fmt.Errorf("no spec files found")
				}
			}

			dryRun := c.Bool("dry-run")

			if dryRun {
				if c.Bool("auto") {
					fmt.Fprintln(os.Stderr, "[dry-run] bundle install")
				}
				fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(specFiles))
				for _, file := range specFiles {
					args := []string{"bundle", "exec", "rspec", "--format", "progress", file}
					if c.Bool("json") {
						args = append(args, "--format", "json", "--out", "/tmp/results.json")
					}
					fmt.Fprintf(os.Stderr, "[dry-run] %s\n", strings.Join(args, " "))
				}
				return nil
			}

			// Run bundle install if --auto flag is set
			if c.Bool("auto") {
				fmt.Println("Installing dependencies...")
				bundleCmd := exec.Command("bundle", "install")
				bundleCmd.Stdout = os.Stdout
				bundleCmd.Stderr = os.Stderr

				if err := bundleCmd.Run(); err != nil {
					return fmt.Errorf("error running bundle install: %v", err)
				}
			}

			workerCount := GetWorkerCount(c.Int("workers"))
			actualWorkers := workerCount
			if len(specFiles) < workerCount {
				actualWorkers = len(specFiles)
			}

			fmt.Printf("Running %d spec files in parallel using %d workers (%d cores available)...\n",
				len(specFiles), actualWorkers, runtime.NumCPU())

			saveJSON := c.Bool("json")
			results, wallTime := RunTestsInParallel(specFiles, dryRun, saveJSON, workerCount)
			hasFailures := PrintResults(results, wallTime)

			// Exit with error if any tests failed
			if hasFailures {
				os.Exit(1)
			}

			return nil
		},
	}
}

func runDatabaseTask(task string, c *cli.Context) error {
	workerCount := GetWorkerCount(c.Int("workers"))
	dryRun := c.Bool("dry-run")
	
	return RunDatabaseTask(task, workerCount, dryRun)
}

func main() {
	app := createApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
