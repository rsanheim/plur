package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

type WatchFindCmd struct {
	FilePath string `arg:"" help:"File path to check for watch mappings" required:"true"`
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	selected, err := runtime.SelectJobFromRuntimeConfig(globals.runtimeConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to select watch job: %w", err)
	}

	jobs := globals.runtimeConfig.Jobs
	watches := globals.runtimeConfig.Watches
	runtime.LogInheritedFields(selected.Name, selected.Inherited)

	if len(watches) == 0 {
		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	filePath := cmd.FilePath
	if filepath.IsAbs(filePath) {
		if rel, err := filepath.Rel(cwd, filePath); err == nil {
			filePath = rel
		}
	}

	out := logger.StdoutLogger

	out.Info("checking watch", "file", filePath)

	findResult, err := watch.FindTargetsForFile(filePath, jobs, watches, cwd)
	if err != nil {
		return fmt.Errorf("error processing file: %w", err)
	}

	if len(findResult.MatchedRules) == 0 {
		out.Info("found rules", "count", 0)
		return ExitCode{Code: 2}
	}

	for _, rule := range findResult.MatchedRules {
		name := rule.Name
		if name == "" {
			name = "(unnamed)"
		}
		targetTemplate := "[source file]"
		if rule.NoTargets {
			targetTemplate = "[no targets]"
		} else if len(rule.Targets) > 0 {
			targetTemplate = rule.Targets[0]
		}
		out.Info("found rules",
			"name", name,
			"source", rule.Source,
			"jobs", rule.Jobs,
			"target", targetTemplate)
	}

	if findResult.HasExistingTargets() {
		allFiles := findResult.ExistingTargetFiles()
		if len(allFiles) > 0 {
			out.Info("found files", "files", strings.Join(allFiles, ", "))
		}
	}

	if findResult.HasMissingTargets() {
		for _, targets := range findResult.MissingTargets {
			for _, target := range targets {
				out.Warn("not found", "file", target)
			}
		}
	}

	if !findResult.HasExistingTargets() {
		return ExitCode{Code: 2}
	}

	return nil
}
