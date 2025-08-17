package task

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
)

// MappingRule defines how source files map to test files
type MappingRule struct {
	Pattern  string `toml:"pattern"`  // Source file glob pattern
	Target   string `toml:"target"`   // Target pattern with {{path}}, {{name}}, {{file}} tokens
	Priority int    `toml:"priority"` // Higher priority rules are checked first
}

// Task defines how to run tests, linters, or other jobs in a project
type Task struct {
	Name           string        `toml:"-"`               // Task name (e.g., "rspec", "minitest")
	Description    string        `toml:"description"`     // Human-readable description
	Run            string        `toml:"run"`             // Command to run (e.g., "bundle exec rspec")
	SourceDirs     []string      `toml:"source_dirs"`     // Directories to watch/search
	Mappings       []MappingRule `toml:"mappings"`        // File mapping rules
	IgnorePatterns []string      `toml:"ignore_patterns"` // Patterns to ignore (for watch)
}

// BuildCommand constructs the command to execute for this task
func (t *Task) BuildCommand(files []string, globalConfig *config.GlobalConfig, taskOverride *Task) *exec.Cmd {
	// Use override command if provided
	command := t.Run
	if taskOverride != nil && taskOverride.Run != "" {
		command = taskOverride.Run
	}

	// Parse command into executable and base args
	parts := strings.Fields(command)
	if len(parts) == 0 {
		// Special case for minitest where we build the command
		if t.Name == "minitest" {
			parts = t.buildMinitestCommand(files, globalConfig)
		} else {
			return nil
		}
	}

	cmd := parts[0]
	args := parts[1:]

	// Add framework-specific arguments
	if t.Name == "rspec" && globalConfig != nil {
		args = t.addRSpecArgs(args, globalConfig)
	}

	// Add files to command
	args = append(args, files...)

	return exec.Command(cmd, args...)
}

// buildMinitestCommand builds the minitest command
func (t *Task) buildMinitestCommand(files []string, globalConfig *config.GlobalConfig) []string {
	// Default minitest command building
	parts := []string{"bundle", "exec", "ruby", "-Itest"}

	// Add files with -r flag for each file
	for _, file := range files {
		parts = append(parts, "-r", file)
	}

	// Add the test runner
	parts = append(parts, "-e", "nil")

	return parts
}

