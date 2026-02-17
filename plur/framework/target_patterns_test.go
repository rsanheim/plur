package framework

import (
	"testing"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetPatternsForJob_ExplicitTargetPattern(t *testing.T) {
	j := job.Job{
		Name:          "custom",
		Framework:     "rspec",
		TargetPattern: "app/spec/**/*_spec.rb",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"app/spec/**/*_spec.rb"}, patterns)
}

func TestTargetPatternsForJob_RSpecFramework(t *testing.T) {
	j := job.Job{
		Name:      "rspec",
		Framework: "rspec",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_spec.rb"}, patterns)
}

func TestTargetPatternsForJob_MinitestFramework(t *testing.T) {
	j := job.Job{
		Name:      "minitest",
		Framework: "minitest",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_test.rb"}, patterns)
}

func TestTargetPatternsForJob_PassthroughNoDetectPatterns(t *testing.T) {
	j := job.Job{
		Name:      "lint",
		Framework: "passthrough",
	}

	_, err := TargetPatternsForJob(j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no target_pattern")
	assert.Contains(t, err.Error(), "no detect patterns")
}

func TestTargetPatternsForJob_UnknownFramework(t *testing.T) {
	j := job.Job{
		Name:      "bad",
		Framework: "nope",
	}

	_, err := TargetPatternsForJob(j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown framework")
}

func TestTargetPatternsForJobWithSpec_ReturnsDetectPatterns(t *testing.T) {
	j := job.Job{Name: "test-job"}
	spec := Spec{
		Name:           "custom",
		DetectPatterns: []string{"**/*_test.go", "**/*_spec.go"},
	}

	patterns, err := TargetPatternsForJobWithSpec(j, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_test.go", "**/*_spec.go"}, patterns)
}

func TestTargetPatternsForJobWithSpec_ExplicitTargetPatternTakesPrecedence(t *testing.T) {
	j := job.Job{
		Name:          "test-job",
		TargetPattern: "my/**/*.rb",
	}
	spec := Spec{
		Name:           "rspec",
		DetectPatterns: []string{"**/*_spec.rb"},
	}

	patterns, err := TargetPatternsForJobWithSpec(j, spec)
	require.NoError(t, err)
	assert.Equal(t, []string{"my/**/*.rb"}, patterns)
}

func TestTargetPatternsForJobWithSpec_EmptyDetectPatternsReturnsError(t *testing.T) {
	j := job.Job{
		Name: "lint",
	}
	spec := Spec{
		Name:           "passthrough",
		DetectPatterns: nil,
	}

	_, err := TargetPatternsForJobWithSpec(j, spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `job "lint"`)
	assert.Contains(t, err.Error(), `framework "passthrough"`)
}
