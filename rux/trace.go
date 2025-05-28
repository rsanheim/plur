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
	eventChan chan TraceEvent
	done      chan bool
	wg        sync.WaitGroup
}

var globalTracer = &Tracer{}

// InitTracer initializes the global tracer
func InitTracer(enabled bool) error {
	globalTracer.enabled = enabled
	if !enabled {
		return nil
	}

	// Create trace file in repo tmp directory with timestamp
	// Try to find repo root by looking for Rakefile (rux-meta root marker)
	repoRoot := "."
	currentDir, _ := os.Getwd()
	
	// Walk up directory tree looking for Rakefile
	for dir := currentDir; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "Rakefile")); err == nil {
			repoRoot = dir
			break
		}
	}
	
	traceDir := filepath.Join(repoRoot, "tmp", "rux-traces")
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
	globalTracer.eventChan = make(chan TraceEvent, 1000) // Buffered channel
	globalTracer.done = make(chan bool)
	
	// Write opening bracket for JSON array
	if _, err := file.WriteString("[\n"); err != nil {
		return err
	}

	// Start async writer goroutine
	globalTracer.wg.Add(1)
	go globalTracer.asyncWriter()

	fmt.Fprintf(os.Stderr, "Tracing enabled, writing to: %s\n", traceFile)
	return nil
}

// asyncWriter writes trace events asynchronously
func (t *Tracer) asyncWriter() {
	defer t.wg.Done()
	
	firstEvent := true
	for {
		select {
		case event := <-t.eventChan:
			// Marshal event to JSON
			data, err := json.MarshalIndent(event, "  ", "  ")
			if err != nil {
				continue
			}
			
			// If not the first event, add a comma
			if !firstEvent {
				t.file.WriteString(",\n")
			}
			firstEvent = false
			
			// Write event
			t.file.Write(data)
			
		case <-t.done:
			// Drain any remaining events
			for len(t.eventChan) > 0 {
				event := <-t.eventChan
				data, _ := json.MarshalIndent(event, "  ", "  ")
				if !firstEvent {
					t.file.WriteString(",\n")
				}
				firstEvent = false
				t.file.Write(data)
			}
			return
		}
	}
}

// CloseTracer closes the trace file
func CloseTracer() error {
	if !globalTracer.enabled || globalTracer.file == nil {
		return nil
	}

	// Signal the writer to stop
	close(globalTracer.done)
	
	// Wait for writer to finish
	globalTracer.wg.Wait()

	// Write closing bracket
	if _, err := globalTracer.file.WriteString("\n]"); err != nil {
		return err
	}

	// Final sync and close
	globalTracer.file.Sync()
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
	if !t.enabled || t.eventChan == nil {
		return
	}

	// Send event to async writer (non-blocking)
	select {
	case t.eventChan <- event:
		// Event sent successfully
	default:
		// Channel full, drop event rather than block
		// This ensures tracing never slows down the main execution
	}
}

// GetTraceFilePath returns the path to the current trace file
func GetTraceFilePath() string {
	if globalTracer.file != nil {
		return globalTracer.file.Name()
	}
	return ""
}