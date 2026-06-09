package watch

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTokens(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		sourcePattern string
		expected      Tokens
	}{
		{
			name:          "simple lib file",
			path:          "lib/user.rb",
			sourcePattern: "lib/**/*.rb",
			expected: Tokens{
				Match:       "user",
				Path:        "lib/user.rb",
				Dir:         "lib/",
				DirRelative: "",
				Base:        "user.rb",
				Name:        "user",
				Ext:         ".rb",
				ExtNoDot:    "rb",
			},
		},
		{
			name:          "nested lib file",
			path:          "lib/models/user.rb",
			sourcePattern: "lib/**/*.rb",
			expected: Tokens{
				Match:       "models/user",
				Path:        "lib/models/user.rb",
				Dir:         "lib/models/",
				DirRelative: "models/",
				Base:        "user.rb",
				Name:        "user",
				Ext:         ".rb",
				ExtNoDot:    "rb",
			},
		},
		{
			name:          "app model file",
			path:          "app/models/post.rb",
			sourcePattern: "app/**/*.rb",
			expected: Tokens{
				Match:       "models/post",
				Path:        "app/models/post.rb",
				Dir:         "app/models/",
				DirRelative: "models/",
				Base:        "post.rb",
				Name:        "post",
				Ext:         ".rb",
				ExtNoDot:    "rb",
			},
		},
		{
			name:          "go test file",
			path:          "internal/task/task.go",
			sourcePattern: "**/*.go",
			expected: Tokens{
				Match:       "internal/task/task",
				Path:        "internal/task/task.go",
				Dir:         "internal/task/",
				DirRelative: "internal/task/",
				Base:        "task.go",
				Name:        "task",
				Ext:         ".go",
				ExtNoDot:    "go",
			},
		},
		{
			name:          "root level file",
			path:          "main.go",
			sourcePattern: "*.go",
			expected: Tokens{
				Match:       "main",
				Path:        "main.go",
				Dir:         "./",
				DirRelative: "",
				Base:        "main.go",
				Name:        "main",
				Ext:         ".go",
				ExtNoDot:    "go",
			},
		},
		{
			name:          "spec file",
			path:          "spec/models/user_spec.rb",
			sourcePattern: "spec/**/*_spec.rb",
			expected: Tokens{
				Match:       "models/user_spec",
				Path:        "spec/models/user_spec.rb",
				Dir:         "spec/models/",
				DirRelative: "models/",
				Base:        "user_spec.rb",
				Name:        "user_spec",
				Ext:         ".rb",
				ExtNoDot:    "rb",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTokens(tt.path, tt.sourcePattern)

			assert.Equal(t, tt.expected.Match, result.Match, "Match mismatch")
			assert.Equal(t, tt.expected.Path, result.Path, "Path mismatch")
			assert.Equal(t, tt.expected.Dir, result.Dir, "Dir mismatch")
			assert.Equal(t, tt.expected.DirRelative, result.DirRelative, "DirRelative mismatch")
			assert.Equal(t, tt.expected.Base, result.Base, "Base mismatch")
			assert.Equal(t, tt.expected.Name, result.Name, "Name mismatch")
			assert.Equal(t, tt.expected.Ext, result.Ext, "Ext mismatch")
			assert.Equal(t, tt.expected.ExtNoDot, result.ExtNoDot, "ExtNoDot mismatch")
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	tokens := Tokens{
		Match:       "models/user",
		Path:        "lib/models/user.rb",
		Dir:         "lib/models/",
		DirRelative: "models/",
		Base:        "user.rb",
		Name:        "user",
		Ext:         ".rb",
		ExtNoDot:    "rb",
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "match token",
			template: "spec/{{match}}_spec.rb",
			expected: filepath.FromSlash("spec/models/user_spec.rb"),
		},
		{
			name:     "path token",
			template: "{{path}}",
			expected: filepath.FromSlash("lib/models/user.rb"),
		},
		{
			name:     "dir token",
			template: "{{dir}}test_{{name}}.rb",
			expected: filepath.FromSlash("lib/models/test_user.rb"),
		},
		{
			name:     "dir_relative token",
			template: "spec/{{dir_relative}}{{name}}_spec.rb",
			expected: filepath.FromSlash("spec/models/user_spec.rb"),
		},
		{
			name:     "name and ext tokens",
			template: "test/{{name}}_test{{ext}}",
			expected: filepath.FromSlash("test/user_test.rb"),
		},
		{
			name:     "ext_no_dot token",
			template: "{{name}}.test.{{ext_no_dot}}",
			expected: filepath.FromSlash("user.test.rb"),
		},
		{
			name:     "multiple tokens",
			template: "{{dir}}{{name}}_backup{{ext}}",
			expected: filepath.FromSlash("lib/models/user_backup.rb"),
		},
		{
			name:     "base token",
			template: "backup/{{base}}",
			expected: filepath.FromSlash("backup/user.rb"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderTemplate(tt.template, tokens)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderTemplateErrors(t *testing.T) {
	tokens := Tokens{
		Match: "test",
		Path:  "test.rb",
	}

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "invalid token",
			template: "{{invalid}}",
		},
		{
			name:     "typo in token",
			template: "{{pth}}",
		},
		{
			name:     "unclosed token",
			template: "{{match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderTemplate(tt.template, tokens)
			assert.Error(t, err)
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid match template",
			template: "spec/{{match}}_spec.rb",
			wantErr:  false,
		},
		{
			name:     "valid multiple tokens",
			template: "{{dir}}{{name}}_test{{ext}}",
			wantErr:  false,
		},
		{
			name:     "invalid token",
			template: "{{invalid}}",
			wantErr:  true,
		},
		{
			name:     "unclosed braces",
			template: "{{match",
			wantErr:  true,
		},
		{
			name:     "no tokens is valid",
			template: "static/path.rb",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildTokensWithComplexPaths(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		sourcePattern string
		checkFunc     func(*testing.T, Tokens)
	}{
		{
			name:          "deeply nested file",
			path:          "lib/api/v1/resources/user.rb",
			sourcePattern: "lib/**/*.rb",
			checkFunc: func(t *testing.T, tok Tokens) {
				assert.Equal(t, "api/v1/resources/user", tok.Match)
				assert.Equal(t, "api/v1/resources/", tok.DirRelative)
			},
		},
		{
			name:          "file with multiple dots",
			path:          "lib/user.model.rb",
			sourcePattern: "lib/**/*.rb",
			checkFunc: func(t *testing.T, tok Tokens) {
				assert.Equal(t, "user.model", tok.Name)
				assert.Equal(t, ".rb", tok.Ext)
				assert.Equal(t, "user.model.rb", tok.Base)
			},
		},
		{
			name:          "file without extension",
			path:          "scripts/build",
			sourcePattern: "scripts/*",
			checkFunc: func(t *testing.T, tok Tokens) {
				assert.Equal(t, "build", tok.Name)
				assert.Equal(t, "", tok.Ext)
				assert.Equal(t, "", tok.ExtNoDot)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTokens(tt.path, tt.sourcePattern)
			tt.checkFunc(t, result)
		})
	}
}
