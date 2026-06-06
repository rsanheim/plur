package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/fileset"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/internal/testruntime"
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

	if err := rejectRunModeTargetTemplate(currentJob.Name, currentJob.UsesTargets(), selected.Inherited.Cmd); err != nil {
		return err
	}

	if len(r.Tags) > 0 && currentJob.Framework != "rspec" {
		return fmt.Errorf("--tag is only supported for rspec (current framework: %s)", currentJob.Framework)
	}

	targetPatterns, _ := framework.TargetPatternsForJob(currentJob)
	logger.Logger.Debug("SpecCmd.Run", "job", currentJob.Name, "framework", currentJob.Framework, "patterns", r.Patterns, "target_patterns", targetPatterns, "reason", selected.Reason)

	excludes := slices.Concat(currentJob.ExcludePatterns, r.ExcludePatterns)
	discovery, err := fileset.Discover(currentJob, r.Patterns, excludes)
	if err != nil {
		return err
	}
	testFiles := discovery.Files
	if len(testFiles) == 0 {
		switch {
		case len(excludes) > 0:
			return fmt.Errorf("no test files remain after applying exclude patterns")
		case len(r.Patterns) > 0:
			return fmt.Errorf("no test files found matching provided patterns")
		case len(targetPatterns) > 0:
			return fmt.Errorf("no test files found (looking for %s)", strings.Join(targetPatterns, ", "))
		}
		return fmt.Errorf("no test files found")
	}
	logger.Logger.Debug("discovered test files", "count", len(testFiles), "exclude_patterns", excludes, "files", testFiles)

	warnings := unmatchedCLIExcludeWarnings(r.ExcludePatterns, discovery.ExcludeMatches)
	targetWarnings, err := explicitTargetMismatchWarnings(r.Patterns, targetPatterns, currentJob.Name)
	if err != nil {
		return err
	}
	warnings = append(warnings, targetWarnings...)
	printWarnings(warnings)

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
	aborted := false
	for _, result := range results {
		if result.State == types.StateError {
			aborted = true
		}
		if result.State != types.StateError && result.ExampleCount > 0 {
			hasValidRuntimeData = true
		}
	}

	if hasValidRuntimeData {
		runKind := testruntime.ClassifyRunKind(r.Patterns, r.Tags, parent.passthroughArgs, aborted)
		if err := runner.Tracker().SaveToFile(runKind); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save runtime data: %v\n", err)
		} else {
			logger.Logger.Debug("Runtime data saved", "runtime_path", runner.Tracker().RuntimeFilePath(), "run_kind", runKind)
		}
	}

	summary := BuildTestSummary(results, wallTime, currentJob)
	PrintResults(summary, cfg.ColorOutput, currentJob)

	if !summary.Success {
		os.Exit(1)
	}

	return nil
}

func rejectRunModeTargetTemplate(jobName string, usesTargets, inheritedCmd bool) error {
	if !usesTargets || inheritedCmd {
		return nil
	}
	return fmt.Errorf("job %q command uses {{target}}, but run mode appends targets automatically; remove {{target}} from job cmd", jobName)
}

func unmatchedCLIExcludeWarnings(patterns []string, matches map[string]int) []string {
	var warnings []string
	for _, pattern := range patterns {
		if matches[pattern] == 0 {
			warnings = append(warnings, fmt.Sprintf("--exclude-pattern %s matched no selected files", shellSingleQuote(pattern)))
		}
	}
	return warnings
}

func explicitTargetMismatchWarnings(patterns, targetPatterns []string, jobName string) ([]string, error) {
	mismatches, err := fileset.ExplicitTargetMismatches(patterns, targetPatterns)
	if err != nil {
		return nil, err
	}
	var warnings []string
	for _, mismatch := range mismatches {
		warnings = append(warnings, fmt.Sprintf("target %s does not match selected job %s target pattern %s",
			shellSingleQuote(mismatch),
			shellSingleQuote(jobName),
			shellSingleQuote(strings.Join(targetPatterns, ", "))))
	}
	return warnings, nil
}

func printWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "[warn] %s\n", warning)
	}
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
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
