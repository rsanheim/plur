package main

import (
	"errors"
	"testing"

	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
)

func TestWatchFindExitCodeReportsPlanningErrors(t *testing.T) {
	plan := watch.Plan{
		Errors: []watch.PlanError{{Path: "lib/user.rb", Err: errors.New("bad watch mapping")}},
	}

	assert.Equal(t, 1, watchFindExitCode(plan))
}

func TestBuildWatchFindPlanIncludesPlanningErrors(t *testing.T) {
	plan := watch.Plan{
		ExistingTargets: map[string][]string{},
		MissingTargets:  map[string][]string{},
		Errors:          []watch.PlanError{{Path: "lib/user.rb", Err: errors.New("bad watch mapping")}},
	}

	out := buildWatchFindPlan("lib/user.rb", plan, "/project", 1)

	assert.Equal(t, 1, out.ExitCode)
	assert.Equal(t, []WatchFindPlanError{
		{Path: "lib/user.rb", Error: "bad watch mapping"},
	}, out.Errors)
}
