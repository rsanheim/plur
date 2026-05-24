package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/internal/watchsession"
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

	watches := globals.runtimeConfig.Watches

	cwd, err := watchsession.CurrentWorkingDirectory()
	if err != nil {
		return err
	}
	filePath := watchsession.NormalizePath(cwd, cmd.FilePath)

	if len(watches) == 0 {
		if globals.runtimeConfig.Use != "" {
			session, err := watchsession.New(globals.runtimeConfig, watchsession.Options{})
			if err != nil {
				return err
			}
			runtime.LogInheritedFields(session.Selected.Name, session.Selected.Inherited)
		}

		if cmd.Format == "json" {
			if err := writeWatchFindPlan(filePath, emptyWatchFindPlan(filePath), cwd, 2); err != nil {
				return err
			}
			return ExitCode{Code: 2}
		}

		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return ExitCode{Code: 2}
	}

	session, err := watchsession.New(globals.runtimeConfig, watchsession.Options{
		IgnorePatterns: parent.Ignore,
	})
	if err != nil {
		return err
	}
	selected := session.Selected
	runtime.LogInheritedFields(selected.Name, selected.Inherited)
	admission := session.AdmitPathForPreview(cmd.FilePath)
	filePath = admission.Path
	if filePath == "" {
		filePath = session.NormalizePath(cmd.FilePath)
	}

	if cmd.Format == "text" {
		fmt.Printf("[watch] Checking %s\n", filePath)
	}

	if !admission.Admitted {
		if cmd.Format == "json" {
			if err := writeWatchFindPlanWithAdmission(filePath, emptyWatchFindPlan(filePath), session.CWD, 2, &admission); err != nil {
				return err
			}
			return ExitCode{Code: 2}
		}
		printWatchFindAdmissionRejection(admission)
		return ExitCode{Code: 2}
	}

	findPlan := session.PlanPath(filePath)

	exitCode := watchFindExitCode(findPlan)
	if cmd.Format == "json" {
		if err := writeWatchFindPlan(filePath, findPlan, session.CWD, exitCode); err != nil {
			return err
		}
		if exitCode != 0 {
			return ExitCode{Code: exitCode}
		}
		return nil
	}

	if len(findPlan.Errors) > 0 {
		printWatchFindErrors(findPlan.Errors)
		return ExitCode{Code: 1}
	}

	if len(findPlan.MatchedRules) == 0 {
		printWatchNoRule(filePath)
		return ExitCode{Code: 2}
	}

	printWatchFindRules(findPlan.MatchedRules)
	printWatchFindExistingTargets(findPlan.ExistingTargets)
	printWatchFindJobPlans(findPlan.JobPlans, session.CWD)
	printWatchFindMissingTargets(filePath, findPlan)

	// Exit code 2 if nothing would actually run
	if !hasWatchFindTargets(findPlan.ExistingTargets) {
		if !hasWatchFindTargets(findPlan.MissingTargets) {
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
	case "dry-run", "dry-run-format", "rspec-split", "workers":
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
	Version         int                  `json:"version"`
	Mode            string               `json:"mode"`
	File            string               `json:"file"`
	Admission       *WatchFindAdmission  `json:"admission,omitempty"`
	MatchedRules    []WatchFindPlanRule  `json:"matched_rules"`
	ExistingTargets map[string][]string  `json:"existing_targets"`
	MissingTargets  map[string][]string  `json:"missing_targets"`
	JobPlans        []WatchFindJobPlan   `json:"job_plans"`
	Errors          []WatchFindPlanError `json:"errors,omitempty"`
	ExitCode        int                  `json:"exit_code"`
}

type WatchFindJobPlan struct {
	Job     string   `json:"job"`
	Targets []string `json:"targets"`
	Argv    []string `json:"argv"`
	Env     []string `json:"env"`
	CWD     string   `json:"cwd"`
	Shell   string   `json:"shell"`
}

type WatchFindPlanRule struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Jobs   []string `json:"jobs"`
	Target string   `json:"target"`
}

type WatchFindAdmission struct {
	Path     string `json:"path"`
	Admitted bool   `json:"admitted"`
	Reason   string `json:"reason,omitempty"`
}

type WatchFindPlanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func watchFindExitCode(plan watch.Plan) int {
	if len(plan.Errors) > 0 {
		return 1
	}
	if hasWatchFindTargets(plan.ExistingTargets) {
		return 0
	}
	return 2
}

func emptyWatchFindPlan(filePath string) watch.Plan {
	return watch.Plan{
		Paths:           []string{filePath},
		MatchedRules:    []watch.WatchMapping{},
		ExistingTargets: map[string][]string{},
		MissingTargets:  map[string][]string{},
	}
}

