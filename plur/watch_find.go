package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

// WatchFindCmd implements the 'plur watch find' command
// Shows what would be executed for a given file change
type WatchFindCmd struct {
	FilePath string `arg:"" help:"File path to check for watch mappings" required:"true"`
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	// Get current task for context
	currentTask := task.DetectFramework()

	// Load watch configuration using the same logic as watch mode
	jobs, watches, err := loadWatchConfiguration(globals, currentTask)
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	if len(watches) == 0 {
		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return nil
	}

	// Create event processor
	processor := watch.NewEventProcessor(jobs, watches)

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Normalize the file path to be relative to cwd
	filePath := cmd.FilePath
	if filepath.IsAbs(filePath) {
		if rel, err := filepath.Rel(cwd, filePath); err == nil {
			filePath = rel
		}
	}

	// Process the file path through EventProcessor
	fmt.Printf("Checking file: %s\n\n", filePath)

	jobTargets, err := processor.ProcessPath(filePath)
	if err != nil {
		return fmt.Errorf("error processing file: %w", err)
	}

	if len(jobTargets) == 0 {
		fmt.Println("No matching watch rules for this file.")
		fmt.Println("\nConfigured watch mappings:")
		for i, w := range watches {
			fmt.Printf("  %d. %s\n", i+1, w.Source)
			if w.Name != "" {
				fmt.Printf("     Name: %s\n", w.Name)
			}
			if w.Targets != nil {
				fmt.Printf("     Targets: %v\n", *w.Targets)
			} else {
				fmt.Printf("     Targets: [source file]\n")
			}
			fmt.Printf("     Jobs: %v\n", w.Jobs.Slice())
			if len(w.Exclude) > 0 {
				fmt.Printf("     Exclude: %v\n", w.Exclude)
			}
			fmt.Println()
		}
		return nil
	}

	// Display what would be executed
	fmt.Println("Matched watch rules:")
	for jobName, targets := range jobTargets {
		job, exists := jobs[jobName]
		if !exists {
			logger.Logger.Warn("Job not found", "job", jobName)
			continue
		}

		fmt.Printf("\nJob: %s\n", jobName)
		fmt.Printf("  Command template: %s\n", strings.Join(job.Cmd, " "))
		if len(job.Env) > 0 {
			fmt.Printf("  Environment: %s\n", strings.Join(job.Env, ", "))
		}
		fmt.Printf("  Target files (%d):\n", len(targets))
		for _, target := range targets {
			// Build actual command that would be executed
			cmd := watch.BuildJobCmd(job, target)
			fmt.Printf("    - %s\n", target)
			fmt.Printf("      Command: %s\n", strings.Join(cmd, " "))
		}
	}

	return nil
}
