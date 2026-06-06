package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetPatternsForJob_ExplicitTargetPattern(t *testing.T) {
	j := Job{
		Name:          "custom",
		FrameworkName: "rspec",
		TargetPattern: "app/spec/**/*_spec.rb",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"app/spec/**/*_spec.rb"}, patterns)
}

func TestTargetPatternsForJob_RSpecFramework(t *testing.T) {
	j := Job{
		Name:          "rspec",
		FrameworkName: "rspec",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_spec.rb"}, patterns)
}

func TestTargetPatternsForJob_MinitestFramework(t *testing.T) {
	j := Job{
		Name:          "minitest",
		FrameworkName: "minitest",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_test.rb"}, patterns)
}

func TestTargetPatternsForJob_PassthroughNoDetectPatterns(t *testing.T) {
	j := Job{
		Name:          "lint",
		FrameworkName: "passthrough",
	}

	_, err := TargetPatternsForJob(j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no target_pattern")
	assert.Contains(t, err.Error(), "no detect patterns")
}

func TestTargetPatternsForJob_UnknownFramework(t *testing.T) {
	j := Job{
		Name:          "bad",
		FrameworkName: "nope",
	}

	_, err := TargetPatternsForJob(j)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown framework")
}

func TestTargetPatternsForJob_PassthroughWithExplicitPattern(t *testing.T) {
	j := Job{
		Name:          "lint",
		FrameworkName: "passthrough",
		TargetPattern: "app/**/*.rb",
	}

	patterns, err := TargetPatternsForJob(j)
	require.NoError(t, err)
	assert.Equal(t, []string{"app/**/*.rb"}, patterns)
}

func TestNormalize_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "rspec", Normalize("RSpec"))
	assert.Equal(t, "minitest", Normalize("  Minitest  "))
	assert.Equal(t, "", Normalize("   "))
}

func TestIsKnown(t *testing.T) {
	assert.True(t, IsKnown("rspec"))
	assert.True(t, IsKnown("RSpec"))
	assert.True(t, IsKnown("minitest"))
	assert.True(t, IsKnown("go-test"))
	assert.False(t, IsKnown("unknown"))
}

func TestTargetPatternsForJobWithFramework_MultipleDetectPatterns(t *testing.T) {
	j := Job{Name: "test-job"}
	fw := Framework{
		Name:           "custom",
		DetectPatterns: []string{"**/*_test.go", "**/*_spec.go"},
	}

	patterns, err := TargetPatternsForJobWithFramework(j, fw)
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_test.go", "**/*_spec.go"}, patterns)
}

func TestTargetPatternsForJobWithFramework_ExplicitTargetPatternTakesPrecedence(t *testing.T) {
	j := Job{
		Name:          "test-job",
		TargetPattern: "my/**/*.rb",
	}
	fw := Framework{
		Name:           "rspec",
		DetectPatterns: []string{"**/*_spec.rb"},
	}

	patterns, err := TargetPatternsForJobWithFramework(j, fw)
	require.NoError(t, err)
	assert.Equal(t, []string{"my/**/*.rb"}, patterns)
}

func TestTargetPatternsForJobWithFramework_EmptyDetectPatternsReturnsError(t *testing.T) {
	j := Job{
		Name: "lint",
	}
	fw := Framework{
		Name:           "passthrough",
		DetectPatterns: nil,
	}

	_, err := TargetPatternsForJobWithFramework(j, fw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `job "lint"`)
	assert.Contains(t, err.Error(), `framework "passthrough"`)
}
