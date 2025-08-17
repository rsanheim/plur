package task

import (
	"os"
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

	args := task.BuildCommand(files, globalConfig, "")
	require.NotNil(t, args)

	// Check command and args
	assert.Contains(t, args[0], "bundle")

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
	files := []string{"spec/test_spec.rb"}

	args := task.BuildCommand(files, nil, "bin/rspec")
	require.NotNil(t, args)

	// Should use override command
	assert.Contains(t, args[0], "bin/rspec")
	assert.Contains(t, args, "spec/test_spec.rb")
}

func TestBuildCommand_Minitest(t *testing.T) {
	task := NewMinitestTask()
	files := []string{"test/models/user_test.rb", "test/controllers/posts_test.rb"}

	args := task.BuildCommand(files, nil, "")
	require.NotNil(t, args)

	// Check minitest command structure
	assert.Contains(t, args[0], "bundle")
	assert.Contains(t, args, "exec")
	assert.Contains(t, args, "ruby")
	assert.Contains(t, args, "-Itest")

	// For multiple files, should use -e with require pattern
	assert.Contains(t, args, "-e")

	// Find the -e flag and check the require pattern
	eIdx := indexOf(args, "-e")
	require.NotEqual(t, -1, eIdx, "Should have -e flag")
	require.Less(t, eIdx+1, len(args), "Should have argument after -e")

	requirePattern := args[eIdx+1]
	assert.Contains(t, requirePattern, "models/user_test")
	assert.Contains(t, requirePattern, "controllers/posts_test")
	assert.Contains(t, requirePattern, ".each { |f| require f }")
}

