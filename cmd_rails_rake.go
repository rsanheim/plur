package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

type RailsCmd struct {
	Args []string `arg:"" optional:"" name:"args" help:"Rails or Rake arguments to run once per worker"`
}

func (r *RailsCmd) Help() string {
	return `Runs the configured rails or rake job once per worker, appending the
given arguments literally. Each worker gets PARALLEL_TEST_GROUPS and
TEST_ENV_NUMBER in its environment.

Put plur flags before the command args. Use -- to pass flags through
to rails/rake unchanged.

Examples:

	plur rails db:prepare -n 4
	plur rails db:migrate VERSION=20260429000000 -n 4
	plur rails db:migrate -n 4 -- --trace
	plur rake db:setup -n 4
	plur rake db:create db:migrate -n 4
	plur rake -n 1 -- --tasks`
}

func (r *RailsCmd) Run(parent *PlurCLI, ctx *kong.Context) error {
	jobName := railsCommandJobName(ctx)
	j, ok := parent.runtimeConfig.Jobs[jobName]
	if !ok {
		return fmt.Errorf("job %q not found", jobName)
	}

	args := append([]string{}, r.Args...)
	args = append(args, parent.passthroughArgs...)

	runner, err := NewRunner(parent.globalConfig, nil, j, nil)
	if err != nil {
		return err
	}
	return runner.RunArgsPerWorker(args)
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
