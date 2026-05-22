package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/testruntime"
	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerWorkerBudget(t *testing.T) {
	files := []string{"spec/a.rb", "spec/b.rb", "spec/c.rb"}
	runtimes := map[string]float64{
		"spec/a.rb": 6.0,
		"spec/b.rb": 2.0,
		// c missing → 1.0 default
	}
	assert.InDelta(t, 3.0, perWorkerBudget(runtimes, files, 3), 0.001)
}

func TestPerWorkerBudget_ZeroWorkers(t *testing.T) {
	assert.Equal(t, 0.0, perWorkerBudget(nil, []string{"a"}, 0))
}

func TestCache_ExampleLines(t *testing.T) {
	cache := testruntime.NewCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 0, 0, 1.0, map[string]*testruntime.ExampleEntry{
		"./spec/foo_spec.rb[1:1]": {LineNumber: 20, RuntimeSeconds: 1.0},
		"./spec/foo_spec.rb[1:2]": {LineNumber: 5, RuntimeSeconds: 1.0},
		"./spec/foo_spec.rb[1:3]": {LineNumber: 5, RuntimeSeconds: 1.0}, // duplicate line — dedup
		"./spec/foo_spec.rb[2:1]": {LineNumber: 0, RuntimeSeconds: 1.0}, // zero line — skip
	})

	lines := cache.ExampleLines("spec/foo_spec.rb")
	assert.ElementsMatch(t, []int{5, 20}, lines, "duplicates and zero-line entries are dropped")
}

func TestExpandRspecSplits_SplitsLongFile(t *testing.T) {
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "slow_spec.rb")
	writeFile(t, specPath, "# slow spec\n")

	cfg := &config.GlobalConfig{
		WorkerCount: 4,
		RuntimeDir:  tempDir,
		RspecSplit:  true,
	}
	rspecJob := job.Job{Name: "rspec", Framework: "rspec"}
	runner, err := NewRunner(cfg, []string{specPath}, rspecJob, nil)
	require.NoError(t, err)

	// Seed v2 cache with a long-running file and 4 example lines.
	cache := runner.tracker.Cache()
	mtime, size, ok := testruntime.SourceFreshness(specPath)
	require.True(t, ok)
	cache.MergeAggregateRun(specPath, mtime, size, 8.0, map[string]*testruntime.ExampleEntry{
		"id1": {LineNumber: 5, RuntimeSeconds: 2.0},
		"id2": {LineNumber: 10, RuntimeSeconds: 2.0},
		"id3": {LineNumber: 15, RuntimeSeconds: 2.0},
		"id4": {LineNumber: 20, RuntimeSeconds: 2.0},
	})

	fileRuntimes := map[string]float64{specPath: 8.0}
	expandedFiles, expandedRuntimes := runner.expandRspecSplits(fileRuntimes)

	assert.Len(t, expandedFiles, 4, "should split into 4 chunks for 4 workers")
	for _, target := range expandedFiles {
		assert.Contains(t, target, specPath+":", "every target should be file:line form")
		assert.InDelta(t, 2.0, expandedRuntimes[target], 0.001, "per-chunk runtime = total/chunks")
	}
}

func TestExpandRspecSplits_PassesThroughFreshButShortFile(t *testing.T) {
	tempDir := t.TempDir()
	fastPath := filepath.Join(tempDir, "fast_spec.rb")
	slowPath := filepath.Join(tempDir, "slow_spec.rb")
	writeFile(t, fastPath, "# fast spec\n")
	writeFile(t, slowPath, "# slow spec\n")

	cfg := &config.GlobalConfig{WorkerCount: 4, RuntimeDir: tempDir, RspecSplit: true}
	rspecJob := job.Job{Name: "rspec", Framework: "rspec"}
	runner, err := NewRunner(cfg, []string{fastPath, slowPath}, rspecJob, nil)
	require.NoError(t, err)

	cache := runner.tracker.Cache()
	fastMtime, fastSize, ok := testruntime.SourceFreshness(fastPath)
	require.True(t, ok)
	slowMtime, slowSize, ok := testruntime.SourceFreshness(slowPath)
	require.True(t, ok)

	cache.MergeAggregateRun(fastPath, fastMtime, fastSize, 0.5, map[string]*testruntime.ExampleEntry{
		"id1": {LineNumber: 5, RuntimeSeconds: 0.25},
		"id2": {LineNumber: 10, RuntimeSeconds: 0.25},
	})
	cache.MergeAggregateRun(slowPath, slowMtime, slowSize, 19.5, map[string]*testruntime.ExampleEntry{
		"id3": {LineNumber: 1, RuntimeSeconds: 19.5},
	})

	// Budget = (0.5 + 19.5) / 4 = 5.0 s. fast (0.5s) is way under; slow has
	// only one example line so cannot be split.
	expanded, expandedRuntimes := runner.expandRspecSplits(map[string]float64{
		fastPath: 0.5,
		slowPath: 19.5,
	})
	assert.ElementsMatch(t, []string{fastPath, slowPath}, expanded, "neither file should be split")
	assert.Equal(t, 0.5, expandedRuntimes[fastPath])
	assert.Equal(t, 19.5, expandedRuntimes[slowPath])
}

func TestExpandRspecSplits_PassesThroughStaleCache(t *testing.T) {
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "slow_spec.rb")
	writeFile(t, specPath, "# original\n")

	cfg := &config.GlobalConfig{WorkerCount: 4, RuntimeDir: tempDir, RspecSplit: true}
	rspecJob := job.Job{Name: "rspec", Framework: "rspec"}
	runner, err := NewRunner(cfg, []string{specPath}, rspecJob, nil)
	require.NoError(t, err)

	cache := runner.tracker.Cache()
	// Seed cache with a different mtime/size than what's on disk.
	cache.MergeAggregateRun(specPath, 1, 999999, 8.0, map[string]*testruntime.ExampleEntry{
		"id1": {LineNumber: 5, RuntimeSeconds: 4.0},
		"id2": {LineNumber: 10, RuntimeSeconds: 4.0},
	})

	expanded, expandedRuntimes := runner.expandRspecSplits(map[string]float64{specPath: 8.0})
	assert.Equal(t, []string{specPath}, expanded, "stale cache must fall back to file-level")
	assert.Equal(t, 8.0, expandedRuntimes[specPath])
}

func TestShouldExpandSplits(t *testing.T) {
	tempDir := t.TempDir()
	rspecJob := job.Job{Name: "rspec", Framework: "rspec"}
	minitestJob := job.Job{Name: "minitest", Framework: "minitest"}

	cases := []struct {
		name string
		cfg  *config.GlobalConfig
		j    job.Job
		want bool
	}{
		{"flag off", &config.GlobalConfig{WorkerCount: 4, RuntimeDir: tempDir}, rspecJob, false},
		{"single worker", &config.GlobalConfig{WorkerCount: 1, RspecSplit: true, RuntimeDir: tempDir}, rspecJob, false},
		{"non-rspec job", &config.GlobalConfig{WorkerCount: 4, RspecSplit: true, RuntimeDir: tempDir}, minitestJob, false},
		{"all conditions met", &config.GlobalConfig{WorkerCount: 4, RspecSplit: true, RuntimeDir: tempDir}, rspecJob, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := NewRunner(tc.cfg, nil, tc.j, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.want, runner.shouldExpandSplits())
		})
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))
}
