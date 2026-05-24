package watchsession

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuildsSharedSessionSurface(t *testing.T) {
	projectDir := makeSessionTestProject(t)
	writeSessionTestFile(t, projectDir, "lib/user.rb")
	writeSessionTestFile(t, projectDir, "spec/user_spec.rb")
	writeSessionTestFile(t, projectDir, "node_modules/pkg/file.js")
	chdirSessionTestProject(t, projectDir)

	rc := sessionRuntimeConfig()
	session, err := New(rc, Options{FilterWatchDirs: true})
	require.NoError(t, err)

	assert.Equal(t, "rspec", session.Selected.Name)
	assert.Equal(t, []string{"lib", "spec"}, session.RawWatchDirs)
	assert.Equal(t, []string{"lib", "spec"}, session.WatchDirs)
	assert.Equal(t, watch.DefaultIgnorePatterns, session.IgnorePatterns)

	resolvedProjectDir, err := filepath.EvalSymlinks(projectDir)
	require.NoError(t, err)
	assert.Equal(t, resolvedProjectDir, session.CWD)

	plan := session.PlanPath(filepath.Join(projectDir, "lib/user.rb"))
	assert.Equal(t, []string{"spec/user_spec.rb"}, plan.ExistingTargets["rspec"])

	admission := session.AdmitEvent(watch.Event{
		PathName:   filepath.Join(projectDir, "node_modules/pkg/file.js"),
		EffectType: "modify",
	})
	assert.False(t, admission.Admitted)
	assert.Equal(t, "ignored", admission.Reason)

	handler := session.Handler()
	assert.Equal(t, session.Jobs, handler.Jobs)
	assert.Equal(t, session.Watches, handler.Watches)
	assert.Equal(t, session.CWD, handler.CWD)
}

func TestNewRequiresFilteredWatchDirectoriesWhenRequested(t *testing.T) {
	projectDir := makeSessionTestProject(t)
	chdirSessionTestProject(t, projectDir)

	rc := sessionRuntimeConfig()
	rc.Watches = []watch.WatchMapping{{
		Name:    "missing-lib",
		Source:  "missing/**/*.rb",
		Targets: []string{"spec/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}}

	_, err := New(rc, Options{FilterWatchDirs: true, RequireWatchDirs: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no directories to watch found in watch mappings")
}

func TestNewUsesCustomIgnorePatterns(t *testing.T) {
	projectDir := makeSessionTestProject(t)
	writeSessionTestFile(t, projectDir, "tmp/cache.rb")
	chdirSessionTestProject(t, projectDir)

	session, err := New(sessionRuntimeConfig(), Options{
		IgnorePatterns: []string{"tmp/**"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"tmp/**"}, session.IgnorePatterns)
	admission := session.AdmitEvent(watch.Event{
		PathName:   filepath.Join(projectDir, "tmp/cache.rb"),
		EffectType: "modify",
	})
	assert.False(t, admission.Admitted)
	assert.Equal(t, "ignored", admission.Reason)
}

func TestSessionAdmitPathForPreviewUsesLiveAdmission(t *testing.T) {
	projectDir := makeSessionTestProject(t)
	writeSessionTestFile(t, projectDir, "lib/user.rb")
	writeSessionTestFile(t, projectDir, "tmp/cache.rb")
	chdirSessionTestProject(t, projectDir)

	session, err := New(sessionRuntimeConfig(), Options{
		IgnorePatterns: []string{"tmp/**"},
	})
	require.NoError(t, err)

	admitted := session.AdmitPathForPreview(filepath.Join(projectDir, "lib/user.rb"))
	assert.True(t, admitted.Admitted)
	assert.Equal(t, filepath.Join("lib", "user.rb"), admitted.Path)

	ignored := session.AdmitPathForPreview("tmp/cache.rb")
	assert.False(t, ignored.Admitted)
	assert.Equal(t, filepath.Join("tmp", "cache.rb"), ignored.Path)
	assert.Equal(t, "ignored", ignored.Reason)
}

func TestSessionPlanPathMatchesLiveHandlerBatch(t *testing.T) {
	projectDir := makeSessionTestProject(t)
	writeSessionTestFile(t, projectDir, "lib/user.rb")
	writeSessionTestFile(t, projectDir, "spec/user_spec.rb")
	writeSessionTestFile(t, projectDir, "spec/spec_helper.rb")
	chdirSessionTestProject(t, projectDir)

	session, err := New(sessionRuntimeConfig(), Options{FilterWatchDirs: true})
	require.NoError(t, err)

	changedPath := filepath.Join(projectDir, "lib/user.rb")
	previewPlan := session.PlanPath(changedPath)

	var calls []sessionExecutorCall
	handler := session.Handler()
	handler.Executor = func(j job.Job, targets []string, cwd string) error {
		calls = append(calls, sessionExecutorCall{
			jobName: j.Name,
			targets: append([]string{}, targets...),
			cwd:     cwd,
		})
		return nil
	}
	liveResult := handler.HandleBatch([]string{session.NormalizePath(changedPath)})

	require.Len(t, previewPlan.JobPlans, 1)
	require.Len(t, calls, 1)
	assert.Equal(t, previewPlan.JobPlans[0].JobName, calls[0].jobName)
	assert.Equal(t, previewPlan.JobPlans[0].Targets, calls[0].targets)
	assert.Equal(t, session.CWD, calls[0].cwd)
	assert.Equal(t, []string{previewPlan.JobPlans[0].JobName}, liveResult.ExecutedJobs)
	assert.Equal(t, previewPlan.ShouldReload, liveResult.ShouldReload)
	assert.Equal(t, previewPlan.NoRunnableChanges, liveResult.NoRunnableChanges)

	noRulePlan := session.PlanPath("spec/spec_helper.rb")
	noRuleResult := handler.HandleBatch([]string{"spec/spec_helper.rb"})
	assert.Equal(t, noRulePlan.ShouldReload, noRuleResult.ShouldReload)
	assert.Equal(t, noRulePlan.NoRunnableChanges, noRuleResult.NoRunnableChanges)
}

type sessionExecutorCall struct {
	jobName string
	targets []string
	cwd     string
}

func sessionRuntimeConfig() *runtime.RuntimeConfig {
	return &runtime.RuntimeConfig{
		Use: "rspec",
		Jobs: map[string]job.Job{
			"rspec": {
				Name:          "rspec",
				Framework:     "rspec",
				Cmd:           []string{"bundle", "exec", "rspec", "{{target}}"},
				TargetPattern: "spec/**/*_spec.rb",
			},
		},
		Watches: []watch.WatchMapping{
			{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
			{
				Name:   "spec-files",
				Source: "spec/**/*_spec.rb",
				Jobs:   []string{"rspec"},
			},
		},
		Inherited: map[string]runtime.InheritedFields{},
	}
}

func makeSessionTestProject(t *testing.T) string {
	t.Helper()

	root := findSessionRepoRoot(t)
	tmpRoot := filepath.Join(root, "tmp")
	require.NoError(t, os.MkdirAll(tmpRoot, 0o755))

	projectDir, err := os.MkdirTemp(tmpRoot, "watchsession-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(projectDir))
	})
	return projectDir
}

func findSessionRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

func writeSessionTestFile(t *testing.T, root, path string) {
	t.Helper()

	fullPath := filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte("# test\n"), 0o644))
}

func chdirSessionTestProject(t *testing.T, projectDir string) {
	t.Helper()

	previousDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previousDir))
	})
}
