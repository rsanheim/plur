package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConventionBasedTargetPattern(t *testing.T) {
	tests := []struct {
		name            string
		jobName         string
		explicitPattern string
		expected        string
	}{
		{
			name:            "rspec in name gets rspec pattern",
			jobName:         "rspec",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "rspec-integration gets rspec pattern",
			jobName:         "rspec-integration",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "rspec-fast gets rspec pattern",
			jobName:         "rspec-fast",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "minitest in name gets minitest pattern",
			jobName:         "minitest",
			explicitPattern: "",
			expected:        "test/**/*_test.rb",
		},
		{
			name:            "minitest-unit gets minitest pattern",
			jobName:         "minitest-unit",
			explicitPattern: "",
			expected:        "test/**/*_test.rb",
		},
		{
			name:            "minitest-fast gets minitest pattern",
			jobName:         "minitest-fast",
			explicitPattern: "",
			expected:        "test/**/*_test.rb",
		},
		{
			name:            "case insensitive - RSpec",
			jobName:         "RSpec-Fast",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "case insensitive - MiniTest",
			jobName:         "MiniTest-CI",
			explicitPattern: "",
			expected:        "test/**/*_test.rb",
		},
		{
			name:            "case insensitive - RSPEC",
			jobName:         "RSPEC",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "substring match - my-rspec-runner",
			jobName:         "my-rspec-runner",
			explicitPattern: "",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "substring match - custom-minitest",
			jobName:         "custom-minitest",
			explicitPattern: "",
			expected:        "test/**/*_test.rb",
		},
		{
			name:            "explicit pattern overrides convention for rspec",
			jobName:         "rspec",
			explicitPattern: "custom/**/*_custom.rb",
			expected:        "custom/**/*_custom.rb",
		},
		{
			name:            "explicit pattern overrides convention for minitest",
			jobName:         "minitest",
			explicitPattern: "integration/**/*_test.rb",
			expected:        "integration/**/*_test.rb",
		},
		{
			name:            "no convention match returns empty",
			jobName:         "rubocop",
			explicitPattern: "",
			expected:        "",
		},
		{
			name:            "custom job without convention",
			jobName:         "custom-lint",
			explicitPattern: "",
			expected:        "",
		},
		{
			name:            "empty job name returns empty",
			jobName:         "",
			explicitPattern: "",
			expected:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := Job{
				Name:          tt.jobName,
				TargetPattern: tt.explicitPattern,
			}
			result := j.GetTargetPattern()
			assert.Equal(t, tt.expected, result, "GetTargetPattern() should return expected pattern")

			// Also test GetConventionBasedTargetPattern directly when no explicit pattern
			if tt.explicitPattern == "" {
				conventionResult := j.GetConventionBasedTargetPattern()
				assert.Equal(t, tt.expected, conventionResult, "GetConventionBasedTargetPattern() should match expected")
			}
		})
	}
}

func TestGetTargetSuffix_WithConventions(t *testing.T) {
	tests := []struct {
		name            string
		jobName         string
		explicitPattern string
		expected        string
	}{
		{
			name:            "rspec convention returns _spec.rb suffix",
			jobName:         "rspec-integration",
			explicitPattern: "",
			expected:        "_spec.rb",
		},
		{
			name:            "minitest convention returns _test.rb suffix",
			jobName:         "minitest-fast",
			explicitPattern: "",
			expected:        "_test.rb",
		},
		{
			name:            "explicit pattern returns custom suffix",
			jobName:         "rspec",
			explicitPattern: "custom/**/*_custom.rb",
			expected:        "_custom.rb",
		},
		{
			name:            "no convention or pattern returns empty",
			jobName:         "rubocop",
			explicitPattern: "",
			expected:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := Job{
				Name:          tt.jobName,
				TargetPattern: tt.explicitPattern,
			}
			result := j.GetTargetSuffix()
			assert.Equal(t, tt.expected, result, "GetTargetSuffix() should return expected suffix")
		})
	}
}
