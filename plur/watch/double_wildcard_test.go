package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchDoubleWildcard(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		filePath string
		expected bool
	}{
		{
			name:     "lib nested file",
			pattern:  "lib/**/*.rb",
			filePath: "lib/models/temp_model.rb",
			expected: true,
		},
		{
			name:     "lib direct file",
			pattern:  "lib/**/*.rb",
			filePath: "lib/calculator.rb",
			expected: false, // This shouldn't match because there's no subdirectory
		},
		{
			name:     "lib direct file with single star",
			pattern:  "lib/*.rb",
			filePath: "lib/calculator.rb",
			expected: false, // matchDoubleWildcard only handles ** patterns
		},
		{
			name:     "app nested file",
			pattern:  "app/**/*.rb",
			filePath: "app/models/user.rb",
			expected: true,
		},
		{
			name:     "spec nested file",
			pattern:  "**/*_spec.rb",
			filePath: "spec/models/user_spec.rb",
			expected: true,
		},
		{
			name:     "spec deeply nested file",
			pattern:  "**/*_spec.rb",
			filePath: "spec/lib/validators/email_validator_spec.rb",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchDoubleWildcard(tt.pattern, tt.filePath)
			assert.Equal(t, tt.expected, result, "matchDoubleWildcard(%q, %q)", tt.pattern, tt.filePath)
		})
	}
}

func TestFileMapperWithNestedLib(t *testing.T) {
	// Test the exact case that's failing
	config := NewMappingConfig()
	config.ProvideFeedback = false
	config.ShowSuggestions = false
	err := config.CompileRules()
	if err != nil {
		t.Fatal(err)
	}
	fm := NewFileMapperWithConfig(config)

	// Test the specific file that's failing in the integration test
	specs := fm.MapFileToSpecs("lib/models/temp_model.rb")
	assert.NotNil(t, specs, "should map lib/models/temp_model.rb")
	assert.Equal(t, []string{"spec/models/temp_model_spec.rb"}, specs)
}
