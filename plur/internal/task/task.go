package task

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/minitest"
	"github.com/rsanheim/plur/rspec"
	"github.com/rsanheim/plur/types"
)

// Task defines how to run tests, linters, or other jobs in a project
type Task struct {
	Name        string   `toml:"-"`           // Task name (e.g., "rspec", "minitest")
	Description string   `toml:"description"` // Human-readable description
	Run         string   `toml:"run"`         // Command to run (e.g., "bundle exec rspec")
	SourceDirs  []string `toml:"source_dirs"` // Directories to watch/search
	TestGlob    string   `toml:"test_glob"`   // Glob pattern for test files (e.g., "spec/**/*_spec.rb")
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

// CreateParser creates the appropriate test output parser for this task
func (t *Task) CreateParser() (types.TestOutputParser, error) {
	switch t.Name {
	case "rspec":
		return rspec.NewOutputParser(), nil
	case "minitest":
		return minitest.NewOutputParser(), nil
	default:
		return nil, fmt.Errorf("unsupported task type: %s", t.Name)
	}
}

// IsMinitestStyle returns true if this task is minitest-style (for formatting decisions)
func (t *Task) IsMinitestStyle() bool {
	return t.Name == "minitest"
}

// GetWatchDirs returns the directories that should be watched for this task
func (t *Task) GetWatchDirs() []string {
	var dirs []string

	// Add directories from SourceDirs that actually exist
	for _, dir := range t.SourceDirs {
		if exists(dir) {
			dirs = append(dirs, dir)
		}
	}

	return dirs
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

// NewRSpecTask creates the default RSpec task configuration
func NewRSpecTask() *Task {
	return &Task{
		Name:        "rspec",
		Description: "Run RSpec specs",
		Run:         "bundle exec rspec",
		SourceDirs:  []string{"spec", "lib", "app"},
		TestGlob:    "spec/**/*_spec.rb",
	}
}

// NewMinitestTask creates the default Minitest task configuration
func NewMinitestTask() *Task {
	return &Task{
		Name:        "minitest",
		Description: "Run Minitest tests",
		Run:         "", // Special handling in BuildCommand
		SourceDirs:  []string{"test", "lib", "app"},
		TestGlob:    "test/**/*_test.rb",
	}
}

// DetectFramework returns the appropriate task based on directory structure
func DetectFramework() *Task {
	if exists("spec") {
		return NewRSpecTask()
	}

	if exists("test") {
		return NewMinitestTask()
	}

	return NewRSpecTask()
}

// BothFrameworksExist returns true if both spec/ and test/ directories exist
func BothFrameworksExist() bool {
	return exists("spec") && exists("test")
}

// exists checks if a path exists
func exists(path string) bool {
	matches, err := filepath.Glob(path)
	return err == nil && len(matches) > 0
}
