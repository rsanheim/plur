package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobTargetPatterns_ExplicitTargetPattern(t *testing.T) {
	j := Job{
		Name:          "custom",
		FrameworkName: "rspec",
		TargetPattern: "app/spec/**/*_spec.rb",
	}
	j = mustResolveJob(t, j)

	patterns, err := j.TargetPatterns()
	require.NoError(t, err)
	assert.Equal(t, []string{"app/spec/**/*_spec.rb"}, patterns)
}

func TestJobTargetPatterns_RSpecFramework(t *testing.T) {
	j := Job{
		Name:          "rspec",
		FrameworkName: "rspec",
	}
	j = mustResolveJob(t, j)

	patterns, err := j.TargetPatterns()
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_spec.rb"}, patterns)
}

func TestJobTargetPatterns_MinitestFramework(t *testing.T) {
	j := Job{
		Name:          "minitest",
		FrameworkName: "minitest",
	}
	j = mustResolveJob(t, j)

	patterns, err := j.TargetPatterns()
	require.NoError(t, err)
	assert.Equal(t, []string{"**/*_test.rb"}, patterns)
}

func TestJobTargetPatterns_PassthroughNoDetectPatterns(t *testing.T) {
	j := Job{
		Name:          "lint",
		FrameworkName: "passthrough",
	}
	j = mustResolveJob(t, j)

	_, err := j.TargetPatterns()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no target_pattern")
	assert.Contains(t, err.Error(), "no detect patterns")
}

func TestResolveFramework_UnknownFramework(t *testing.T) {
	j := Job{
		Name:          "bad",
		FrameworkName: "nope",
	}

	_, err := j.ResolveFramework()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown framework")
}

func TestJobTargetPatterns_PassthroughWithExplicitPattern(t *testing.T) {
	j := Job{
		Name:          "lint",
		FrameworkName: "passthrough",
		TargetPattern: "app/**/*.rb",
	}
	j = mustResolveJob(t, j)

	patterns, err := j.TargetPatterns()
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

func mustResolveJob(t *testing.T, j Job) Job {
	t.Helper()
	resolved, err := j.ResolveFramework()
	require.NoError(t, err)
	return resolved
}

func TestJobTargetPatterns_UnresolvedFrameworkReturnsError(t *testing.T) {
	j := Job{
		Name: "test-job",
	}

	_, err := j.TargetPatterns()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no resolved framework")
}
