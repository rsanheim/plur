package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

type RailsCmd struct {
	Args []string `arg:"" optional:"" name:"args" help:"Rails or Rake arguments to run once per worker"`
}

func (r *RailsCmd) Run(parent *PlurCLI, ctx *kong.Context) error {
	jobName := railsCommandJobName(ctx)
	j, ok := parent.runtimeConfig.Jobs[jobName]
	if !ok {
		return fmt.Errorf("job %q not found", jobName)
	}

	runner, err := NewRunner(parent.globalConfig, nil, j, nil)
	if err != nil {
		return err
	}
	return runner.RunArgsPerWorker(r.Args)
}

func railsCommandJobName(ctx *kong.Context) string {
	if ctx == nil {
		return "rails"
	}

	for _, path := range ctx.Path {
		if path.Command == nil || path.Command.Name != "rails" {
			continue
		}

		commandIndex := len(ctx.Args) - len(path.Remainder()) - 1
		if commandIndex >= 0 && commandIndex < len(ctx.Args) && ctx.Args[commandIndex] == "rake" {
			return "rake"
		}
		return "rails"
	}

	return "rails"
}
