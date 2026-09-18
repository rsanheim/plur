package main

import (
	"fmt"

	"github.com/rsanheim/plur/internal/runner"
)

type RailsCmd struct {
	Args []string `arg:"" optional:"" name:"args" help:"Rails command arguments to run once per worker"`
}

func (r *RailsCmd) Help() string {
	return `Runs the configured rails job once per worker, appending the
given arguments literally. Each worker gets PARALLEL_TEST_GROUPS and
TEST_ENV_NUMBER in its environment.

Put plur flags before the command args. Use -- to pass flags through
to rails unchanged.

Examples:

	plur rails db:test:prepare
	plur rails db:test:prepare -n 4
	plur rails db:migrate -n 4 -- --trace`
}

func (r *RailsCmd) Run(parent *PlurCLI) error {
	return runPerWorkerJob(parent, "rails", r.Args)
}

type RakeCmd struct {
	Args []string `arg:"" optional:"" name:"args" help:"Rake task arguments to run once per worker"`
}

func (r *RakeCmd) Help() string {
	return `Runs the configured rake job once per worker, appending the
given arguments literally. Each worker gets PARALLEL_TEST_GROUPS and
TEST_ENV_NUMBER in its environment.

Put plur flags before the command args. Use -- to pass flags through
to rake unchanged.

Examples:

	plur rake db:setup -n 4
	plur rake -n 1 -- --tasks`
}

func (r *RakeCmd) Run(parent *PlurCLI) error {
	return runPerWorkerJob(parent, "rake", r.Args)
}

func runPerWorkerJob(parent *PlurCLI, jobName string, args []string) error {
	j, ok := parent.runtimeConfig.Jobs[jobName]
	if !ok {
		return fmt.Errorf("job %q not found", jobName)
	}

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, parent.passthroughArgs...)

	run, err := runner.NewRunner(parent.globalConfig, nil, j, nil)
	if err != nil {
		return err
	}
	return run.RunArgsPerWorker(allArgs)
}
