package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONMessage represents a single JSON message from the streaming formatter
type JSONMessage struct {
	Type            string         `json:"type"`
	Count           int            `json:"count,omitempty"`
	LoadTime        float64        `json:"load_time,omitempty"`
	Description     string         `json:"description,omitempty"`
	FullDescription string         `json:"full_description,omitempty"`
	Location        string         `json:"location,omitempty"`
	FilePath        string         `json:"file_path,omitempty"`
	LineNumber      int            `json:"line_number,omitempty"`
	PendingMessage  string         `json:"pending_message,omitempty"`
	Exception       *JSONException `json:"exception,omitempty"`
}

// JSONException represents exception details in failed examples
type JSONException struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

// ParseJSONMessage parses a JSON line from the streaming formatter
func ParseJSONMessage(line string) (*JSONMessage, error) {
	// Remove the separator prefix
	separator := "RUX_JSON:"
	if !strings.HasPrefix(line, separator) {
		return nil, nil // Not a JSON message
	}

	jsonStr := strings.TrimPrefix(line, separator)

	var msg JSONMessage
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
	Examples     []JSONMessage
}

// AddExample adds an example result to the streaming results
func (sr *StreamingResults) AddExample(msg JSONMessage) {
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

// ConvertToRSpecJSON converts streaming results to the expected RSpecJSONOutput format
func (sr *StreamingResults) ConvertToRSpecJSON() *RSpecJSONOutput {
	output := &RSpecJSONOutput{
		Version:  "3.12.0", // Use a default version
		Examples: make([]RSpecExample, 0, len(sr.Examples)),
		Summary: RSpecSummary{
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

		example := RSpecExample{
			ID:              fmt.Sprintf("[%d:%d]", 1, msg.LineNumber), // Generate a simple ID
			Description:     msg.Description,
			FullDescription: msg.FullDescription,
			FilePath:        msg.FilePath,
			LineNumber:      msg.LineNumber,
			RunTime:         0, // Not provided by streaming formatter
		}

		switch msg.Type {
		case "example_passed":
			example.Status = "passed"
		case "example_failed":
			example.Status = "failed"
			if msg.Exception != nil {
				example.Exception = &RSpecException{
					Class:     msg.Exception.Class,
					Message:   msg.Exception.Message,
					Backtrace: msg.Exception.Backtrace,
				}
			}
		case "example_pending":
			example.Status = "pending"
			// Note: RSpecExample doesn't have a PendingMessage field
		}

		output.Examples = append(output.Examples, example)
	}

	return output
}
