package main

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/rsanheim/plur/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildTestSummary(t *testing.T) {
	results := []WorkerResult{
		{
			ExampleCount: 10,
			FailureCount: 0,
			FileLoadTime: 50 * time.Millisecond,
			Tests:        []types.TestCaseNotification{},
		},
		{
			ExitCode:     1,
			ExampleCount: 5,
			FailureCount: 2,
			FileLoadTime: 75 * time.Millisecond,
			Tests: []types.TestCaseNotification{
				{
					Event:           types.TestFailed,
					TestID:          "test-1",
					FullDescription: "Controller GET /index returns 200",
					LineNumber:      10,
				},
				{
					Event:           types.TestFailed,
					TestID:          "test-2",
					FullDescription: "Controller POST /create creates resource",
					LineNumber:      20,
				},
			},
		},
		{
			ExitCode:     1,
			ExampleCount: 0,
			FailureCount: 0,
			FileLoadTime: 25 * time.Millisecond,
			Error:        errors.New("Failed to load spec file"),
		},
	}

	wallTime := 250 * time.Millisecond
	summary := BuildTestSummary(results, wallTime)

	assert := assert.New(t)
	assert.Equal(15, summary.TotalExamples)
	assert.Equal(2, summary.TotalFailures, "total failures")
	assert.Len(summary.AllFailures, 2, "failure details")

	assert.Equal(wallTime, summary.WallTime, "wall time")
	assert.Equal(75*time.Millisecond, summary.TotalFileLoadTime, "file load time should be the max of all workers")

	assert.Equal(1, summary.ExitCode)

	assert.Len(summary.ErroredFiles, 1, "errored files")
}

func TestBuildTestSummaryNoFailures(t *testing.T) {
	results := []WorkerResult{
		{
			ExampleCount: 10,
			FailureCount: 0,
			FileLoadTime: 40 * time.Millisecond,
		},
		{
			ExampleCount: 5,
			FailureCount: 0,
			FileLoadTime: 60 * time.Millisecond,
		},
	}

	summary := BuildTestSummary(results, 250*time.Millisecond)

	assert.Equal(t, 15, summary.TotalExamples)
	assert.Equal(t, 0, summary.TotalFailures)
	assert.Zero(t, summary.ExitCode)
	assert.Empty(t, summary.AllFailures, "should have no failures")
	assert.Empty(t, summary.ErroredFiles, "should have no errored files")
	assert.Empty(t, summary.FormattedSummary, "summary with multiple results should be empty")
}

func TestBuildTestSummaryExitCodePrecedence(t *testing.T) {
	failure := WorkerResult{ExitCode: 17, ExampleCount: 1, FailureCount: 1}
	suiteError := WorkerResult{ExitCode: 3, ExampleCount: 1, ErrorCount: 1}
	loadError := WorkerResult{ExitCode: 5, ErrorCount: 1}
	otherFailure := WorkerResult{ExitCode: 42, ExampleCount: 1}
	workerError := WorkerResult{ExitCode: workerErrorExitCode, AbnormalExit: true}
	configured70 := WorkerResult{ExitCode: 70, ExampleCount: 1, FailureCount: 1}

	for _, tt := range []struct {
		name     string
		results  []WorkerResult
		want     int
		abnormal bool
	}{
		{"normal configured 70", []WorkerResult{configured70}, 70, false},
		{"failure", []WorkerResult{failure}, 17, false},
		{"suite error after failure", []WorkerResult{failure, suiteError}, 3, false},
		{"suite error before failure", []WorkerResult{suiteError, failure}, 3, false},
		{"load error after failure", []WorkerResult{failure, loadError}, 5, false},
		{"first failing worker", []WorkerResult{failure, otherFailure}, 17, false},
		{"first errored worker", []WorkerResult{suiteError, loadError}, 3, false},
		{"worker error overrides framework error", []WorkerResult{suiteError, workerError}, workerErrorExitCode, true},
		{"framework error cannot override worker error", []WorkerResult{workerError, suiteError}, workerErrorExitCode, true},
		{"configured code 70 is not a worker error", []WorkerResult{configured70, suiteError}, 3, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			summary := BuildTestSummary(tt.results, 0)
			assert.Equal(t, tt.want, summary.ExitCode)
			assert.Equal(t, tt.abnormal, slices.ContainsFunc(summary.ErroredFiles, func(result WorkerResult) bool {
				return result.AbnormalExit
			}))
		})
	}
}

func TestSingleWorkerResultIsSingleWorkerMode(t *testing.T) {
	results := []WorkerResult{
		{
			ExampleCount:     10,
			FailureCount:     0,
			FileLoadTime:     30 * time.Millisecond,
			FormattedSummary: "10 examples, 0 failures",
		},
	}

	summary := BuildTestSummary(results, 100*time.Millisecond)

	assert.Equal(t, 10, summary.TotalExamples)
	assert.Zero(t, summary.ExitCode)
	assert.Equal(t, "10 examples, 0 failures", summary.FormattedSummary)
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