func TestBuildCommand_MinitestSingleFile(t *testing.T) {
	task := NewMinitestTask()
	files := []string{"test/models/user_test.rb"}

	args := task.BuildCommand(files, nil, "")
	require.NotNil(t, args)

	// Check minitest command structure
	assert.Contains(t, args[0], "bundle")
	assert.Contains(t, args, "exec")
	assert.Contains(t, args, "ruby")
	assert.Contains(t, args, "-Itest")

	// For single file, should pass file directly
	assert.Contains(t, args, "test/models/user_test.rb")
	assert.NotContains(t, args, "-e")
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

func TestGetTestSuffix_RSpec(t *testing.T) {
	task := NewRSpecTask()

	suffix := task.GetTestSuffix()
	assert.Equal(t, "_spec.rb", suffix)
}

func TestGetTestSuffix_Minitest(t *testing.T) {
	task := NewMinitestTask()

	suffix := task.GetTestSuffix()
	assert.Equal(t, "_test.rb", suffix)
}

func TestGetTestSuffix_Custom(t *testing.T) {
	task := &Task{
		Name:     "custom",
		TestGlob: "tests/**/*_custom.js",
	}

	suffix := task.GetTestSuffix()
	assert.Equal(t, "_custom.js", suffix)
}

func TestGetTestPattern_RSpec(t *testing.T) {
	task := NewRSpecTask()

	pattern := task.GetTestPattern()
	assert.Equal(t, "spec/**/*_spec.rb", pattern)
}

func TestGetTestPattern_Minitest(t *testing.T) {
	task := NewMinitestTask()

	pattern := task.GetTestPattern()
	assert.Equal(t, "test/**/*_test.rb", pattern)
}

func TestGetTestPattern_Custom(t *testing.T) {
	task := &Task{
		Name:     "custom",
		TestGlob: "tests/**/*.test.js",
	}

	pattern := task.GetTestPattern()
	assert.Equal(t, "tests/**/*.test.js", pattern)
}

func TestDetectFramework_RSpec(t *testing.T) {
	// Create temporary directory structure
	err := os.MkdirAll("temp_rspec_test", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("temp_rspec_test")

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir("temp_rspec_test")

	// Create spec directory
	err = os.MkdirAll("spec", 0755)
	require.NoError(t, err)

	detectedTask := DetectFramework()
	assert.Equal(t, "rspec", detectedTask.Name)
	assert.Equal(t, "_spec.rb", detectedTask.GetTestSuffix())
	assert.Equal(t, "spec/**/*_spec.rb", detectedTask.GetTestPattern())
}

func TestDetectFramework_Minitest(t *testing.T) {
	// Create temporary directory structure
	err := os.MkdirAll("temp_minitest_test", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("temp_minitest_test")

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir("temp_minitest_test")

	// Create test directory
	err = os.MkdirAll("test", 0755)
	require.NoError(t, err)

	detectedTask := DetectFramework()
	assert.Equal(t, "minitest", detectedTask.Name)
	assert.Equal(t, "_test.rb", detectedTask.GetTestSuffix())
	assert.Equal(t, "test/**/*_test.rb", detectedTask.GetTestPattern())
}

func TestDetectFramework_Mixed_PrefersMinitest(t *testing.T) {
	// Create temporary directory structure
	err := os.MkdirAll("temp_mixed_test", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("temp_mixed_test")

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir("temp_mixed_test")

	// Create both test and spec directories (minitest should win)
	err = os.MkdirAll("test", 0755)
	require.NoError(t, err)
	err = os.MkdirAll("spec", 0755)
	require.NoError(t, err)

	detectedTask := DetectFramework()
	assert.Equal(t, "minitest", detectedTask.Name, "When both test/ and spec/ exist, should prefer minitest")
}

func TestDetectFramework_DefaultToRSpec(t *testing.T) {
	// Create temporary directory structure
	err := os.MkdirAll("temp_default_test", 0755)
	require.NoError(t, err)
	defer os.RemoveAll("temp_default_test")

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir("temp_default_test")

	// No test or spec directories - should default to RSpec
	detectedTask := DetectFramework()
	assert.Equal(t, "rspec", detectedTask.Name)
}

func TestExtractSuffixFromGlob(t *testing.T) {
	tests := []struct {
		name     string
		glob     string
		expected string
	}{
		{
			name:     "RSpec pattern",
			glob:     "spec/**/*_spec.rb",
			expected: "_spec.rb",
		},
		{
			name:     "Minitest pattern",
			glob:     "test/**/*_test.rb",
			expected: "_test.rb",
		},
		{
			name:     "Custom JS pattern",
			glob:     "tests/**/*_custom.js",
			expected: "_custom.js",
		},
		{
			name:     "Simple spec pattern",
			glob:     "spec/*_spec.rb",
			expected: "_spec.rb",
		},
		{
			name:     "No wildcards with spec",
			glob:     "spec/user_spec.rb",
			expected: "_spec.rb",
		},
		{
			name:     "No wildcards with test",
			glob:     "test/user_test.rb",
			expected: "_test.rb",
		},
		{
			name:     "No valid suffix",
			glob:     "src/**/*.rb",
			expected: "",
		},
		{
			name:     "Empty pattern",
			glob:     "",
			expected: "",
		},
		{
			name:     "Python test pattern",
			glob:     "test/**/*_test.py",
			expected: "_test.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSuffixFromGlob(tt.glob)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTestSuffix_WithGlob(t *testing.T) {
	tests := []struct {
		name     string
		task     *Task
		expected string
	}{
		{
			name: "RSpec task with glob",
			task: &Task{
				Name:     "rspec",
				TestGlob: "spec/**/*_spec.rb",
			},
			expected: "_spec.rb",
		},
		{
			name: "Minitest task with glob",
			task: &Task{
				Name:     "minitest",
				TestGlob: "test/**/*_test.rb",
			},
			expected: "_test.rb",
		},
		{
			name: "Custom task with glob",
			task: &Task{
				Name:     "custom",
				TestGlob: "tests/**/*_custom.js",
			},
			expected: "_custom.js",
		},
		{
			name: "Task without glob returns empty",
			task: &Task{
				Name: "minitest",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.GetTestSuffix()
			assert.Equal(t, tt.expected, result)
		})
	}
}
