package task

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
)

// MappingRule defines how source files map to test files
type MappingRule struct {
	Pattern string `toml:"pattern"` // Source file glob pattern
	Target  string `toml:"target"`  // Target pattern with {{path}}, {{name}}, {{file}} tokens
}

// Task defines how to run tests, linters, or other jobs in a project
type Task struct {
	Name           string        `toml:"-"`               // Task name (e.g., "rspec", "minitest")
	Description    string        `toml:"description"`     // Human-readable description
	Run            string        `toml:"run"`             // Command to run (e.g., "bundle exec rspec")
	SourceDirs     []string      `toml:"source_dirs"`     // Directories to watch/search
	Mappings       []MappingRule `toml:"mappings"`        // File mapping rules
	IgnorePatterns []string      `toml:"ignore_patterns"` // Patterns to ignore (for watch)
	TestGlob       string        `toml:"test_glob"`       // Glob pattern for test files (e.g., "spec/**/*_spec.rb")
}

// GetFramework returns the TestFramework enum for this task
func (t *Task) GetFramework() config.TestFramework {
	switch t.Name {
	case "minitest":
		return config.FrameworkMinitest
	case "rspec":
		return config.FrameworkRSpec
	default:
		return config.FrameworkRSpec // Default fallback
	}
}

// BuildCommand constructs the command to execute for this task
func (t *Task) BuildCommand(files []string, globalConfig *config.GlobalConfig, commandOverride string) []string {
	// Use override command if provided
	command := t.Run
	if commandOverride != "" {
		command = commandOverride
	}

	// Parse command into executable and base args
	parts := strings.Fields(command)
	if len(parts) == 0 {
		// Special case for minitest where we build the command
		if t.Name == "minitest" {
			return t.buildMinitestCommand(files, globalConfig)
		} else {
			return nil
		}
	}

	args := parts

	// Add framework-specific arguments
	if t.Name == "rspec" && globalConfig != nil {
		args = t.addRSpecArgs(args, globalConfig)
	}

	// Add files to command
	args = append(args, files...)

	return args
}

// GetTestSuffix returns the test file suffix derived from TestGlob
func (t *Task) GetTestSuffix() string {
	return extractSuffixFromGlob(t.TestGlob)
}

// GetTestPattern returns the glob pattern for this task
func (t *Task) GetTestPattern() string {
	return t.TestGlob
}

// extractSuffixFromGlob extracts the file suffix from a glob pattern
// Examples: "spec/**/*_spec.rb" -> "_spec.rb", "test/**/*_test.rb" -> "_test.rb"
func extractSuffixFromGlob(glob string) string {
	// Find the last * in the pattern
	lastStar := strings.LastIndex(glob, "*")
	if lastStar == -1 {
		// No wildcards, try to extract from end
		if strings.Contains(glob, "_spec.") {
			// Find _spec. and take everything from there
			if idx := strings.Index(glob, "_spec."); idx != -1 {
				return glob[idx:]
			}
		}
		if strings.Contains(glob, "_test.") {
			// Find _test. and take everything from there
			if idx := strings.Index(glob, "_test."); idx != -1 {
				return glob[idx:]
			}
		}
		return ""
	}

	// Get everything after the last *
	suffix := glob[lastStar+1:]

	// If it looks like a valid test suffix, return it
	// Look for patterns like _spec.rb, _test.rb, _custom.js, etc.
	if strings.Contains(suffix, "_") && strings.Contains(suffix, ".") {
		// Find the underscore and check if it looks like a test pattern
		underscoreIdx := strings.Index(suffix, "_")
		if underscoreIdx != -1 {
			return suffix
		}
	}

	return ""
}

// buildMinitestCommand builds the minitest command
func (t *Task) buildMinitestCommand(files []string, globalConfig *config.GlobalConfig) []string {
	// Build base command
	cmd := []string{"bundle", "exec", "ruby", "-Itest"}

	// Handle files differently based on count
	if len(files) == 1 {
		// Single file: pass directly
		cmd = append(cmd, files[0])
	} else if len(files) > 1 {
		// Multiple files: use -e with require pattern
		requires := make([]string, len(files))
		for i, file := range files {
			// Strip the "test/" prefix if present since we're using -Itest
			testFile := strings.TrimPrefix(file, "test/")
			// Remove the .rb extension for require
			testFile = strings.TrimSuffix(testFile, ".rb")
			requires[i] = strings.ReplaceAll(testFile, "\"", "\\\"") // Escape quotes
		}

		// Create the require pattern
		requireList := `"` + strings.Join(requires, `", "`) + `"`
		cmd = append(cmd, "-e", `[`+requireList+`].each { |f| require f }`)
	}

	return cmd
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
		// Try each mapping rule in order
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

// NewRSpecTask creates the default RSpec task configuration
func NewRSpecTask() *Task {
	return &Task{
		Name:        "rspec",
		Description: "Run RSpec specs",
		Run:         "bundle exec rspec",
		SourceDirs:  []string{"spec", "lib", "app"},
		Mappings: []MappingRule{
			{
				Pattern: "lib/**/*.rb",
				Target:  "spec/{{path}}/{{name}}_spec.rb",
			},
			{
				Pattern: "app/**/*.rb",
				Target:  "spec/{{path}}/{{name}}_spec.rb",
			},
			{
				Pattern: "spec/**/*_spec.rb",
				Target:  "{{file}}",
			},
		},
		IgnorePatterns: []string{".git", "tmp", "log"},
		TestGlob:       "spec/**/*_spec.rb",
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
				Pattern: "lib/**/*.rb",
				Target:  "test/{{path}}/{{name}}_test.rb",
			},
			{
				Pattern: "app/**/*.rb",
				Target:  "test/{{path}}/{{name}}_test.rb",
			},
			{
				Pattern: "test/**/*_test.rb",
				Target:  "{{file}}",
			},
		},
		IgnorePatterns: []string{".git", "tmp", "log"},
		TestGlob:       "test/**/*_test.rb",
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
