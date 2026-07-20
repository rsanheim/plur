package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildTestSummary(t *testing.T) {
	results := []WorkerResult{
		{
			State:        types.StateSuccess,
			ExampleCount: 10,
			FailureCount: 0,
			Duration:     100 * time.Millisecond,
			FileLoadTime: 50 * time.Millisecond,
			Tests:        []types.TestCaseNotification{},
		},
		{
			State:        types.StateFailed,
			ExampleCount: 5,
			FailureCount: 2,
			Duration:     200 * time.Millisecond,
			FileLoadTime: 75 * time.Millisecond,
			Tests: []types.TestCaseNotification{
				{
					Event:           types.TestFailed,
					TestID:          "test-1",
					FullDescription: "Controller GET /index returns 200",
					LineNumber:      10,
					Exception: &types.TestException{
						Message:   "expected 200, got 404",
						Backtrace: []string{"spec/controller_spec.rb:10"},
					},
				},
				{
					Event:           types.TestFailed,
					TestID:          "test-2",
					FullDescription: "Controller POST /create creates resource",
					LineNumber:      20,
					Exception: &types.TestException{
						Message:   "expected resource to be created",
						Backtrace: []string{"spec/controller_spec.rb:20"},
					},
				},
			},
		},
		{
			State:        types.StateError,
			ExampleCount: 0,
			FailureCount: 0,
			Duration:     50 * time.Millisecond,
			FileLoadTime: 25 * time.Millisecond,
			Error:        fmt.Errorf("Failed to load spec file"),
		},
	}

	wallTime := 250 * time.Millisecond
	testJob := framework.Job{Name: "rspec", FrameworkName: "rspec"}
	summary := BuildTestSummary(results, wallTime, testJob)

	assert := assert.New(t)
	assert.Equal(15, summary.TotalExamples)
	assert.Equal(2, summary.TotalFailures, "total failures")
	assert.Equal(2, len(summary.AllFailures), "failure details")

	assert.Equal(350*time.Millisecond, summary.TotalCPUTime, "total CPU time")
	assert.Equal(wallTime, summary.WallTime, "wall time")
	assert.Equal(75*time.Millisecond, summary.TotalFileLoadTime, "file load time should be the max of all workers")

	assert.True(summary.HasFailures, "should have failures")
	assert.False(summary.Success, "should not be successful when there are failures")

	assert.Len(summary.ErroredFiles, 1, "errored files")
}

func TestBuildTestSummaryNoFailures(t *testing.T) {
	results := []WorkerResult{
		{
			State:        types.StateSuccess,
			ExampleCount: 10,
			FailureCount: 0,
			Duration:     100 * time.Millisecond,
			FileLoadTime: 40 * time.Millisecond,
		},
		{
			State:        types.StateSuccess,
			ExampleCount: 5,
			FailureCount: 0,
			Duration:     200 * time.Millisecond,
			FileLoadTime: 60 * time.Millisecond,
		},
	}

	testJob := framework.Job{Name: "rspec", FrameworkName: "rspec"}
	summary := BuildTestSummary(results, 250*time.Millisecond, testJob)

	assert.Equal(t, 15, summary.TotalExamples)
	assert.Equal(t, 0, summary.TotalFailures)
	assert.False(t, summary.HasFailures, "should have no failures when all tests pass")
	assert.True(t, summary.Success, "should be successful when all tests pass")
	assert.Empty(t, summary.AllFailures, "should have no failures")
	assert.Empty(t, summary.ErroredFiles, "should have no errored files")
	assert.Equal(t, "", summary.FormattedSummary, "summary with multiple results should be empty")
}

func TestSingleWorkerResultIsSingleWorkerMode(t *testing.T) {
	results := []WorkerResult{
		{
			State:            types.StateSuccess,
			ExampleCount:     10,
			FailureCount:     0,
			Duration:         100 * time.Millisecond,
			FileLoadTime:     30 * time.Millisecond,
			FormattedSummary: "10 examples, 0 failures",
		},
	}

	testJob := framework.Job{Name: "rspec", FrameworkName: "rspec"}
	summary := BuildTestSummary(results, 100*time.Millisecond, testJob)

	assert.Equal(t, 10, summary.TotalExamples)
	assert.True(t, summary.Success)
	assert.Equal(t, summary.FormattedSummary, "10 examples, 0 failures")
}

func TestRenumberSummaryOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "no placeholders passes through",
			in:   "Finished in 1.0 seconds\n3 examples, 0 failures\n",
			want: "Finished in 1.0 seconds\n3 examples, 0 failures\n",
		},
		{
			name: "single top-level failure",
			in:   "  ‽) does a thing\n     Failure/Error: expect(1).to eq(2)\n",
			want: "  1) does a thing\n     Failure/Error: expect(1).to eq(2)\n",
		},
		{
			name: "multiple top-level failures increment",
			in:   "  ‽) first\n  ‽) second\n  ‽) third\n",
			want: "  1) first\n  2) second\n  3) third\n",
		},
		{
			name: "aggregate sub-markers inherit parent number",
			in:   "  ‽) aggregate example\n     Got 2 failures:\n\n     ‽.1) Failure/Error: expect(1).to eq(2)\n     ‽.2) Failure/Error: expect(3).to eq(4)\n",
			want: "  1) aggregate example\n     Got 2 failures:\n\n     1.1) Failure/Error: expect(1).to eq(2)\n     1.2) Failure/Error: expect(3).to eq(4)\n",
		},
		{
			name: "aggregate nested among plain failures keeps numbering aligned",
			in:   "  ‽) first\n  ‽) aggregate\n     ‽.1) sub a\n     ‽.2) sub b\n  ‽) third\n",
			want: "  1) first\n  2) aggregate\n     2.1) sub a\n     2.2) sub b\n  3) third\n",
		},
		{
			name: "double-digit aggregate sub-index",
			in:   "  ‽) aggregate\n     ‽.10) tenth sub\n",
			want: "  1) aggregate\n     1.10) tenth sub\n",
		},
		{
			name: "stray placeholder not part of a marker is left untouched",
			in:   "  ‽) message contains a literal ‽ here\n",
			want: "  1) message contains a literal ‽ here\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, renumberSummaryOutput(tt.in))
		})
	}
}