func writeWatchFindPlan(filePath string, plan watch.Plan, cwd string, exitCode int) error {
	return writeWatchFindPlanWithAdmission(filePath, plan, cwd, exitCode, nil)
}

func writeWatchFindPlanWithAdmission(filePath string, plan watch.Plan, cwd string, exitCode int, admission *watch.AdmissionResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(buildWatchFindPlanWithAdmission(filePath, plan, cwd, exitCode, admission))
}

func buildWatchFindPlan(filePath string, plan watch.Plan, cwd string, exitCode int) WatchFindPlan {
	return buildWatchFindPlanWithAdmission(filePath, plan, cwd, exitCode, nil)
}

func buildWatchFindPlanWithAdmission(filePath string, plan watch.Plan, cwd string, exitCode int, admission *watch.AdmissionResult) WatchFindPlan {
	return WatchFindPlan{
		Version:         1,
		Mode:            "watch_find",
		File:            filePath,
		Admission:       watchFindAdmission(admission),
		MatchedRules:    watchFindPlanRules(plan.MatchedRules),
		ExistingTargets: cloneTargetMap(plan.ExistingTargets),
		MissingTargets:  cloneTargetMap(plan.MissingTargets),
		JobPlans:        watchFindJobPlans(watch.BuildExecutionPlans(plan.JobPlans, cwd)),
		Errors:          watchFindPlanErrors(plan.Errors),
		ExitCode:        exitCode,
	}
}

func watchFindAdmission(admission *watch.AdmissionResult) *WatchFindAdmission {
	if admission == nil {
		return nil
	}
	return &WatchFindAdmission{
		Path:     admission.Path,
		Admitted: admission.Admitted,
		Reason:   admission.Reason,
	}
}

func watchFindJobPlans(executionPlans []watch.ExecutionPlan) []WatchFindJobPlan {
	plans := make([]WatchFindJobPlan, 0, len(executionPlans))
	for _, executionPlan := range executionPlans {
		plans = append(plans, WatchFindJobPlan{
			Job:     executionPlan.JobName,
			Targets: slices.Clone(executionPlan.Targets),
			Argv:    slices.Clone(executionPlan.Argv),
			Env:     slices.Clone(executionPlan.Env),
			CWD:     executionPlan.CWD,
			Shell:   watchFindShell(executionPlan.Env, executionPlan.Argv),
		})
	}
	return plans
}

func watchFindPlanErrors(errors []watch.PlanError) []WatchFindPlanError {
	planErrors := make([]WatchFindPlanError, 0, len(errors))
	for _, err := range errors {
		if err.Err == nil {
			continue
		}
		planErrors = append(planErrors, WatchFindPlanError{
			Path:  err.Path,
			Error: err.Err.Error(),
		})
	}
	return planErrors
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

func printWatchFindJobPlans(jobPlans []watch.JobPlan, cwd string) {
	for _, executionPlan := range watch.BuildExecutionPlans(jobPlans, cwd) {
		fmt.Printf("[watch] Command: %s\n", watchFindShell(executionPlan.Env, executionPlan.Argv))
	}
}

func printWatchFindErrors(errors []watch.PlanError) {
	for _, err := range errors {
		if err.Err == nil {
			continue
		}
		fmt.Fprintf(os.Stderr, "[watch] Error planning %s: %v\n", err.Path, err.Err)
	}
}

func printWatchFindAdmissionRejection(admission watch.AdmissionResult) {
	path := admission.Path
	if path == "" {
		path = "(unknown path)"
	}
	switch admission.Reason {
	case "ignored":
		fmt.Printf("[watch] Ignored %s\n", path)
	default:
		fmt.Printf("[watch] Ignored %s (reason: %s)\n", path, admission.Reason)
	}
}

func printWatchFindMissingTargets(filePath string, plan watch.Plan) {
	if !hasWatchFindTargets(plan.MissingTargets) {
		return
	}

	missing := flattenTargetMap(plan.MissingTargets)
	label := "No existing targets"
	if hasWatchFindTargets(plan.ExistingTargets) {
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

func watchFindShell(env []string, argv []string) string {
	parts := make([]string, 0, len(env)+len(argv))
	parts = append(parts, env...)
	parts = append(parts, argv...)
	return strings.Join(shellQuoteArgs(parts), " ")
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

func hasWatchFindTargets(values map[string][]string) bool {
	for _, targets := range values {
		if len(targets) > 0 {
			return true
		}
	}
	return false
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
