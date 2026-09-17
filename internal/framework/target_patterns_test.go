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

func TestNormalize_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "rspec", Normalize("RSpec"))
	assert.Equal(t, "minitest", Normalize("  Minitest  "))
	assert.Empty(t, Normalize("   "))
}

func mustResolveJob(t *testing.T, j Job) Job {
	t.Helper()
	fw, err := Get(j.FrameworkName)
	require.NoError(t, err)
	j.Framework = fw
	return j
}

func TestGet(t *testing.T) {
	fw, err := Get("  RSpec ")
	require.NoError(t, err)
	assert.Equal(t, "rspec", fw.Name)
	assert.NotNil(t, fw.Parser)
	assert.NotEmpty(t, fw.DetectPatterns)

	_, err = Get("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown framework")

	_, err = Get("   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "framework is required")
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

func TestJobTargetPatterns_UnresolvedFrameworkReturnsError(t *testing.T) {
	j := Job{
		Name: "test-job",
	}

	_, err := j.TargetPatterns()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no resolved framework")
}
