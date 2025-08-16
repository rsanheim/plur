package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		file     string
		expected bool
		vars     map[string]string
	}{
		{
			name:     "spec file match",
			pattern:  "*_spec.rb",
			file:     "calculator_spec.rb",
			expected: true,
			vars: map[string]string{
				"file": "calculator_spec.rb",
				"name": "calculator_spec",
				"path": "",
			},
		},
		{
			name:     "lib file with path",
			pattern:  "lib/**/*.rb",
			file:     "lib/models/user.rb",
			expected: true,
			vars: map[string]string{
				"file": "lib/models/user.rb",
				"path": "models",
				"name": "user",
			},
		},
		{
			name:     "app file with path",
			pattern:  "app/**/*.rb",
			file:     "app/models/user.rb",
			expected: true,
			vars: map[string]string{
				"file": "app/models/user.rb",
				"path": "models",
				"name": "user",
			},
		},
		{
			name:     "no match",
			pattern:  "spec/**/*_spec.rb",
			file:     "lib/user.rb",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, vars := matchGlobPattern(tt.pattern, tt.file)
			assert.Equal(t, tt.expected, matched)
			if tt.expected && tt.vars != nil {
				assert.Equal(t, tt.vars["name"], vars["name"])
				assert.Equal(t, tt.vars["path"], vars["path"])
			}
		})
	}
}

func TestExpandTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		vars     map[string]string
		expected string
	}{
		{
			name:   "expand file variable",
			target: "{file}",
			vars: map[string]string{
				"file": "spec/models/user_spec.rb",
			},
			expected: "spec/models/user_spec.rb",
		},
		{
			name:   "expand path and name",
			target: "spec/{path}/{name}_spec.rb",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTarget(tt.target, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMappingConfig_MatchFile(t *testing.T) {
	config := NewMappingConfig()
	err := config.CompileRules()
	require.NoError(t, err)

	tests := []struct {
		name         string
		file         string
		expectRule   bool
		expectTarget string
	}{
		{
			name:         "spec file",
			file:         "spec/models/user_spec.rb",
			expectRule:   true,
			expectTarget: "spec/models/user_spec.rb",
		},
		{
			name:         "spec_helper",
			file:         "spec/spec_helper.rb",
			expectRule:   true,
			expectTarget: "spec",
		},
		{
			name:         "lib file",
			file:         "lib/calculator.rb",
			expectRule:   true,
			expectTarget: "spec/calculator_spec.rb",
		},
		{
			name:         "app model",
			file:         "app/models/user.rb",
			expectRule:   true,
			expectTarget: "spec/models/user_spec.rb",
		},
		{
			name:       "unknown file",
			file:       "config/database.yml",
			expectRule: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, vars := config.MatchFile(tt.file)
			if tt.expectRule {
				require.NotNil(t, rule, "expected to find a rule for %s", tt.file)
				target := ExpandTarget(rule.Target, vars)
				assert.Equal(t, tt.expectTarget, target)
			} else {
				assert.Nil(t, rule)
			}
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
			name:        "lib file",
			file:        "lib/services/user_service.rb",
			expectCount: 2,
			contains:    "spec/services/user_service_spec.rb",
		},
		{
			name:        "app controller",
			file:        "app/controllers/users_controller.rb",
			expectCount: 4,
			contains:    "spec/controllers/users_controller_spec.rb",
		},
		{
			name:        "spec file gets no suggestions",
			file:        "spec/models/user_spec.rb",
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

func TestCustomRules(t *testing.T) {
	config := NewMappingConfig()

	// Add a custom rule
	config.CustomRules = []MappingRule{
		{
			Pattern:     "config/*.rb",
			Target:      "spec/config/{name}_spec.rb",
			Description: "Config files",
			Priority:    60,
			Type:        "glob",
		},
	}

	err := config.CompileRules()
	require.NoError(t, err)

	// Test that custom rule works
	rule, vars := config.MatchFile("config/application.rb")
	require.NotNil(t, rule)

	target := ExpandTarget(rule.Target, vars)
	assert.Equal(t, "spec/config/application_spec.rb", target)
}
