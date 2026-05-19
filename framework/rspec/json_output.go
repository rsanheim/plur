package rspec

// StreamingMessage represents a single PLUR_JSON line from the formatter
type StreamingMessage struct {
	Type            string         `json:"type"`
	Example         *StreamExample `json:"example,omitempty"`
	ExampleGroup    *ExampleGroup  `json:"group,omitempty"`
	Summary         *LoadSummary   `json:"summary,omitempty"`
	Message         string         `json:"message,omitempty"`
	FormattedOutput string         `json:"formatted_output,omitempty"`
	// dump_summary fields (flat at root level)
	ExampleCount int     `json:"example_count,omitempty"`
	FailureCount int     `json:"failure_count,omitempty"`
	PendingCount int     `json:"pending_count,omitempty"`
	ErrorCount   int     `json:"errors_outside_of_examples_count,omitempty"`
	Duration     float64 `json:"duration,omitempty"`
}

// ExampleGroup matches the group format from the Ruby formatter
type ExampleGroup struct {
	Description string `json:"description"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
}

// StreamExample matches the example format from the Ruby formatter
type StreamExample struct {
	ID                    string           `json:"id,omitempty"`
	Description           string           `json:"description"`
	FullDescription       string           `json:"full_description"`
	Location              string           `json:"location"`
	FilePath              string           `json:"file_path"`
	AbsoluteFilePath      string           `json:"absolute_file_path,omitempty"`
	LineNumber            int              `json:"line_number"`
	LocationRerunArgument string           `json:"location_rerun_argument,omitempty"`
	ScopedID              string           `json:"scoped_id,omitempty"`
	Status                string           `json:"status"`
	RunTime               float64          `json:"run_time"`
	PendingMessage        string           `json:"pending_message,omitempty"`
	Exception             *StreamException `json:"exception,omitempty"`
}

// StreamException matches the exception format from the Ruby formatter
type StreamException struct {
	Class     string   `json:"class"`
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

// LoadSummary represents the load_summary message's summary field
type LoadSummary struct {
	Count    int     `json:"count"`
	LoadTime float64 `json:"load_time"`
}
