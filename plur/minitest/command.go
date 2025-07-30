package minitest

import (
	"fmt"
	"strings"
)

// BuildCommand creates the minitest command for running test files
// We mostly follow how parallel_tests runs minitests in parallel via
// the pattern: ruby -Itest -e "require files"
func BuildCommand(files []string, options BuildOptions) []string {
	var cmd []string
	if options.UseBundler {
		cmd = []string{"bundle", "exec", "ruby", "-Itest"}
	} else {
		cmd = []string{"ruby", "-Itest"}
	}

	if options.Verbose {
		cmd = append(cmd, "-v")
	}

	// Add the requires if necessary...
	if len(files) == 1 { // single file
		cmd = append(cmd, files[0])
	} else {
		// Multiple files: use -e with require
		// We need to require the files using their path relative to the test directory
		requires := make([]string, len(files))
		for i, file := range files {
			// Strip the "test/" prefix if present since we're using -Itest
			testFile := strings.TrimPrefix(file, "test/")
			// Remove the .rb extension for require
			testFile = strings.TrimSuffix(testFile, ".rb")
			requires[i] = fmt.Sprintf("%q", testFile)
		}
		requireList := strings.Join(requires, ", ")
		cmd = append(cmd, "-e", fmt.Sprintf("[%s].each { |f| require f }", requireList))
	}

	// Add any additional test options
	if len(options.TestOptions) > 0 {
		cmd = append(cmd, "--")
		cmd = append(cmd, options.TestOptions...)
	}

	return cmd
}

// BuildOptions contains options for building the minitest command
type BuildOptions struct {
	Verbose     bool     // Add -v flag for verbose output
	TestOptions []string // Additional options to pass to minitest
	UseBundler  bool     // Whether to use bundle exec
}
