package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

// WatchFindCmd implements the 'plur watch find' command
// Shows what would be executed for a given file change
type WatchFindCmd struct {
	FilePath string `arg:"" help:"File path to check for watch mappings" required:"true"`
	Format   string `help:"Output format: text or json" default:"text" enum:"text,json"`
}

func (cmd *WatchFindCmd) BeforeApply(ctx *kong.Context) error {
	if err := rejectWatchFindNoOpFlags(ctx); err != nil {
		return err
	}
	return nil
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	if cmd.Format != "text" && cmd.Format != "json" {
		return fmt.Errorf("--format must be text or json")
	}

	jobs := globals.runtimeConfig.Jobs
	watches := globals.runtimeConfig.Watches

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

	if len(watches) == 0 {
		if globals.runtimeConfig.Use != "" {
			selected, err := runtime.SelectJobFromRuntimeConfig(globals.runtimeConfig, nil)
			if err != nil {
				return fmt.Errorf("failed to select watch job: %w", err)
			}
			runtime.LogInheritedFields(selected.Name, selected.Inherited)
		}

		if cmd.Format == "json" {
			if err := writeWatchFindPlan(filePath, emptyWatchFindResult(filePath, jobs), 2); err != nil {
				return err
			}
			return ExitCode{Code: 2}
		}

		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return ExitCode{Code: 2}
	}

	selected, err := runtime.SelectJobFromRuntimeConfig(globals.runtimeConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to select watch job: %w", err)
	}
	runtime.LogInheritedFields(selected.Name, selected.Inherited)

	if cmd.Format == "text" {
		fmt.Printf("[watch] Checking %s\n", filePath)
	}

	// Use shared find logic
	findResult, err := watch.FindTargetsForFile(filePath, jobs, watches, cwd)
	if err != nil {
		return fmt.Errorf("error processing file: %w", err)
	}

	exitCode := watchFindExitCode(findResult)
	if cmd.Format == "json" {
		if err := writeWatchFindPlan(filePath, findResult, exitCode); err != nil {
			return err
		}
		if exitCode != 0 {
			return ExitCode{Code: exitCode}
		}
		return nil
	}

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

func rejectWatchFindNoOpFlags(ctx *kong.Context) error {
	if ctx == nil {
		return nil
	}

	for _, path := range ctx.Path {
		if path.Flag == nil || path.Resolved {
			continue
		}
		flag := watchFindNoOpFlagName(path.Flag)
		if flag == "" {
			continue
		}
		return fmt.Errorf("%s does not apply to plur watch find; %s", flag, watchFindNoOpFlagGuidance(flag))
	}

	return nil
}

func watchFindNoOpFlagName(flag *kong.Flag) string {
	switch flag.Name {
	case "first-is-1":
		if flag.Negated {
			return "--no-first-is-1"
		}
		return "--first-is-1"
	case "dry-run", "dry-run-format", "ignore", "rspec-split", "workers":
		return "--" + flag.Name
	default:
		return ""
	}
}

func watchFindNoOpFlagGuidance(flag string) string {
	switch flag {
	case "--dry-run", "--dry-run-format":
		return "use `plur watch find --format=json <file>` for a structured watch preview, or `plur --dry-run [patterns...]` for a one-shot test plan"
	case "--ignore":
		return "`--ignore` filters live watch events, not watch find previews"
	default:
		return "watch find previews mappings and does not run test workers"
	}
}

type WatchFindPlan struct {
	Version         int                 `json:"version"`
	Mode            string              `json:"mode"`
	File            string              `json:"file"`
	MatchedRules    []WatchFindPlanRule `json:"matched_rules"`
	ExistingTargets map[string][]string `json:"existing_targets"`
	MissingTargets  map[string][]string `json:"missing_targets"`
	ExitCode        int                 `json:"exit_code"`
}

type WatchFindPlanRule struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Jobs   []string `json:"jobs"`
	Target string   `json:"target"`
}

func watchFindExitCode(result *watch.FindResult) int {
	if result.HasExistingTargets() {
		return 0
	}
	return 2
}

func emptyWatchFindResult(filePath string, jobs map[string]job.Job) *watch.FindResult {
	return &watch.FindResult{
		FilePath:        filePath,
		MatchedRules:    []watch.WatchMapping{},
		ExistingTargets: map[string][]string{},
		MissingTargets:  map[string][]string{},
		Jobs:            jobs,
	}
}

func writeWatchFindPlan(filePath string, result *watch.FindResult, exitCode int) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(buildWatchFindPlan(filePath, result, exitCode))
}

func buildWatchFindPlan(filePath string, result *watch.FindResult, exitCode int) WatchFindPlan {
	return WatchFindPlan{
		Version:         1,
		Mode:            "watch_find",
		File:            filePath,
		MatchedRules:    watchFindPlanRules(result.MatchedRules),
		ExistingTargets: cloneTargetMap(result.ExistingTargets),
		MissingTargets:  cloneTargetMap(result.MissingTargets),
		ExitCode:        exitCode,
	}
}

func watchFindPlanRules(rules []watch.WatchMapping) []WatchFindPlanRule {
	planRules := make([]WatchFindPlanRule, 0, len(rules))
	for _, rule := range rules {
		planRules = append(planRules, WatchFindPlanRule{
			Name:   rule.Name,
			Source: rule.Source,
			Jobs:   append([]string{}, rule.Jobs...),
			Target: formatWatchFindTargets(rule.Targets),
		})
	}
	return planRules
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

func cloneTargetMap(values map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(values))
	for _, key := range sortedMapKeys(values) {
		targets := append([]string{}, values[key]...)
		slices.Sort(targets)
		cloned[key] = targets
	}
	return cloned
}
