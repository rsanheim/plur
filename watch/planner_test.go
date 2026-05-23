package watch

import (
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanner_PlanPath(t *testing.T) {
	tmpDir := makeWatchTestProject(t)
	writeWatchTestFile(t, tmpDir, "lib/user.rb")
	writeWatchTestFile(t, tmpDir, "spec/user_spec.rb")
	writeWatchTestFile(t, tmpDir, "config/settings.yml")

	jobs := map[string]job.Job{
		"rspec": {Name: "rspec", Cmd: []string{"rspec", "{{target}}"}},
	}

	t.Run("runnable target", func(t *testing.T) {
		planner := Planner{
			Jobs: jobs,
			Watches: []WatchMapping{
				{
					Name:    "lib-to-spec",
					Source:  "lib/**/*.rb",
					Targets: []string{"spec/{{match}}_spec.rb"},
					Jobs:    []string{"rspec"},
				},
			},
			CWD: tmpDir,
		}

		plan := planner.PlanPath("lib/user.rb")

		require.Len(t, plan.JobPlans, 1)
		assert.Equal(t, "rspec", plan.JobPlans[0].JobName)
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, plan.JobPlans[0].Targets)
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, plan.ExistingTargets["rspec"])
		assert.Empty(t, plan.MissingTargets)
		assert.Empty(t, plan.NoRunnableChanges)
		assert.False(t, plan.ShouldReload)
	})

	t.Run("missing target", func(t *testing.T) {
		planner := Planner{
			Jobs: jobs,
			Watches: []WatchMapping{
				{
					Name:    "lib-to-spec",
					Source:  "lib/**/*.rb",
					Targets: []string{"spec/{{match}}_spec.rb"},
					Jobs:    []string{"rspec"},
				},
			},
			CWD: tmpDir,
		}

		plan := planner.PlanPath("lib/missing.rb")

		assert.Empty(t, plan.JobPlans)
		assert.Equal(t, []string{filepath.FromSlash("spec/missing_spec.rb")}, plan.MissingTargets["rspec"])
		assert.Equal(t, []NoRunnableChange{
			{
				Path:           "lib/missing.rb",
				Reason:         NoRunnableMissingTargets,
				MissingTargets: []string{filepath.FromSlash("spec/missing_spec.rb")},
			},
		}, plan.NoRunnableChanges)
		assert.False(t, plan.ShouldReload)
	})

	t.Run("reload only", func(t *testing.T) {
		planner := Planner{
			Jobs: map[string]job.Job{},
			Watches: []WatchMapping{
				{
					Name:   "config-reload",
					Source: "config/**/*.yml",
					Reload: true,
				},
			},
			CWD: tmpDir,
		}

		plan := planner.PlanPath("config/settings.yml")

		assert.True(t, plan.ShouldReload)
		assert.Empty(t, plan.JobPlans)
		assert.Empty(t, plan.NoRunnableChanges)
	})
}

func TestPlanner_PlanBatch(t *testing.T) {
	tmpDir := makeWatchTestProject(t)
	writeWatchTestFile(t, tmpDir, "lib/user.rb")
	writeWatchTestFile(t, tmpDir, "lib/post.rb")
	writeWatchTestFile(t, tmpDir, "spec/user_spec.rb")
	writeWatchTestFile(t, tmpDir, "spec/post_spec.rb")
	writeWatchTestFile(t, tmpDir, "spec/spec_helper.rb")

	jobs := map[string]job.Job{
		"rspec": {Name: "rspec", Cmd: []string{"rspec", "{{target}}"}},
	}
	planner := Planner{
		Jobs: jobs,
		Watches: []WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
		CWD: tmpDir,
	}

	plan := planner.PlanBatch([]string{"lib/user.rb", "spec/spec_helper.rb", "lib/post.rb"})

	require.Len(t, plan.JobPlans, 1)
	assert.Equal(t, "rspec", plan.JobPlans[0].JobName)
	assert.Equal(t, []string{
		filepath.FromSlash("spec/user_spec.rb"),
		filepath.FromSlash("spec/post_spec.rb"),
	}, plan.JobPlans[0].Targets)
	assert.Equal(t, []NoRunnableChange{
		{Path: "spec/spec_helper.rb", Reason: NoRunnableNoRule},
	}, plan.NoRunnableChanges)
}
