package main

import (
	"fmt"
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
	planner, _, err := buildWatchPlanner(globals, parent)
	if err != nil {
		return err
	}

	if len(planner.Watches) == 0 {
		fmt.Println("No watch mappings configured.")
		fmt.Println("Either add job/watch configuration to .plur.toml or ensure your project structure")
		fmt.Println("matches a supported framework (Ruby with Gemfile, Go with go.mod).")
		return nil
	}

	out := logger.StdoutLogger

	path, admitted := planner.Admit(cmd.FilePath)
	out.Info("checking watch", "file", path)
	if !admitted {
		out.Info("ignored", "file", path)
		return ExitCode{Code: 2}
	}

	plan := planner.Plan([]string{path})

	if len(plan.Matches) == 0 {
		out.Info("found rules", "count", 0)
		return ExitCode{Code: 2}
	}

	for _, m := range plan.Matches {
		name := m.Rule.Name
		if name == "" {
			name = "(unnamed)"
		}
		targetTemplate := "[source file]"
		if m.Rule.NoTargets {
			targetTemplate = "[no targets]"
		} else if len(m.Rule.Targets) > 0 {
			targetTemplate = m.Rule.Targets[0]
		}
		out.Info("found rules",
			"name", name,
			"source", m.Rule.Source,
			"jobs", m.Rule.Jobs,
			"target", targetTemplate)
	}

	var allFiles []string
	for _, run := range plan.Runs {
		allFiles = append(allFiles, run.Targets...)
	}
	if len(allFiles) > 0 {
		out.Info("found files", "files", strings.Join(allFiles, ", "))
	}

	for _, run := range plan.Runs {
		out.Info("would run",
			"job", run.Job.Name,
			"cmd", watch.CommandString(run.Command(planner.CWD), run.Job.Env))
	}

	for _, m := range plan.Matches {
		for _, target := range m.Missing {
			out.Warn("not found", "file", target)
		}
	}

	if len(plan.Runs) == 0 {
		return ExitCode{Code: 2}
	}

	return nil
}
