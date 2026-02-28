package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

// WatchFindCmd implements the 'plur watch find' command
// Shows what would be executed for a given file change
type WatchFindCmd struct {
	FilePath string `arg:"" help:"File path to check for watch mappings" required:"true"`
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	// Load watch configuration using the same logic as watch mode
	resolved, watches, err := loadWatchConfiguration(globals, "")
	if err != nil {
		return fmt.Errorf("failed to load watch configuration: %w", err)
	}

	// Build jobs map for FindTargetsForFile - include all user-defined jobs
	jobs := resolved.ResolvedJobs

	if len(watches) == 0 {
		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return nil
	}

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

	out := logger.StdoutLogger

	out.Info("checking watch", "file", filePath)

	// Use shared find logic
	findResult, err := watch.FindTargetsForFile(filePath, jobs, watches, cwd)
	if err != nil {
		return fmt.Errorf("error processing file: %w", err)
	}

	// Show matched rules
	if len(findResult.MatchedRules) == 0 {
		out.Info("found rules", "count", 0)
		return ExitCode{Code: 2}
	}

	// Print matched rules in concise format
	for _, rule := range findResult.MatchedRules {
		name := rule.Name
		if name == "" {
			name = "(unnamed)"
		}
		targetTemplate := "[source file]"
		if len(rule.Targets) > 0 {
			targetTemplate = rule.Targets[0]
		}
		out.Info("found rules",
			"name", name,
			"source", rule.Source,
			"jobs", rule.Jobs,
			"target", targetTemplate)
	}

	// Show found files
	if findResult.HasExistingTargets() {
		var allFiles []string
		for _, targets := range findResult.ExistingTargets {
			allFiles = append(allFiles, targets...)
		}
		out.Info("found files", "files", strings.Join(allFiles, ", "))
	}

	// Show missing files
	if findResult.HasMissingTargets() {
		for _, targets := range findResult.MissingTargets {
			for _, target := range targets {
				out.Warn("not found", "file", target)
			}
		}
	}

	// Exit code 2 if nothing would actually run
	if !findResult.HasExistingTargets() {
		return ExitCode{Code: 2}
	}

	return nil
}
