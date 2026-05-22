package testruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedFile populates a Cache with one file entry whose RuntimeSeconds matches
// the sum of per-example runtimes (so the budget-check math reads naturally
// at the call sites).
func seedFile(filePath string, fileRuntime float64, examples map[string]*ExampleEntry) *Cache {
	c := NewCache()
	c.MergeAggregateRun(filePath, 0, 0, fileRuntime, examples)
	return c
}

func TestSplitFile_NoSplitConditions(t *testing.T) {
	t.Run("runtime within budget", func(t *testing.T) {
		c := seedFile("spec/fast.rb", 1.0, map[string]*ExampleEntry{
			"a": {LineNumber: 5, RuntimeSeconds: 0.5},
			"b": {LineNumber: 10, RuntimeSeconds: 0.5},
		})
		got := c.SplitFile("spec/fast.rb", 4, 2.0)
		assert.Equal(t, SplitDecision{"spec/fast.rb": 1.0}, got)
	})

	t.Run("single worker", func(t *testing.T) {
		c := seedFile("spec/slow.rb", 10.0, map[string]*ExampleEntry{
			"a": {LineNumber: 5, RuntimeSeconds: 5.0},
			"b": {LineNumber: 10, RuntimeSeconds: 5.0},
		})
		got := c.SplitFile("spec/slow.rb", 1, 2.0)
		assert.Equal(t, SplitDecision{"spec/slow.rb": 10.0}, got)
	})

	t.Run("zero budget", func(t *testing.T) {
		c := seedFile("spec/slow.rb", 100.0, map[string]*ExampleEntry{
			"a": {LineNumber: 5, RuntimeSeconds: 50.0},
			"b": {LineNumber: 10, RuntimeSeconds: 50.0},
		})
		got := c.SplitFile("spec/slow.rb", 4, 0)
		assert.Equal(t, SplitDecision{"spec/slow.rb": 100.0}, got)
	})

	t.Run("fewer than two examples", func(t *testing.T) {
		c := seedFile("spec/slow.rb", 10.0, map[string]*ExampleEntry{
			"only": {LineNumber: 12, RuntimeSeconds: 10.0},
		})
		got := c.SplitFile("spec/slow.rb", 4, 2.0)
		assert.Equal(t, SplitDecision{"spec/slow.rb": 10.0}, got)
	})

	t.Run("unknown file", func(t *testing.T) {
		c := NewCache()
		got := c.SplitFile("spec/never_seen.rb", 4, 2.0)
		assert.Equal(t, SplitDecision{"spec/never_seen.rb": 0}, got, "unknown file produces a defensive zero-runtime entry")
	})
}

func TestSplitFile_BinPackingBalancesEvenRuntimes(t *testing.T) {
	c := seedFile("spec/slow.rb", 8.0, map[string]*ExampleEntry{
		"a": {LineNumber: 5, RuntimeSeconds: 1.0},
		"b": {LineNumber: 10, RuntimeSeconds: 1.0},
		"c": {LineNumber: 15, RuntimeSeconds: 1.0},
		"d": {LineNumber: 20, RuntimeSeconds: 1.0},
		"e": {LineNumber: 25, RuntimeSeconds: 1.0},
		"f": {LineNumber: 30, RuntimeSeconds: 1.0},
		"g": {LineNumber: 35, RuntimeSeconds: 1.0},
		"h": {LineNumber: 40, RuntimeSeconds: 1.0},
	})
	got := c.SplitFile("spec/slow.rb", 4, 2.0)
	require.Len(t, got, 4)
	for target, rt := range got {
		assert.InDelta(t, 2.0, rt, 0.001, "all bins balanced at 2.0s for target %s", target)
	}
}

func TestSplitFile_LongestProcessingTimeIsolatesHeavyExample(t *testing.T) {
	c := seedFile("spec/slow.rb", 7.0, map[string]*ExampleEntry{
		"heavy": {LineNumber: 5, RuntimeSeconds: 5.0},
		"a":     {LineNumber: 10, RuntimeSeconds: 0.5},
		"b":     {LineNumber: 15, RuntimeSeconds: 0.5},
		"c":     {LineNumber: 20, RuntimeSeconds: 0.5},
		"d":     {LineNumber: 25, RuntimeSeconds: 0.5},
	})
	got := c.SplitFile("spec/slow.rb", 4, 1.0)

	// The heavy example should be alone in its bin (LPT property): there
	// exists exactly one target containing line 5, and that target's runtime
	// equals the heavy example's runtime.
	var heavyTarget string
	var heavyRuntime float64
	for target, rt := range got {
		if target == "spec/slow.rb:5" {
			heavyTarget = target
			heavyRuntime = rt
		}
	}
	require.Equal(t, "spec/slow.rb:5", heavyTarget, "heavy example must end up isolated")
	assert.InDelta(t, 5.0, heavyRuntime, 0.001)

	// The remaining bins should not be empty — total of light bins == 2.0.
	var lightSum float64
	for target, rt := range got {
		if target != heavyTarget {
			lightSum += rt
		}
	}
	assert.InDelta(t, 2.0, lightSum, 0.001, "light examples spread across remaining bins")
}

