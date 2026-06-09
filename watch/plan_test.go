package watch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFileTree(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte("# test"), 0o644))
	}
}

func rspecJobs() map[string]framework.Job {
	return map[string]framework.Job{
		"rspec": {Name: "rspec", Cmd: []string{"bundle", "exec", "rspec"}},
	}
}

func libToSpec() WatchMapping {
	return WatchMapping{
		Name:    "lib-to-spec",
		Source:  "lib/**/*.rb",
		Targets: []string{"spec/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}
}

func TestPlannerPlan_RendersTargetsAndBuildsRun(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb", "spec/models/post_spec.rb")

	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{libToSpec()}, CWD: tmpDir}

	t.Run("simple lib file", func(t *testing.T) {
		plan := planner.Plan([]string{"lib/user.rb"})

		require.Len(t, plan.Matches, 1)
		assert.Equal(t, "lib-to-spec", plan.Matches[0].Rule.Name)
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, plan.Matches[0].Existing)
		assert.Empty(t, plan.Matches[0].Missing)

		require.Len(t, plan.Runs, 1)
		assert.Equal(t, "rspec", plan.Runs[0].Job.Name)
		assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, plan.Runs[0].Targets)
		assert.False(t, plan.Reload)
	})

	t.Run("nested lib file", func(t *testing.T) {
		plan := planner.Plan([]string{"lib/models/post.rb"})

		require.Len(t, plan.Runs, 1)
		assert.Equal(t, []string{filepath.FromSlash("spec/models/post_spec.rb")}, plan.Runs[0].Targets)
	})

	t.Run("non-matching file", func(t *testing.T) {
		plan := planner.Plan([]string{"app/models/user.rb"})

		assert.Empty(t, plan.Matches)
		assert.Empty(t, plan.Runs)
	})
}

func TestPlannerPlan_SourceFileAsTargetWhenNoTargetsConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/models/user_spec.rb")

	planner := Planner{
		Jobs: rspecJobs(),
		Watches: []WatchMapping{
			{Name: "spec-files", Source: "spec/**/*_spec.rb", Jobs: []string{"rspec"}},
		},
		CWD: tmpDir,
	}

	plan := planner.Plan([]string{"spec/models/user_spec.rb"})

	require.Len(t, plan.Runs, 1)
	assert.Equal(t, []string{filepath.FromSlash("spec/models/user_spec.rb")}, plan.Runs[0].Targets)
}

func TestPlannerPlan_NoTargetsRuleRunsJobBare(t *testing.T) {
	planner := Planner{
		Jobs: map[string]framework.Job{
			"build": {Name: "build", Cmd: []string{"bin/rake", "install"}},
		},
		Watches: []WatchMapping{
			{Name: "go-build", Source: "**/*.go", NoTargets: true, Jobs: []string{"build"}},
		},
		CWD: t.TempDir(),
	}

	plan := planner.Plan([]string{"runner.go"})

	require.Len(t, plan.Runs, 1)
	assert.Equal(t, "build", plan.Runs[0].Job.Name)
	assert.Empty(t, plan.Runs[0].Targets)
}

func TestPlannerPlan_NoTargetsRunsEvenWhenOtherRuleTargetsMissing(t *testing.T) {
	planner := Planner{
		Jobs: map[string]framework.Job{
			"build": {Name: "build", Cmd: []string{"bin/rake", "install"}},
		},
		Watches: []WatchMapping{
			{Name: "go-build", Source: "**/*.go", NoTargets: true, Jobs: []string{"build"}},
			{Name: "generated-target", Source: "**/*.go", Targets: []string{"generated/missing.txt"}, Jobs: []string{"build"}},
		},
		CWD: t.TempDir(),
	}

	plan := planner.Plan([]string{"runner.go"})

	require.Len(t, plan.Runs, 1)
	assert.Empty(t, plan.Runs[0].Targets)
}

func TestPlannerPlan_MissingTargetRecordedButJobDoesNotRun(t *testing.T) {
	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{libToSpec()}, CWD: t.TempDir()}

	plan := planner.Plan([]string{"lib/user.rb"})

	require.Len(t, plan.Matches, 1)
	assert.Empty(t, plan.Matches[0].Existing)
	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, plan.Matches[0].Missing)
	assert.Empty(t, plan.Runs)
}

