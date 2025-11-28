package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/types"
)

// WorkerResult represents the accumulated results from a worker executing one or more test files
type WorkerResult struct {
	State        types.TestState
	Output       string
	Error        error
	Duration     time.Duration
	FileLoadTime time.Duration
	ExampleCount int
	FailureCount int
	PendingCount int
	Tests        []types.TestCaseNotification // All test notifications

	// Formatted output from RSpec
	FormattedFailures string
	FormattedSummary  string
}

// Success returns true if the test execution was successful (no failures or errors)
func (r WorkerResult) Success() bool {
	return r.State == types.StateSuccess
}

type OutputMessage struct {
	WorkerID int
	Type     string // "dot", "failure", "pending", "error", "stderr"
	Content  string
}

// GetWorkerCount determines the number of workers to use based on CLI, env, and defaults
func GetWorkerCount(cliWorkers int) int {
	if cliWorkers > 0 {
		return cliWorkers
	}

	if envVar := os.Getenv("PARALLEL_TEST_PROCESSORS"); envVar != "" {
		if count, err := strconv.Atoi(envVar); err == nil && count > 0 {
			return count
		}
	}

	// Default: cores minus 2, minimum 1
	workers := runtime.NumCPU() - 2
	if workers < 1 {
		workers = 1
	}
	return workers
}

// GetTestEnvNumber returns the TEST_ENV_NUMBER for a given worker index
func GetTestEnvNumber(workerIndex int, config *config.GlobalConfig) string {
	if config.FirstIs1 {
		return fmt.Sprintf("%d", workerIndex+1)
	}

	// Legacy behavior: first worker gets "", others get 2,3,4...
	if workerIndex == 0 {
		return ""
	}
	return fmt.Sprintf("%d", workerIndex+1)
}

// ANSI color codes
const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// Pre-compiled output strings to avoid repeated concatenation
var (
	greenDot   = []byte(colorGreen + "." + colorReset)
	redF       = []byte(colorRed + "F" + colorReset)
	yellowStar = []byte(colorYellow + "*" + colorReset)
	plainDot   = []byte(".")
	plainF     = []byte("F")
	plainStar  = []byte("*")
)

// outputAggregator handles all output from workers to avoid lock contention
func outputAggregator(outputChan <-chan OutputMessage, colorOutput bool) {
	for msg := range outputChan {
		switch msg.Type {
		case "dot":
			if colorOutput {
				os.Stdout.Write(greenDot)
			} else {
				os.Stdout.Write(plainDot)
			}
		case "failure":
			if colorOutput {
				os.Stdout.Write(redF)
			} else {
				os.Stdout.Write(plainF)
			}
		case "pending":
			if colorOutput {
				os.Stdout.Write(yellowStar)
			} else {
				os.Stdout.Write(plainStar)
			}
		case "stderr":
			fmt.Fprintln(os.Stderr, msg.Content)
		case "error":
			// For JSON parse errors or other output
			fmt.Fprintln(os.Stderr, msg.Content)
		}
	}
}

func errorResult(err error, start time.Time) WorkerResult {
	errorOutput := ""
	if err != nil {
		errorOutput = fmt.Sprintf("Error: %v\n", err)
	}

	return WorkerResult{
		State:    types.StateError,
		Output:   errorOutput,
		Error:    err,
		Duration: time.Since(start),
	}
}

// insertBeforeFiles inserts additional arguments before the file arguments in a command
// This is used to add formatter and color flags for RSpec before the spec file paths
func insertBeforeFiles(args []string, files []string, newArgs ...string) []string {
	filesStart := -1
	for i, arg := range args {
		for _, file := range files {
			if arg == file {
				filesStart = i
				break
			}
		}
		if filesStart != -1 {
			break
		}
	}

	// If we didn't find files, just append new args
	if filesStart == -1 {
		return append(args, newArgs...)
	}

	// Insert new args before files
	result := make([]string, 0, len(args)+len(newArgs))
	result = append(result, args[:filesStart]...)
	result = append(result, newArgs...)
	result = append(result, args[filesStart:]...)
	return result
}

// buildRSpecCommand builds an RSpec command with framework-specific flags
// Adds formatter (if available) and color flags before the file arguments
func buildRSpecCommand(j job.Job, files []string, globalConfig *config.GlobalConfig) []string {
	args := job.BuildJobCmd(j, files)

	// Add formatter if available
	formatterPath := globalConfig.ConfigPaths.GetJSONRowsFormatterPath()
	if formatterPath != "" {
		args = insertBeforeFiles(args, files, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")
	}

	if !globalConfig.ColorOutput {
		args = insertBeforeFiles(args, files, "--no-color")
	} else {
		args = insertBeforeFiles(args, files, "--force-color", "--tty")
	}

	return args
}

// buildMinitestCommand builds a Minitest command with framework-specific handling
// For multiple files, uses -e option "execute given ruby code" - we use this to require
// all necessary test files.
// For single file, uses BuildJobCmd directly
// TODO - this should probably use `job.BuildJobCmd` instead of building the command manually...
// but running multiple minitest files using just the stock lib is hard
func buildMinitestCommand(j job.Job, files []string, _globalConfig *config.GlobalConfig) []string {
	if len(files) > 1 {
		cmd := []string{"bundle", "exec", "ruby", "-Itest"}
		requires := make([]string, 0, len(files))
		for _, file := range files {
			// Strip the "test/" prefix if present since we're using -Itest, and strip the .rb extension
			testFile := strings.TrimPrefix(file, "test/")
			testFile = strings.TrimSuffix(testFile, ".rb")
			requires = append(requires, testFile)
		}

		// Create the require pattern
		requireList := `"` + strings.Join(requires, `", "`) + `"`
		cmd = append(cmd, "-e", `[`+requireList+`].each { |f| require f }`)
		return cmd
	}

	// For single file, use BuildJobCmd directly
	return job.BuildJobCmd(j, files)
}
