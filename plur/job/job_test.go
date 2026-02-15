package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTargetPattern(t *testing.T) {
	tests := []struct {
		name            string
		explicitPattern string
		expected        string
	}{
		{
			name:            "explicit pattern returns value",
			explicitPattern: "spec/**/*_spec.rb",
			expected:        "spec/**/*_spec.rb",
		},
		{
			name:            "empty pattern returns empty string",
			explicitPattern: "",
			expected:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := Job{
				TargetPattern: tt.explicitPattern,
			}
			result := j.GetTargetPattern()
			assert.Equal(t, tt.expected, result, "GetTargetPattern() should return expected pattern")
		})
	}
}
