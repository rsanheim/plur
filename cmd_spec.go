package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

func (r *SpecCmd) Run(parent *PlurCLI) error {
	cfg := parent.globalConfig
	fmt.Fprintf(os.Stderr, "plur version=%s\n", buildinfo.GetVersionInfo())
	logger.Logger.Debug("running plur", "command", "spec", "args", os.Args[1:])

	selected, err := runtime.SelectJobFromRuntimeConfig(parent.runtimeConfig, r.Patterns)
	if err != nil {
		return err
	}

	currentJob := selected.Job

	runtime.LogInheritedFields(currentJob.Name, selected.Inherited)

	if len(r.Tags) > 0 && currentJob.Framework != "rspec" {
		return fmt.Errorf("--tag is only supported for rspec (current framework: %s)", currentJob.Framework)
	}

	targetPatterns, _ := framework.TargetPatternsForJob(currentJob)
	logger.Logger.Debug("SpecCmd.Run", "job", currentJob.Name, "framework", currentJob.Framework, "patterns", r.Patterns, "target_patterns", targetPatterns, "reason", selected.Reason)

	// Discover test files
	var testFiles []string
	if len(r.Patterns) > 0 {
		testFiles, err = ExpandPatternsFromJob(r.Patterns, currentJob)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			return fmt.Errorf("no test files found matching provided patterns")
		}
	} else {
		testFiles, err = FindFilesFromJob(currentJob)
		if err != nil {
			return err
		}
		if len(testFiles) == 0 {
			patterns, err := framework.TargetPatternsForJob(currentJob)
			if err != nil || len(patterns) == 0 {
				return fmt.Errorf("no test files found")
			}
			return fmt.Errorf("no test files found (looking for %s)", strings.Join(patterns, ", "))
		}
	}
	msg := fmt.Sprintf("found %v test files", len(testFiles))
	logger.Logger.Debug(msg, "testFiles", testFiles)

	if r.Auto {
		depManager := NewDependencyManager(cfg.DryRun)
		if err := depManager.InstallDependencies(); err != nil {
			return err
		}
	}

	cfg.Auto = r.Auto
	cfg.RspecTrace = r.RspecTrace

	extraArgs := buildTagArgs(r.Tags)
	extraArgs = append(extraArgs, parent.passthroughArgs...)

	runner, err := NewRunner(cfg, testFiles, currentJob, extraArgs)
	if err != nil {
		return err
	}
	results, wallTime, err := runner.Run()
	if err != nil {
		return err
	}

	if cfg.DryRun {
		return nil
	}

	// Save runtime data if tests actually ran
	hasValidRuntimeData := false
	for _, result := range results {
		if result.State != types.StateError && result.ExampleCount > 0 {
			hasValidRuntimeData = true
			break
		}
	}

	if hasValidRuntimeData {
		if err := runner.Tracker().SaveToFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save runtime data: %v\n", err)
		} else {
			logger.Logger.Debug("Runtime data saved", "runtime_path", runner.Tracker().RuntimeFilePath())
		}
	}

	summary := BuildTestSummary(results, wallTime, currentJob)
	PrintResults(summary, cfg.ColorOutput, currentJob)

	if !summary.Success {
		os.Exit(1)
	}

	return nil
}

func buildTagArgs(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	args := make([]string, 0, len(tags)*2)
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}
	return args
}
