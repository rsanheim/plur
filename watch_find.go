package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/watch"
)

// WatchFindCmd implements the 'plur watch find' command
// Shows what would be executed for a given file change
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

	fmt.Printf("[watch] Checking %s\n", filePath)

	// Use shared find logic
	findResult, err := watch.FindTargetsForFile(filePath, jobs, watches, cwd)
	if err != nil {
		return fmt.Errorf("error processing file: %w", err)
	}

	// Show matched rules
	if len(findResult.MatchedRules) == 0 {
		fmt.Printf("[watch] No matching rule for %s\n", filePath)
		return ExitCode{Code: 2}
	}

	printWatchFindRules(findResult.MatchedRules)
	printWatchFindExistingTargets(findResult.ExistingTargets)
	printWatchFindMissingTargets(filePath, findResult)

	// Exit code 2 if nothing would actually run
	if !findResult.HasExistingTargets() {
		if !findResult.HasMissingTargets() {
			fmt.Printf("[watch] No runnable targets for %s\n", filePath)
		}
		return ExitCode{Code: 2}
	}

	return nil
}

func printWatchFindRules(rules []watch.WatchMapping) {
	for _, rule := range rules {
		name := rule.Name
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Printf("[watch] Matched rule %s (source: %s, jobs: %s, target: %s)\n",
			name, rule.Source, formatWatchFindList(rule.Jobs), formatWatchFindTargets(rule.Targets))
	}
}

func printWatchFindExistingTargets(existingTargets map[string][]string) {
	for _, jobName := range sortedMapKeys(existingTargets) {
		targets := slices.Clone(existingTargets[jobName])
		slices.Sort(targets)
		fmt.Printf("[watch] Would run job %s with %s\n", jobName, strings.Join(targets, ", "))
	}
}

func printWatchFindMissingTargets(filePath string, result *watch.FindResult) {
	if !result.HasMissingTargets() {
		return
	}

	missing := flattenTargetMap(result.MissingTargets)
	label := "No existing targets"
	if result.HasExistingTargets() {
		label = "Missing targets"
	}
	fmt.Printf("[watch] %s for %s (missing: %s)\n", label, filePath, strings.Join(missing, ", "))
}

func formatWatchFindTargets(targets []string) string {
	if len(targets) == 0 {
		return "[source file]"
	}
	return formatWatchFindList(targets)
}

func formatWatchFindList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func sortedMapKeys(values map[string][]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if len(values[key]) > 0 {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func flattenTargetMap(values map[string][]string) []string {
	var out []string
	for _, key := range sortedMapKeys(values) {
		out = append(out, values[key]...)
	}
	slices.Sort(out)
	return out
}
