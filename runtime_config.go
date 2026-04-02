package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

type RuntimeConfig struct {
	Use       string
	Jobs      map[string]job.Job
	Watches   []watch.WatchMapping
	Inherited map[string]autodetect.InheritedFields
	Sources   []string
}

func buildRuntimeConfig(cli *PlurCLI) (*RuntimeConfig, error) {
	jobs, inherited, err := autodetect.BuildResolvedJobs(cli.Job)
	if err != nil {
		return nil, err
	}

	watches := cli.WatchMappings
	if len(watches) == 0 {
		result, err := autodetect.ResolveJob(cli.Use, cli.Job, nil)
		if err != nil {
			return nil, err
		}
		watches = result.Watches
	}

	rc := &RuntimeConfig{
		Use:       cli.Use,
		Jobs:      jobs,
		Watches:   watches,
		Inherited: inherited,
		Sources:   runtimeConfigSources(cli.configFiles),
	}

	if err := validateRuntimeConfig(rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func validateRuntimeConfig(rc *RuntimeConfig) error {
	for name, j := range rc.Jobs {
		if len(j.Cmd) == 0 {
			return fmt.Errorf("configuration error in %v: job %q must define a command", rc.Sources, name)
		}
	}

	for _, w := range rc.Watches {
		for _, jobName := range w.Jobs {
			if _, ok := rc.Jobs[jobName]; !ok {
				return fmt.Errorf("configuration error in %v: watch %q references undefined job %q", rc.Sources, w.Name, jobName)
			}
		}
		for _, target := range w.Targets {
			if err := watch.ValidateTemplate(target); err != nil {
				return fmt.Errorf("configuration error in %v: watch %q has invalid target template %q: %w", rc.Sources, w.Name, target, err)
			}
		}
	}

	return nil
}

func runtimeConfigSources(configFiles []string) []string {
	var out []string
	for _, configFile := range configFiles {
		expanded := kong.ExpandPath(configFile)
		if _, err := os.Stat(expanded); err == nil {
			out = append(out, expanded)
		}
	}
	return out
}