func TestPlannerPlan_RuleIgnorePatterns(t *testing.T) {
	rule := libToSpec()
	rule.Ignore = []string{"lib/generators/**", "lib/vendor/**"}
	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{rule}, CWD: t.TempDir()}

	assert.Empty(t, planner.Plan([]string{"lib/generators/model.rb"}).Matches)
	assert.Empty(t, planner.Plan([]string{"lib/vendor/gem.rb"}).Matches)
	assert.Len(t, planner.Plan([]string{"lib/user.rb"}).Matches, 1)
}

func TestPlannerPlan_MultipleJobsPreserveRuleOrder(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb")

	rule := libToSpec()
	rule.Jobs = []string{"rspec", "rubocop"}
	planner := Planner{
		Jobs: map[string]framework.Job{
			"rspec":   {Name: "rspec", Cmd: []string{"rspec"}},
			"rubocop": {Name: "rubocop", Cmd: []string{"rubocop"}},
		},
		Watches: []WatchMapping{rule},
		CWD:     tmpDir,
	}

	plan := planner.Plan([]string{"lib/user.rb"})

	require.Len(t, plan.Runs, 2)
	assert.Equal(t, "rspec", plan.Runs[0].Job.Name)
	assert.Equal(t, "rubocop", plan.Runs[1].Job.Name)
}

func TestPlannerPlan_JobOrderAcrossRules(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "lib/user.rb")

	planner := Planner{
		Jobs: map[string]framework.Job{
			"lint": {Name: "lint", Cmd: []string{"lint"}},
			"test": {Name: "test", Cmd: []string{"test"}},
		},
		Watches: []WatchMapping{
			{Name: "first", Source: "lib/**/*.rb", Jobs: []string{"lint"}},
			{Name: "second", Source: "lib/**/*.rb", Jobs: []string{"test"}},
		},
		CWD: tmpDir,
	}

	plan := planner.Plan([]string{"lib/user.rb"})

	require.Len(t, plan.Runs, 2)
	assert.Equal(t, "lint", plan.Runs[0].Job.Name)
	assert.Equal(t, "test", plan.Runs[1].Job.Name)
}

func TestPlannerPlan_BatchMergesAndDeduplicatesTargets(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb", "spec/post_spec.rb")

	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{libToSpec()}, CWD: tmpDir}

	t.Run("two files merge into one run, batch order", func(t *testing.T) {
		plan := planner.Plan([]string{"lib/user.rb", "lib/post.rb"})

		require.Len(t, plan.Runs, 1)
		assert.Equal(t, []string{
			filepath.FromSlash("spec/user_spec.rb"),
			filepath.FromSlash("spec/post_spec.rb"),
		}, plan.Runs[0].Targets)
	})

	t.Run("same target from two files is deduplicated", func(t *testing.T) {
		shared := WatchMapping{
			Name:    "both-to-same-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/user_spec.rb"},
			Jobs:    []string{"rspec"},
		}
		p := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{shared}, CWD: tmpDir}

		plan := p.Plan([]string{"lib/a.rb", "lib/b.rb"})

		require.Len(t, plan.Runs, 1)
		assert.Equal(t, []string{"spec/user_spec.rb"}, plan.Runs[0].Targets)
	})

	t.Run("duplicate targets within one rule are deduplicated", func(t *testing.T) {
		dup := WatchMapping{
			Name:    "duplicate-targets",
			Source:  "lib/user.rb",
			Targets: []string{"spec/user_spec.rb", "spec/user_spec.rb"},
			Jobs:    []string{"rspec"},
		}
		p := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{dup}, CWD: tmpDir}

		plan := p.Plan([]string{"lib/user.rb"})

		require.Len(t, plan.Runs, 1)
		assert.Len(t, plan.Runs[0].Targets, 1)
		require.Len(t, plan.Matches, 1)
		assert.Len(t, plan.Matches[0].Existing, 1)
	})
}

func TestPlannerPlan_JobWithOnlyMissingTargetsDoesNotRun(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb")

	planner := Planner{
		Jobs: map[string]framework.Job{
			"rspec":   {Name: "rspec", Cmd: []string{"rspec"}},
			"rubocop": {Name: "rubocop", Cmd: []string{"rubocop"}},
		},
		Watches: []WatchMapping{
			{Name: "lib-to-spec", Source: "lib/**/*.rb", Targets: []string{"spec/{{match}}_spec.rb"}, Jobs: []string{"rspec"}},
			{Name: "lib-to-missing", Source: "lib/**/*.rb", Targets: []string{"missing/{{match}}.txt"}, Jobs: []string{"rubocop"}},
		},
		CWD: tmpDir,
	}

	plan := planner.Plan([]string{"lib/user.rb"})

	require.Len(t, plan.Runs, 1, "job with only missing targets must not run bare")
	assert.Equal(t, "rspec", plan.Runs[0].Job.Name)
}

