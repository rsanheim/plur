package rspec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// JSONOutput represents the root structure of RSpec JSON output
type JSONOutput struct {
	Version  string    `json:"version"`
	Examples []Example `json:"examples"`
	Summary  Summary   `json:"summary"`
}

// Example represents a single test example in the JSON output
type Example struct {
	ID              string     `json:"id"`
	Description     string     `json:"description"`
	FullDescription string     `json:"full_description"`
	Status          string     `json:"status"`
	FilePath        string     `json:"file_path"`
	LineNumber      int        `json:"line_number"`
	RunTime         float64    `json:"run_time"`
	Exception       *Exception `json:"exception,omitempty"`
}

// Exception represents failure information
type Exception struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

// Summary represents the summary statistics
type Summary struct {
	Duration      float64 `json:"duration"`
	ExampleCount  int     `json:"example_count"`
	FailureCount  int     `json:"failure_count"`
	PendingCount  int     `json:"pending_count"`
	ErrorsOutside int     `json:"errors_outside_of_examples_count"`
}

// FailureDetail represents a formatted failure for display
type FailureDetail struct {
	Description string
	FilePath    string
	LineNumber  int
	ErrorClass  string
	Message     string
	Backtrace   []string
}

// ParseJSONOutput reads and parses RSpec JSON output from a file
func ParseJSONOutput(filename string) (*JSONOutput, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Check if file is empty or contains only whitespace
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("JSON file is empty - RSpec likely failed to start (check for missing gems or bundle install needed)")
	}

	var output JSONOutput
	if err := json.Unmarshal(data, &output); err != nil {
		// Provide more helpful error for common case of malformed JSON due to early RSpec failure
		if strings.Contains(err.Error(), "unexpected end of JSON input") {
			return nil, fmt.Errorf("JSON file is incomplete - RSpec may have failed before generating output (check for missing gems or run 'bundle install')")
		}
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &output, nil
}

// ExtractFailures extracts failure details from RSpec examples
func ExtractFailures(examples []Example) []FailureDetail {
	var failures []FailureDetail

	for _, example := range examples {
		if example.Status == "failed" && example.Exception != nil {
			failure := FailureDetail{
				Description: example.FullDescription,
				FilePath:    example.FilePath,
				LineNumber:  example.LineNumber,
				ErrorClass:  example.Exception.Class,
				Message:     example.Exception.Message,
				Backtrace:   example.Exception.Backtrace,
			}
			failures = append(failures, failure)
		}
	}

	return failures
}

// FormatFailure formats a single failure for display
//
// Manual string indentation is necessary here to match RSpec's exact output format.
// RSpec uses specific indentation levels for different parts of the failure output:
// - 2 spaces for the failure number and description
// - 5 spaces for "Failure/Error:" label
// - 7 spaces for error messages and expected/actual values
// This precise formatting ensures compatibility with tools that parse RSpec output.
func FormatFailure(index int, failure FailureDetail) string {
	var sb strings.Builder

	// Failure header
	sb.WriteString(fmt.Sprintf("  %d) %s\n", index, failure.Description))

	// Error/Failure line
	sb.WriteString("     Failure/Error: ")

	// Try to extract the failing line from the source file
	failingLine := ExtractFailingLine(failure.FilePath, failure.LineNumber)
	if failingLine != "" {
		sb.WriteString(failingLine)
		sb.WriteString("\n")
	} else {
		// If we can't read the file, just continue without the line
		sb.WriteString("\n")
	}

	// Error class and message
	if strings.Contains(failure.ErrorClass, "ExpectationNotMetError") {
		// For expectation failures, the message is already formatted with proper indentation
		lines := strings.Split(strings.TrimSpace(failure.Message), "\n")
		for _, line := range lines {
			if line != "" {
				sb.WriteString("       " + line + "\n")
			}
		}
	} else {
		// For other errors, show the error class
		sb.WriteString(fmt.Sprintf("\n     %s:\n", failure.ErrorClass))
		sb.WriteString(fmt.Sprintf("       %s\n", failure.Message))
	}

	// Backtrace (first line only, like RSpec does)
	if len(failure.Backtrace) > 0 {
		sb.WriteString(fmt.Sprintf("     # %s", failure.Backtrace[0]))
	}

	return sb.String()
}

// ExtractFailingLine reads the specified line from a file
//
// This manual extraction is necessary because RSpec's JSON output only provides
// the line number where the test is defined, not the actual failing assertion.
// We need to read the source file and find the likely failing line within the test body.
// This is a heuristic approach that looks for common patterns like 'expect', 'raise', etc.
func ExtractFailingLine(filePath string, lineNumber int) string {
	// For RSpec failures, we want to extract the actual failing line from within the test
	// The lineNumber points to the test definition, but the error is usually inside
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentLine := 1
	var insideTest bool
	var failingLine string

	for scanner.Scan() {
		line := scanner.Text()

		// If we're at the test definition line, start looking inside
		if currentLine == lineNumber {
			insideTest = true
		}

		// If we're inside the test, look for expect() or raise statements
		if insideTest && currentLine > lineNumber {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "expect") ||
				strings.Contains(trimmed, "raise") ||
				strings.Contains(trimmed, ".") && !strings.HasPrefix(trimmed, "it") && !strings.HasPrefix(trimmed, "end") {
				failingLine = trimmed
				break
			}
			// Stop if we hit the end of the test
			if trimmed == "end" {
				break
			}
		}

		currentLine++
	}

	return failingLine
}

// FormatFailedExamples formats the list of failed examples for the summary
func FormatFailedExamples(failures []FailureDetail) string {
	var sb strings.Builder

	for _, failure := range failures {
		sb.WriteString(fmt.Sprintf("rspec %s:%d # %s\n",
			failure.FilePath,
			failure.LineNumber,
			failure.Description))
	}

	return sb.String()
}