// addRSpecArgs adds RSpec-specific arguments
func (t *Task) addRSpecArgs(args []string, globalConfig *config.GlobalConfig) []string {
	// Add formatter if available
	if globalConfig.ConfigPaths != nil {
		formatterPath := globalConfig.ConfigPaths.GetJSONRowsFormatterPath()
		if formatterPath != "" {
			args = append(args, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")
		}
	}

	// Handle color output
	if !globalConfig.ColorOutput {
		args = append(args, "--no-color")
	} else {
		args = append(args, "--force-color", "--tty")
	}

	return args
}

// MapFilesToTarget maps source files to their corresponding test files
func (t *Task) MapFilesToTarget(sourceFiles []string) []string {
	var targetFiles []string
	seen := make(map[string]bool)

	for _, sourceFile := range sourceFiles {
		// Try each mapping rule in priority order
		for _, mapping := range t.Mappings {
			// Check if source file matches the pattern
			matched, err := doublestar.Match(mapping.Pattern, sourceFile)
			if err != nil || !matched {
				continue
			}

			// Apply the mapping transformation
			target := t.applyMapping(sourceFile, mapping.Target)
			if target != "" && !seen[target] {
				seen[target] = true
				targetFiles = append(targetFiles, target)
			}
			break // Use first matching rule
		}
	}

	return targetFiles
}

// applyMapping transforms a source file path using a target pattern
func (t *Task) applyMapping(sourceFile, targetPattern string) string {
	// Extract components from source file
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Remove leading directory if it matches a source dir
	path := dir
	for _, sourceDir := range t.SourceDirs {
		if strings.HasPrefix(dir, sourceDir+"/") {
			path = strings.TrimPrefix(dir, sourceDir+"/")
			break
		} else if dir == sourceDir {
			path = ""
			break
		}
	}

	// Replace tokens in target pattern
	result := targetPattern
	result = strings.ReplaceAll(result, "{{file}}", sourceFile)
	result = strings.ReplaceAll(result, "{{path}}", path)
	result = strings.ReplaceAll(result, "{{name}}", name)

	// Clean up any double slashes or leading/trailing slashes
	result = filepath.Clean(result)

	return result
}

// GetTestPattern returns the glob pattern for finding test files
func (t *Task) GetTestPattern() string {
	switch t.Name {
	case "rspec":
		return "spec/**/*_spec.rb"
	case "minitest":
		return "test/**/*_test.rb"
	default:
		// Try to infer from mappings
		for _, mapping := range t.Mappings {
			if strings.Contains(mapping.Target, "_spec.rb") {
				return "spec/**/*_spec.rb"
			}
			if strings.Contains(mapping.Target, "_test.rb") {
				return "test/**/*_test.rb"
			}
		}
		return "**/*_test.rb"
	}
}

// GetTestSuffix returns the test file suffix for this task
func (t *Task) GetTestSuffix() string {
	switch t.Name {
	case "rspec":
		return "_spec.rb"
	case "minitest":
		return "_test.rb"
	default:
		// Try to infer from mappings
		for _, mapping := range t.Mappings {
			if strings.Contains(mapping.Target, "_spec.rb") {
				return "_spec.rb"
			}
			if strings.Contains(mapping.Target, "_test.rb") {
				return "_test.rb"
			}
		}
		return "_test.rb"
	}
}

// NewRSpecTask creates the default RSpec task configuration
func NewRSpecTask() *Task {
	return &Task{
		Name:        "rspec",
		Description: "Run RSpec specs",
		Run:         "bundle exec rspec",
		SourceDirs:  []string{"spec", "lib", "app"},
		Mappings: []MappingRule{
			{
				Pattern:  "lib/**/*.rb",
				Target:   "spec/{{path}}/{{name}}_spec.rb",
				Priority: 100,
			},
			{
				Pattern:  "app/**/*.rb",
				Target:   "spec/{{path}}/{{name}}_spec.rb",
				Priority: 90,
			},
			{
				Pattern:  "spec/**/*_spec.rb",
				Target:   "{{file}}",
				Priority: 80,
			},
		},
		IgnorePatterns: []string{".git", "tmp", "log"},
	}
}

// NewMinitestTask creates the default Minitest task configuration
func NewMinitestTask() *Task {
	return &Task{
		Name:        "minitest",
		Description: "Run Minitest tests",
		Run:         "", // Special handling in BuildCommand
		SourceDirs:  []string{"test", "lib", "app"},
		Mappings: []MappingRule{
			{
				Pattern:  "lib/**/*.rb",
				Target:   "test/{{path}}/{{name}}_test.rb",
				Priority: 100,
			},
			{
				Pattern:  "app/**/*.rb",
				Target:   "test/{{path}}/{{name}}_test.rb",
				Priority: 90,
			},
			{
				Pattern:  "test/**/*_test.rb",
				Target:   "{{file}}",
				Priority: 80,
			},
		},
		IgnorePatterns: []string{".git", "tmp", "log"},
	}
}

// DetectFramework returns the appropriate task based on directory structure
func DetectFramework() *Task {
	// Check for test/ directory first (minitest)
	if exists("test") {
		return NewMinitestTask()
	}

	// Check for spec/ directory (rspec)
	if exists("spec") {
		return NewRSpecTask()
	}

	// Default to RSpec for backward compatibility
	return NewRSpecTask()
}

// exists checks if a path exists
func exists(path string) bool {
	matches, err := filepath.Glob(path)
	return err == nil && len(matches) > 0
}
