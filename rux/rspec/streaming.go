package rspec

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StreamingMessage represents a single JSON message from the streaming formatter
type StreamingMessage struct {
	Type            string              `json:"type"`
	Count           int                 `json:"count,omitempty"`
	LoadTime        float64             `json:"load_time,omitempty"`
	Description     string              `json:"description,omitempty"`
	FullDescription string              `json:"full_description,omitempty"`
	Location        string              `json:"location,omitempty"`
	FilePath        string              `json:"file_path,omitempty"`
	LineNumber      int                 `json:"line_number,omitempty"`
	PendingMessage  string              `json:"pending_message,omitempty"`
	Exception       *StreamingException `json:"exception,omitempty"`
	Example         *StreamingExample   `json:"example,omitempty"`
}

// StreamingExample represents nested example data with runtime
type StreamingExample struct {
	Description     string              `json:"description"`
	FullDescription string              `json:"full_description"`
	Location        string              `json:"location"`
	FilePath        string              `json:"file_path"`
	LineNumber      int                 `json:"line_number"`
	Status          string              `json:"status"`
	RunTime         float64             `json:"run_time"`
	PendingMessage  string              `json:"pending_message,omitempty"`
	Exception       *StreamingException `json:"exception,omitempty"`
}

// StreamingException represents exception details in failed examples
type StreamingException struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

// ParseStreamingMessage parses a JSON line from the streaming formatter
func ParseStreamingMessage(line string) (*StreamingMessage, error) {
	// Remove the separator prefix
	separator := "RUX_JSON:"
	if !strings.HasPrefix(line, separator) {
		return nil, nil // Not a JSON message
	}

	jsonStr := strings.TrimPrefix(line, separator)

	var msg StreamingMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// StreamingResults accumulates results from streaming JSON messages
type StreamingResults struct {
	ExampleCount int
	FailureCount int
	PendingCount int
	LoadTime     float64
	Examples     []StreamingMessage
}

// AddExample adds an example result to the streaming results
func (sr *StreamingResults) AddExample(msg StreamingMessage) {
	sr.Examples = append(sr.Examples, msg)

	switch msg.Type {
	case "example_passed":
		sr.ExampleCount++
	case "example_failed":
		sr.ExampleCount++
		sr.FailureCount++
	case "example_pending":
		sr.ExampleCount++
		sr.PendingCount++
	}
}

// ConvertToJSONOutput converts streaming results to the expected JSONOutput format
func (sr *StreamingResults) ConvertToJSONOutput() *JSONOutput {
	output := &JSONOutput{
		Version:  "3.12.0", // Use a default version
		Examples: make([]Example, 0, len(sr.Examples)),
		Summary: Summary{
			ExampleCount: sr.ExampleCount,
			FailureCount: sr.FailureCount,
			PendingCount: sr.PendingCount,
			Duration:     sr.LoadTime, // RSpec uses Duration not LoadTime
		},
	}

	// Convert each example
	for _, msg := range sr.Examples {
		if msg.Type == "start" {
			// Update load time from start message
			output.Summary.Duration = msg.LoadTime
			continue
		}

		if msg.Type == "close" {
			continue
		}

		// Use example data from nested field if available, otherwise fall back to top-level fields
		var exampleData *StreamingExample
		if msg.Example != nil {
			exampleData = msg.Example
		} else {
			// Fallback for backwards compatibility
			exampleData = &StreamingExample{
				Description:     msg.Description,
				FullDescription: msg.FullDescription,
				FilePath:        msg.FilePath,
				LineNumber:      msg.LineNumber,
				Exception:       msg.Exception,
			}
		}

		example := Example{
			ID:              fmt.Sprintf("[%d:%d]", 1, exampleData.LineNumber), // Generate a simple ID
			Description:     exampleData.Description,
			FullDescription: exampleData.FullDescription,
			FilePath:        exampleData.FilePath,
			LineNumber:      exampleData.LineNumber,
			RunTime:         exampleData.RunTime,
		}

		switch msg.Type {
		case "example_passed":
			example.Status = "passed"
		case "example_failed":
			example.Status = "failed"
			if exampleData.Exception != nil {
				example.Exception = &Exception{
					Class:     exampleData.Exception.Class,
					Message:   exampleData.Exception.Message,
					Backtrace: exampleData.Exception.Backtrace,
				}
			}
		case "example_pending":
			example.Status = "pending"
			// Note: Example doesn't have a PendingMessage field
		}

		output.Examples = append(output.Examples, example)
	}

	return output
}
