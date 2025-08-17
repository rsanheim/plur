package task

import (
	"strings"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCommand_RSpec(t *testing.T) {
	task := NewRSpecTask()
	globalConfig := &config.GlobalConfig{
		ColorOutput: true,
		ConfigPaths: &config.ConfigPaths{
			JSONRowsFormatter: "/path/to/formatter.rb",
		},
	}
	files := []string{"spec/models/user_spec.rb", "spec/controllers/posts_spec.rb"}

	cmd := task.BuildCommand(files, globalConfig, nil)
	require.NotNil(t, cmd)

	// Check command and args
	assert.Contains(t, cmd.Path, "bundle")
	args := cmd.Args[1:] // Skip the command itself

	// Should have exec rspec
	assert.Contains(t, args, "exec")
	assert.Contains(t, args, "rspec")

	// Should have formatter
	assert.Contains(t, args, "-r")
	assert.Contains(t, args, "/path/to/formatter.rb")
	assert.Contains(t, args, "--format")
	assert.Contains(t, args, "Plur::JsonRowsFormatter")

	// Should have color flags
	assert.Contains(t, args, "--force-color")
	assert.Contains(t, args, "--tty")

	// Should have the files at the end
	argsStr := strings.Join(args, " ")
	assert.Contains(t, argsStr, "spec/models/user_spec.rb")
	assert.Contains(t, argsStr, "spec/controllers/posts_spec.rb")
}

func TestBuildCommand_RSpecWithOverride(t *testing.T) {
	task := NewRSpecTask()
	override := &Task{
		Run: "bin/rspec",
	}
	files := []string{"spec/test_spec.rb"}

	cmd := task.BuildCommand(files, nil, override)
	require.NotNil(t, cmd)

	// Should use override command
	assert.Contains(t, cmd.Path, "bin/rspec")
	assert.Contains(t, cmd.Args, "spec/test_spec.rb")
}

func TestBuildCommand_Minitest(t *testing.T) {
	task := NewMinitestTask()
	files := []string{"test/models/user_test.rb", "test/controllers/posts_test.rb"}

	cmd := task.BuildCommand(files, nil, nil)
	require.NotNil(t, cmd)

	// Check minitest command structure
	assert.Contains(t, cmd.Path, "bundle")
	args := cmd.Args[1:]

	assert.Contains(t, args, "exec")
	assert.Contains(t, args, "ruby")
	assert.Contains(t, args, "-Itest")

	// Should have -r for each file
	for _, file := range files {
		// Find the position of this specific file in args
		fileIdx := indexOf(args, file)
		assert.NotEqual(t, -1, fileIdx, "File should be in args: "+file)
		// Check that -r precedes it
		if fileIdx > 0 {
			assert.Equal(t, "-r", args[fileIdx-1], "File should be preceded by -r flag")
		}
	}

	// Should end with -e nil
	assert.Contains(t, args, "-e")
	assert.Contains(t, args, "nil")
}

func TestMapFilesToTarget_RSpec(t *testing.T) {
	task := NewRSpecTask()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "lib file to spec",
			input:    []string{"lib/models/user.rb"},
			expected: []string{"spec/models/user_spec.rb"},
		},
		{
			name:     "app file to spec",
			input:    []string{"app/controllers/posts_controller.rb"},
			expected: []string{"spec/controllers/posts_controller_spec.rb"},
		},
		{
			name:     "spec file maps to itself",
			input:    []string{"spec/models/user_spec.rb"},
			expected: []string{"spec/models/user_spec.rb"},
		},
		{
			name:     "multiple files",
			input:    []string{"lib/user.rb", "app/post.rb", "spec/helper_spec.rb"},
			expected: []string{"spec/user_spec.rb", "spec/post_spec.rb", "spec/helper_spec.rb"},
		},
		{
			name:     "nested lib path",
			input:    []string{"lib/services/auth/token.rb"},
			expected: []string{"spec/services/auth/token_spec.rb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := task.MapFilesToTarget(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestMapFilesToTarget_Minitest(t *testing.T) {
	task := NewMinitestTask()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "lib file to test",
			input:    []string{"lib/models/user.rb"},
			expected: []string{"test/models/user_test.rb"},
		},
		{
			name:     "app file to test",
			input:    []string{"app/controllers/posts_controller.rb"},
			expected: []string{"test/controllers/posts_controller_test.rb"},
		},
		{
			name:     "test file maps to itself",
			input:    []string{"test/models/user_test.rb"},
			expected: []string{"test/models/user_test.rb"},
		},
		{
			name:     "nested lib path",
			input:    []string{"lib/services/auth/token.rb"},
			expected: []string{"test/services/auth/token_test.rb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := task.MapFilesToTarget(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestMapFilesToTarget_EdgeCases(t *testing.T) {
	task := &Task{
		Name:       "custom",
		SourceDirs: []string{"src"},
		Mappings: []MappingRule{
			{
				Pattern: "src/**/*.go",
				Target:  "{{path}}/{{name}}_test.go",
			},
		},
	}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no matching pattern returns empty",
			input:    []string{"random/file.txt"},
			expected: []string{},
		},
		{
			name:     "empty input returns empty",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "file without extension",
			input:    []string{"src/Makefile"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := task.MapFilesToTarget(tt.input)
			if tt.expected == nil || len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.ElementsMatch(t, tt.expected, result)
			}
		})
	}
}

func TestApplyMapping_TokenReplacement(t *testing.T) {
	task := &Task{
		SourceDirs: []string{"lib", "app"},
	}

	tests := []struct {
		name          string
		sourceFile    string
		targetPattern string
		expected      string
	}{
		{
			name:          "replace {{file}} token",
			sourceFile:    "lib/user.rb",
			targetPattern: "{{file}}",
			expected:      "lib/user.rb",
		},
		{
			name:          "replace {{name}} token",
			sourceFile:    "lib/models/user.rb",
			targetPattern: "spec/{{name}}_spec.rb",
			expected:      "spec/user_spec.rb",
		},
		{
			name:          "replace {{path}} token with lib stripped",
			sourceFile:    "lib/models/user.rb",
			targetPattern: "spec/{{path}}/{{name}}_spec.rb",
			expected:      "spec/models/user_spec.rb",
		},
		{
			name:          "replace {{path}} token with app stripped",
			sourceFile:    "app/controllers/posts.rb",
			targetPattern: "spec/{{path}}/{{name}}_spec.rb",
			expected:      "spec/controllers/posts_spec.rb",
		},
		{
			name:          "multiple tokens",
			sourceFile:    "lib/services/auth.rb",
			targetPattern: "test/{{path}}/{{name}}_test.rb",
			expected:      "test/services/auth_test.rb",
		},
		{
			name:          "root level file in source dir",
			sourceFile:    "lib/user.rb",
			targetPattern: "spec/{{path}}/{{name}}_spec.rb",
			expected:      "spec/user_spec.rb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := task.applyMapping(tt.sourceFile, tt.targetPattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