func TestSplitFile_ZeroRuntimeFallsBackToMean(t *testing.T) {
	// File has 4 examples; two with runtime, two without. Mean = 4.0/4 = 1.0
	// so the zero-runtime examples each contribute 1.0 to their bin.
	c := seedFile("spec/slow.rb", 4.0, map[string]*ExampleEntry{
		"a": {LineNumber: 5, RuntimeSeconds: 2.0},
		"b": {LineNumber: 10, RuntimeSeconds: 0},
		"c": {LineNumber: 15, RuntimeSeconds: 2.0},
		"d": {LineNumber: 20, RuntimeSeconds: 0},
	})
	got := c.SplitFile("spec/slow.rb", 2, 1.0)
	require.Len(t, got, 2)
	var total float64
	for _, rt := range got {
		total += rt
	}
	// Two real (2.0 each) + two fallbacks (1.0 each) = 6.0
	assert.InDelta(t, 6.0, total, 0.001)
}

func TestSplitFile_ChunksBoundedByExampleCount(t *testing.T) {
	c := seedFile("spec/slow.rb", 6.0, map[string]*ExampleEntry{
		"a": {LineNumber: 5, RuntimeSeconds: 3.0},
		"b": {LineNumber: 10, RuntimeSeconds: 3.0},
	})
	got := c.SplitFile("spec/slow.rb", 8, 2.0)
	require.Len(t, got, 2, "chunks bounded by 2 known example lines")
	assert.Contains(t, got, "spec/slow.rb:5")
	assert.Contains(t, got, "spec/slow.rb:10")
}

func TestSplitFile_TargetLinesAreSortedAscending(t *testing.T) {
	c := seedFile("spec/slow.rb", 6.0, map[string]*ExampleEntry{
		"a": {LineNumber: 40, RuntimeSeconds: 1.0},
		"b": {LineNumber: 10, RuntimeSeconds: 1.0},
		"c": {LineNumber: 25, RuntimeSeconds: 1.0},
		"d": {LineNumber: 5, RuntimeSeconds: 1.0},
		"e": {LineNumber: 20, RuntimeSeconds: 1.0},
		"f": {LineNumber: 15, RuntimeSeconds: 1.0},
	})
	got := c.SplitFile("spec/slow.rb", 3, 1.0)
	require.Len(t, got, 3)
	// Every target must list its lines in ascending order regardless of input.
	for target := range got {
		assertLinesAscending(t, target)
	}
}

// assertLinesAscending parses "file:line:line:..." and asserts the lines are
// sorted. We don't pin exact bin contents because LPT bin selection on ties
// is implementation-specific and the spec only guarantees deterministic,
// balanced output — not which line lands in which bin.
func assertLinesAscending(t *testing.T, target string) {
	t.Helper()
	parts := splitColons(target)
	// parts[0] is the file path; parts[1..] are line numbers.
	var prev int
	for i, p := range parts[1:] {
		n := atoi(t, p)
		if i > 0 {
			assert.Less(t, prev, n, "target lines must be ascending: %s", target)
		}
		prev = n
	}
}

func splitColons(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, c := range s {
		require.True(t, c >= '0' && c <= '9', "non-numeric segment %q", s)
		n = n*10 + int(c-'0')
	}
	return n
}

func TestSplitFile_DeterministicAcrossRepeatedCalls(t *testing.T) {
	c := seedFile("spec/slow.rb", 8.0, map[string]*ExampleEntry{
		"a": {LineNumber: 5, RuntimeSeconds: 2.0},
		"b": {LineNumber: 10, RuntimeSeconds: 1.5},
		"c": {LineNumber: 15, RuntimeSeconds: 1.5},
		"d": {LineNumber: 20, RuntimeSeconds: 1.0},
		"e": {LineNumber: 25, RuntimeSeconds: 1.0},
		"f": {LineNumber: 30, RuntimeSeconds: 1.0},
	})
	first := c.SplitFile("spec/slow.rb", 3, 1.0)
	for i := 0; i < 50; i++ {
		again := c.SplitFile("spec/slow.rb", 3, 1.0)
		assert.Equal(t, first, again, "iteration %d diverged", i)
	}
}
