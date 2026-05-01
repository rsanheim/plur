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

// railsCommandJobName returns "rake" when the user invoked the rails command
// via its "rake" alias, otherwise "rails".
//
// Kong destructively rewrites alias tokens to the canonical command name
// during parsing (see github.com/alecthomas/kong/context.go, the block that
// assigns token values to a branch name when tagged as an alias). By the
// time Run() executes, ctx.Command() and ctx.Path[*].Command.Name both
// return "rails" regardless of which alias was typed, so we cannot use them
// to distinguish.
//
// Instead we inspect ctx.Args (the original argv passed to Kong) at the
// position where the command token was matched:
//
//	commandPos = len(ctx.Args) - len(path.Remainder()) - 1
//
// where path.Remainder() is the slice of unparsed args appearing after this
// Path element. This correctly distinguishes the command token from a flag
// value that happens to be "rake" (e.g. `plur -C rake rails db:prepare`).
func railsCommandJobName(ctx *kong.Context) string {
	if ctx == nil {
		return "rails"
	}

	for _, path := range ctx.Path {
		if path.Command == nil || path.Command.Name != "rails" {
			continue
		}

		commandPos := len(ctx.Args) - len(path.Remainder()) - 1
		if commandPos >= 0 && commandPos < len(ctx.Args) && ctx.Args[commandPos] == "rake" {
			return "rake"
		}
		return "rails"
	}

	return "rails"
}