func TestPlannerPlan_ReloadRuleWithJobs(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb")

	rule := libToSpec()
	rule.Reload = true
	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{rule}, CWD: tmpDir}

	plan := planner.Plan([]string{"lib/user.rb"})

	assert.True(t, plan.Reload)
	require.Len(t, plan.Runs, 1)
	assert.Equal(t, "rspec", plan.Runs[0].Job.Name)
}

func TestPlannerPlan_ReloadRuleWithoutJobs(t *testing.T) {
	planner := Planner{
		Jobs: map[string]framework.Job{},
		Watches: []WatchMapping{
			{Name: "config-reload", Source: "config/**/*.yml", Reload: true, Jobs: []string{}},
		},
		CWD: t.TempDir(),
	}

	plan := planner.Plan([]string{"config/settings.yml"})

	assert.True(t, plan.Reload)
	assert.Empty(t, plan.Runs)
	require.Len(t, plan.Matches, 1)
	assert.Empty(t, plan.Matches[0].Missing, "jobless rules should not render or stat targets")
}

func TestPlannerPlan_EmptyWatches(t *testing.T) {
	planner := Planner{Jobs: map[string]framework.Job{}, CWD: t.TempDir()}

	plan := planner.Plan([]string{"foo.rb"})

	assert.Empty(t, plan.Matches)
	assert.Empty(t, plan.Runs)
	assert.False(t, plan.Reload)
}

func TestPlannerPlan_MultipleTargetsPerRule(t *testing.T) {
	tmpDir := t.TempDir()
	writeFileTree(t, tmpDir, "spec/user_spec.rb", "spec/lib/user_spec.rb")

	rule := libToSpec()
	rule.Targets = []string{"spec/{{match}}_spec.rb", "spec/lib/{{match}}_spec.rb"}
	planner := Planner{Jobs: rspecJobs(), Watches: []WatchMapping{rule}, CWD: tmpDir}

	plan := planner.Plan([]string{"lib/user.rb"})

	require.Len(t, plan.Runs, 1)
	assert.Equal(t, []string{
		filepath.FromSlash("spec/user_spec.rb"),
		filepath.FromSlash("spec/lib/user_spec.rb"),
	}, plan.Runs[0].Targets)
}

func TestPlannerAdmit(t *testing.T) {
	cwd := t.TempDir()
	planner := Planner{IgnorePatterns: DefaultIgnorePatterns, CWD: cwd}

	t.Run("relative path passes through", func(t *testing.T) {
		path, ok := planner.Admit("lib/user.rb")
		assert.True(t, ok)
		assert.Equal(t, "lib/user.rb", path)
	})

	t.Run("absolute path becomes CWD-relative", func(t *testing.T) {
		path, ok := planner.Admit(filepath.Join(cwd, "lib", "user.rb"))
		assert.True(t, ok)
		assert.Equal(t, filepath.FromSlash("lib/user.rb"), path)
	})

	t.Run("default ignore patterns reject", func(t *testing.T) {
		_, ok := planner.Admit(".git/objects/pack/abc123")
		assert.False(t, ok)

		_, ok = planner.Admit("node_modules/lodash/index.js")
		assert.False(t, ok)
	})

	t.Run("custom ignore patterns reject", func(t *testing.T) {
		p := Planner{IgnorePatterns: []string{"vendor/**"}, CWD: cwd}
		_, ok := p.Admit("vendor/bundle/ruby/gems/rails/lib/rails.rb")
		assert.False(t, ok)
	})

	t.Run("rejected path is still returned for display", func(t *testing.T) {
		path, ok := planner.Admit(".git/config")
		assert.False(t, ok)
		assert.Equal(t, ".git/config", path)
	})

	t.Run("empty patterns ignore nothing", func(t *testing.T) {
		p := Planner{CWD: cwd}
		_, ok := p.Admit(".git/config")
		assert.True(t, ok)
	})
}
