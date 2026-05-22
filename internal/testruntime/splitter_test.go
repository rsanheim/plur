package testruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitFile(t *testing.T) {
	cases := []struct {
		name          string
		filePath      string
		runtime       float64
		exampleLines  []int
		workerCount   int
		targetRuntime float64
		wantTargets   []string
		wantChunks    int
		wantChunkRT   float64
	}{
		{
			name:          "runtime within budget is not split",
			filePath:      "spec/fast_spec.rb",
			runtime:       1.0,
			exampleLines:  []int{5, 10, 15, 20},
			workerCount:   4,
			targetRuntime: 2.0,
			wantTargets:   []string{"spec/fast_spec.rb"},
			wantChunks:    1,
			wantChunkRT:   1.0,
		},
		{
			name:          "single worker is never split",
			filePath:      "spec/slow_spec.rb",
			runtime:       10.0,
			exampleLines:  []int{5, 10, 15, 20},
			workerCount:   1,
			targetRuntime: 2.0,
			wantTargets:   []string{"spec/slow_spec.rb"},
			wantChunks:    1,
			wantChunkRT:   10.0,
		},
		{
			name:          "fewer than two example lines is not split",
			filePath:      "spec/slow_spec.rb",
			runtime:       10.0,
			exampleLines:  []int{12},
			workerCount:   4,
			targetRuntime: 2.0,
			wantTargets:   []string{"spec/slow_spec.rb"},
			wantChunks:    1,
			wantChunkRT:   10.0,
		},
		{
			name:          "splits into worker_count chunks when above budget",
			filePath:      "spec/slow_spec.rb",
			runtime:       8.0,
			exampleLines:  []int{5, 10, 15, 20, 25, 30, 35, 40},
			workerCount:   4,
			targetRuntime: 2.0,
			// round-robin: bucket 0 -> [5, 25], bucket 1 -> [10, 30],
			//              bucket 2 -> [15, 35], bucket 3 -> [20, 40]
			wantTargets: []string{
				"spec/slow_spec.rb:5:25",
				"spec/slow_spec.rb:10:30",
				"spec/slow_spec.rb:15:35",
				"spec/slow_spec.rb:20:40",
			},
			wantChunks:  4,
			wantChunkRT: 2.0,
		},
		{
			name:          "chunks bounded by example count when fewer than workers",
			filePath:      "spec/slow_spec.rb",
			runtime:       6.0,
			exampleLines:  []int{5, 10},
			workerCount:   8,
			targetRuntime: 2.0,
			wantTargets: []string{
				"spec/slow_spec.rb:5",
				"spec/slow_spec.rb:10",
			},
			wantChunks:  2,
			wantChunkRT: 3.0,
		},
		{
			name:          "deterministic: sorts unsorted input",
			filePath:      "spec/slow_spec.rb",
			runtime:       6.0,
			exampleLines:  []int{20, 10, 30, 40, 5, 15},
			workerCount:   3,
			targetRuntime: 1.0,
			// sorted: [5, 10, 15, 20, 30, 40] -> round-robin into 3 buckets:
			//   bucket 0 -> [5, 20], bucket 1 -> [10, 30], bucket 2 -> [15, 40]
			wantTargets: []string{
				"spec/slow_spec.rb:5:20",
				"spec/slow_spec.rb:10:30",
				"spec/slow_spec.rb:15:40",
			},
			wantChunks:  3,
			wantChunkRT: 2.0,
		},
		{
			name:          "zero budget never splits",
			filePath:      "spec/slow_spec.rb",
			runtime:       100.0,
			exampleLines:  []int{5, 10, 15},
			workerCount:   4,
			targetRuntime: 0,
			wantTargets:   []string{"spec/slow_spec.rb"},
			wantChunks:    1,
			wantChunkRT:   100.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitFile(tc.filePath, tc.runtime, tc.exampleLines, tc.workerCount, tc.targetRuntime)
			assert.Equal(t, tc.wantTargets, got.Targets)
			assert.Equal(t, tc.wantChunks, got.Chunks)
			assert.InDelta(t, tc.wantChunkRT, got.ChunkRuntimeSeconds, 0.001)
		})
	}
}

func TestSplitFile_DoesNotMutateInput(t *testing.T) {
	lines := []int{30, 10, 20}
	SplitFile("spec/x.rb", 10, lines, 3, 1.0)
	assert.Equal(t, []int{30, 10, 20}, lines, "input slice must not be mutated")
}

func TestSplitFile_RepeatedCallsAreStable(t *testing.T) {
	first := SplitFile("spec/slow_spec.rb", 8.0, []int{5, 10, 15, 20, 25, 30}, 3, 1.0)
	second := SplitFile("spec/slow_spec.rb", 8.0, []int{5, 10, 15, 20, 25, 30}, 3, 1.0)
	assert.Equal(t, first, second)
}
