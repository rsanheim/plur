package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		vars     map[string]string
		expected string
	}{
		{
			name:   "expand file variable",
			target: "{{file}}",
			vars: map[string]string{
				"file": "spec/models/user_spec.rb",
			},
			expected: "spec/models/user_spec.rb",
		},
		{
			name:   "expand path and name",
			target: "spec/{{path}}/{{name}}_spec.rb",
			vars: map[string]string{
				"path": "models",
				"name": "user",
			},
			expected: "spec/models/user_spec.rb",
		},
		{
			name:   "no variables",
			target: "spec",
			vars: map[string]string{
				"file": "spec_helper.rb",
			},
			expected: "spec",
		},
		{
			name:   "empty path",
			target: "spec/{{path}}/{{name}}_spec.rb",
			vars: map[string]string{
				"path": "",
				"name": "calculator",
			},
			expected: "spec//calculator_spec.rb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTarget(tt.target, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		expectCount int
		contains    string
	}{
		{
			name:        "lib file generates spec suggestions",
			file:        "lib/services/user_service.rb",
			expectCount: 2,
			contains:    "spec/services/user_service_spec.rb",
		},
		{
			name:        "app controller generates spec suggestions",
			file:        "app/controllers/users_controller.rb",
			expectCount: 1,
			contains:    "spec/controllers/users_controller_spec.rb",
		},
		{
			name:        "generic ruby file generates spec suggestions",
			file:        "config/application.rb",
			expectCount: 2,
			contains:    "spec/application_spec.rb",
		},
		{
			name:        "spec file gets no suggestions",
			file:        "spec/models/user_spec.rb",
			expectCount: 0,
		},
		{
			name:        "test file gets no suggestions",
			file:        "test/models/user_test.rb",
			expectCount: 0,
		},
		{
			name:        "non-ruby file gets no suggestions",
			file:        "config/database.yml",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateSuggestions(tt.file)
			assert.GreaterOrEqual(t, len(suggestions), tt.expectCount)
			if tt.contains != "" && len(suggestions) > 0 {
				found := false
				for _, s := range suggestions {
					if s == tt.contains {
						found = true
						break
					}
				}
				assert.True(t, found, "expected suggestions to contain %s, got %v", tt.contains, suggestions)
			}
		})
	}
}

func TestGenerateSuggestionsForFramework(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		framework   string
		expectCount int
		contains    string
	}{
		{
			name:        "lib file with minitest",
			file:        "lib/calculator.rb",
			framework:   "minitest",
			expectCount: 2,
			contains:    "test/calculator_test.rb",
		},
		{
			name:        "lib file with rspec",
			file:        "lib/calculator.rb",
			framework:   "rspec",
			expectCount: 2,
			contains:    "spec/calculator_spec.rb",
		},
		{
			name:        "app model with minitest",
			file:        "app/models/user.rb",
			framework:   "minitest",
			expectCount: 1,
			contains:    "test/models/user_test.rb",
		},
		{
			name:        "app model with rspec",
			file:        "app/models/user.rb",
			framework:   "rspec",
			expectCount: 1,
			contains:    "spec/models/user_spec.rb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateSuggestionsForFramework(tt.file, tt.framework)
			assert.GreaterOrEqual(t, len(suggestions), tt.expectCount)
			if tt.contains != "" && len(suggestions) > 0 {
				found := false
				for _, s := range suggestions {
					if s == tt.contains {
						found = true
						break
					}
				}
				assert.True(t, found, "expected suggestions to contain %s, got %v", tt.contains, suggestions)
			}
		})
	}
}
