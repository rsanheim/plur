package main

import (
	"testing"
	"time"

	"github.com/rsanheim/plur/rspec"
)

// Complexity detection tests measure execution time at multiple scales
// and alert if time growth exceeds expected O(n) behavior.
//
// If doubling input more than doubles time (with 1.5x tolerance), we may have O(n²).

// TestGrouperComplexity detects if grouping algorithm degrades beyond O(n)
func TestGrouperComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complexity test in short mode")
	}

	sizes := []int{1000, 2000, 4000, 8000}
	workers := 8
	iterations := 50

	times := make([]time.Duration, len(sizes))

	for idx, size := range sizes {
		files := generateSpecFiles(size)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			GroupSpecFilesBySize(files, workers)
		}
		times[idx] = time.Since(start) / time.Duration(iterations)

		t.Logf("GroupSpecFilesBySize: size=%5d, time=%v", size, times[idx])
	}

	checkLinearScaling(t, "GroupSpecFilesBySize", sizes, times)
}

// TestGrouperRuntimeComplexity detects if runtime-based grouping degrades beyond O(n)
func TestGrouperRuntimeComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complexity test in short mode")
	}

	sizes := []int{1000, 2000, 4000, 8000}
	workers := 8
	iterations := 50

	times := make([]time.Duration, len(sizes))

	for idx, size := range sizes {
		files := generateSpecFiles(size)
		runtimeData := generateRuntimeData(files)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			GroupSpecFilesByRuntime(files, workers, runtimeData)
		}
		times[idx] = time.Since(start) / time.Duration(iterations)

		t.Logf("GroupSpecFilesByRuntime: size=%5d, time=%v", size, times[idx])
	}

	checkLinearScaling(t, "GroupSpecFilesByRuntime", sizes, times)
}

// TestRSpecParserComplexity detects if parsing degrades beyond O(n)
func TestRSpecParserComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complexity test in short mode")
	}

	sizes := []int{1000, 2000, 4000, 8000}
	iterations := 20

	times := make([]time.Duration, len(sizes))

	for idx, size := range sizes {
		lines := generateRSpecJSONLines(size, 0.02)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			parser := rspec.NewOutputParser()
			for _, line := range lines {
				parser.ParseLine(line)
			}
		}
		times[idx] = time.Since(start) / time.Duration(iterations)

		t.Logf("RSpecParser: size=%5d, time=%v", size, times[idx])
	}

	checkLinearScaling(t, "RSpecParser", sizes, times)
}

// TestTestCollectorComplexity detects if test collection degrades beyond O(n)
func TestTestCollectorComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complexity test in short mode")
	}

	sizes := []int{1000, 2000, 4000, 8000}
	iterations := 20

	times := make([]time.Duration, len(sizes))

	for idx, size := range sizes {
		notifications := generateTestNotifications(size, 0.05)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			collector := NewTestCollector()
			for _, n := range notifications {
				collector.AddNotification(n)
			}
			collector.BuildResult(5 * time.Second)
		}
		times[idx] = time.Since(start) / time.Duration(iterations)

		t.Logf("TestCollector: size=%5d, time=%v", size, times[idx])
	}

	checkLinearScaling(t, "TestCollector", sizes, times)
}

// checkLinearScaling verifies that time grows at most linearly with input size.
// If time ratio exceeds 2x the size ratio, we likely have O(n²) behavior.
func checkLinearScaling(t *testing.T, name string, sizes []int, times []time.Duration) {
	t.Helper()

	for i := 1; i < len(sizes); i++ {
		sizeRatio := float64(sizes[i]) / float64(sizes[i-1])
		timeRatio := float64(times[i]) / float64(times[i-1])

		// Allow 2x tolerance over linear scaling to account for system noise at small scales
		threshold := sizeRatio * 2.0

		if timeRatio > threshold {
			t.Errorf("%s: potential O(n²) detected between size %d and %d: "+
				"size ratio=%.2f, time ratio=%.2f (threshold=%.2f)",
				name, sizes[i-1], sizes[i], sizeRatio, timeRatio, threshold)
		}
	}
}
