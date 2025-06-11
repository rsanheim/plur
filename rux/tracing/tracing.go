// Package tracing provides performance tracing functionality using Go's runtime/trace.
// It creates trace files that can be viewed with `go tool trace`.
package tracing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"
	"time"
)

// Global tracer state
var (
	enabled   bool
	traceFile *os.File
)

// Init initializes the tracing system.
// If enabled is false, all tracing operations become no-ops.
// Trace files are stored in ~/.cache/rux/traces/
func Init(enable bool) error {
	enabled = enable
	if !enabled {
		return nil
	}

	// Create trace file in centralized ~/.cache/rux/traces directory
	timestamp := time.Now().Format("20060102-150405")
	
	// Get cache directory (same as formatter cache)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}
	
	cacheDir := filepath.Join(homeDir, ".cache", "rux")
	traceDir := filepath.Join(cacheDir, "traces")
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return fmt.Errorf("failed to create trace directory: %v", err)
	}
	
	traceFilePath := filepath.Join(traceDir, fmt.Sprintf("rux-trace-%s.trace", timestamp))

	file, err := os.Create(traceFilePath)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %v", err)
	}
	traceFile = file

	if err := trace.Start(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to start trace: %v", err)
	}


	fmt.Fprintf(os.Stderr, "Tracing enabled, writing to: %s\n", traceFilePath)
	fmt.Fprintf(os.Stderr, "View with: go tool trace %s\n", traceFilePath)
	return nil
}

// Close stops tracing and closes the trace file.
func Close() error {
	if !enabled || traceFile == nil {
		return nil
	}

	trace.Stop()
	return traceFile.Close()
}

// StartRegion starts a trace region and returns a function to end it.
// Usage: defer tracing.StartRegion(ctx, "operation_name")()
func StartRegion(ctx context.Context, name string) func() {
	if !enabled {
		return func() {}
	}
	region := trace.StartRegion(ctx, name)
	return region.End
}

// StartRegionWithWorker starts a trace region with worker context.
// Usage: defer tracing.StartRegionWithWorker(ctx, "operation", workerID, specFile)()
func StartRegionWithWorker(ctx context.Context, name string, workerID int, specFile string) func() {
	if !enabled {
		return func() {}
	}
	region := trace.StartRegion(ctx, name)
	trace.Logf(ctx, "worker", "worker_id=%d spec_file=%s", workerID, specFile)
	return region.End
}


// LogEvent logs a trace event with key-value pairs.
// Usage: tracing.LogEvent(ctx, "event_name", "key1", value1, "key2", value2)
func LogEvent(ctx context.Context, name string, keyvals ...interface{}) {
	if !enabled {
		return
	}
	
	// Build format string and args
	var keys []string
	var vals []interface{}
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			keys = append(keys, fmt.Sprintf("%v=%%v", keyvals[i]))
			vals = append(vals, keyvals[i+1])
		}
	}
	
	if len(keys) > 0 {
		format := strings.Join(keys, " ")
		trace.Logf(ctx, name, format, vals...)
	} else {
		trace.Log(ctx, name, "")
	}
}




// IsEnabled returns whether tracing is currently enabled.
func IsEnabled() bool {
	return enabled
}