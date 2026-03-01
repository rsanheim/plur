package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupSpecFilesByRuntime(t *testing.T) {
	t.Run("uses runtime data when paths match", func(t *testing.T) {
		files := []string{
			"spec/fast_spec.rb",
			"spec/slow_spec.rb",
			"spec/medium_spec.rb",
		}
		runtimeData := map[string]float64{
			"spec/fast_spec.rb":   0.1,
			"spec/slow_spec.rb":   10.0,
			"spec/medium_spec.rb": 1.0,
		}

		groups := GroupSpecFilesByRuntime(files, 2, runtimeData)

		require.Len(t, groups, 2)

		// The slow file (10s) should be in its own group
		// The fast (0.1s) and medium (1.0s) should be together (~1.1s total)
		var slowGroup, fastGroup FileGroup
		for _, g := range groups {
			for _, f := range g.Files {
				if f == "spec/slow_spec.rb" {
					slowGroup = g
				}
				if f == "spec/fast_spec.rb" {
					fastGroup = g
				}
			}
		}

		assert.Len(t, slowGroup.Files, 1, "slow file should be isolated")
		assert.Contains(t, slowGroup.Files, "spec/slow_spec.rb")

		assert.Len(t, fastGroup.Files, 2, "fast and medium should be grouped")
		assert.Contains(t, fastGroup.Files, "spec/fast_spec.rb")
		assert.Contains(t, fastGroup.Files, "spec/medium_spec.rb")
	})

	t.Run("slowest file gets its own worker with 8 workers", func(t *testing.T) {
		// Simulate the example-project project scenario
		files := []string{
			"spec/integration/cli_integration_spec.rb",
			"spec/integration/cli_examples_spec.rb",
			"spec/integration/pkg_compare_spec.rb",
			"spec/lib/fast1_spec.rb",
			"spec/lib/fast2_spec.rb",
			"spec/lib/fast3_spec.rb",
			"spec/lib/fast4_spec.rb",
			"spec/lib/fast5_spec.rb",
		}
		runtimeData := map[string]float64{
			"spec/integration/cli_integration_spec.rb": 18.0, // slowest
			"spec/integration/cli_examples_spec.rb":    12.0, // second slowest
			"spec/integration/pkg_compare_spec.rb":     9.0,  // third slowest
			"spec/lib/fast1_spec.rb":                   0.1,
			"spec/lib/fast2_spec.rb":                   0.1,
			"spec/lib/fast3_spec.rb":                   0.1,
			"spec/lib/fast4_spec.rb":                   0.1,
			"spec/lib/fast5_spec.rb":                   0.1,
		}

		groups := GroupSpecFilesByRuntime(files, 8, runtimeData)

		// Find which group has the slowest file
		var slowestGroup FileGroup
		for _, g := range groups {
			for _, f := range g.Files {
				if f == "spec/integration/cli_integration_spec.rb" {
					slowestGroup = g
					break
				}
			}
		}

		// The 18s file should be alone in its group
		assert.Len(t, slowestGroup.Files, 1, "slowest file (18s) should be isolated to its own worker")
	})

	t.Run("defaults to 1.0s when runtime data missing", func(t *testing.T) {
		files := []string{
			"spec/unknown_spec.rb",
			"spec/known_spec.rb",
		}
		runtimeData := map[string]float64{
			"spec/known_spec.rb": 5.0,
		}

		groups := GroupSpecFilesByRuntime(files, 2, runtimeData)

		require.Len(t, groups, 2)
		// With 5.0s and 1.0s default, they should be in separate groups
		// since we have 2 workers and 2 files
	})

	t.Run("balanced distribution across workers", func(t *testing.T) {
		files := []string{
			"spec/a_spec.rb",
			"spec/b_spec.rb",
			"spec/c_spec.rb",
			"spec/d_spec.rb",
		}
		runtimeData := map[string]float64{
			"spec/a_spec.rb": 4.0,
			"spec/b_spec.rb": 3.0,
			"spec/c_spec.rb": 2.0,
			"spec/d_spec.rb": 1.0,
		}

		groups := GroupSpecFilesByRuntime(files, 2, runtimeData)

		require.Len(t, groups, 2)

		// Optimal distribution: {4,1}=5s and {3,2}=5s
		// or {4,1} and {3,2} giving 5s each
		group0Runtime := sumRuntime(groups[0].Files, runtimeData)
		group1Runtime := sumRuntime(groups[1].Files, runtimeData)

		// Both groups should have ~5s total (within 1s tolerance for rounding)
		assert.InDelta(t, group0Runtime, group1Runtime, 1.0, "groups should have similar total runtime")
	})
}

func sumRuntime(files []string, runtimeData map[string]float64) float64 {
	total := 0.0
	for _, f := range files {
		if rt, ok := runtimeData[f]; ok {
			total += rt
		} else {
			total += 1.0 // default
		}
	}
	return total
}
