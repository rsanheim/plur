package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TraceEvent represents a single trace event
type TraceEvent struct {
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration_ms"`
	WorkerID  int       `json:"worker_id,omitempty"`
	SpecFile  string    `json:"spec_file,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Tracer manages trace events
type Tracer struct {
	enabled bool
	events  []TraceEvent
	mu      sync.Mutex
	file    *os.File
}

var globalTracer = &Tracer{}

// InitTracer initializes the global tracer
func InitTracer(enabled bool) error {
	globalTracer.enabled = enabled
	if !enabled {
		return nil
	}

	// Create trace file in temp directory with timestamp
	traceDir := filepath.Join(os.TempDir(), "rux-traces")
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return fmt.Errorf("failed to create trace directory: %v", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	traceFile := filepath.Join(traceDir, fmt.Sprintf("rux-trace-%s.json", timestamp))
	
	file, err := os.Create(traceFile)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %v", err)
	}

	globalTracer.file = file
	
	// Write opening bracket for JSON array
	if _, err := file.WriteString("[\n"); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Tracing enabled, writing to: %s\n", traceFile)
	return nil
}

// CloseTracer closes the trace file
func CloseTracer() error {
	if !globalTracer.enabled || globalTracer.file == nil {
		return nil
	}

	// Write closing bracket
	if _, err := globalTracer.file.WriteString("\n]"); err != nil {
		return err
	}

	return globalTracer.file.Close()
}

// TraceFunc traces a function execution
func TraceFunc(name string) func() {
	if !globalTracer.enabled {
		return func() {}
	}

	start := time.Now()
	
	return func() {
		end := time.Now()
		duration := end.Sub(start).Seconds() * 1000 // Convert to milliseconds
		
		event := TraceEvent{
			Name:      name,
			StartTime: start,
			EndTime:   end,
			Duration:  duration,
		}
		
		globalTracer.recordEvent(event)
	}
}

// TraceFuncWithWorker traces a function execution with worker context
func TraceFuncWithWorker(name string, workerID int, specFile string) func() {
	if !globalTracer.enabled {
		return func() {}
	}

	start := time.Now()
	
	return func() {
		end := time.Now()
		duration := end.Sub(start).Seconds() * 1000 // Convert to milliseconds
		
		event := TraceEvent{
			Name:      name,
			StartTime: start,
			EndTime:   end,
			Duration:  duration,
			WorkerID:  workerID,
			SpecFile:  specFile,
		}
		
		globalTracer.recordEvent(event)
	}
}

// TraceFuncWithMetadata traces a function execution with additional metadata
func TraceFuncWithMetadata(name string, metadata map[string]interface{}) func() {
	if !globalTracer.enabled {
		return func() {}
	}

	start := time.Now()
	
	return func() {
		end := time.Now()
		duration := end.Sub(start).Seconds() * 1000 // Convert to milliseconds
		
		event := TraceEvent{
			Name:      name,
			StartTime: start,
			EndTime:   end,
			Duration:  duration,
			Metadata:  metadata,
		}
		
		globalTracer.recordEvent(event)
	}
}

// recordEvent records a trace event to the file
func (t *Tracer) recordEvent(event TraceEvent) {
	if !t.enabled || t.file == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Marshal event to JSON
	data, err := json.MarshalIndent(event, "  ", "  ")
	if err != nil {
		return
	}

	// If not the first event, add a comma
	if len(t.events) > 0 {
		if _, err := t.file.WriteString(",\n"); err != nil {
			return
		}
	}

	// Write event
	if _, err := t.file.Write(data); err != nil {
		return
	}

	// Flush to ensure data is written
	t.file.Sync()

	t.events = append(t.events, event)
}

// GetTraceFilePath returns the path to the current trace file
func GetTraceFilePath() string {
	if globalTracer.file != nil {
		return globalTracer.file.Name()
	}
	return ""
}