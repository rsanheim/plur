package watch

import (
	"sync"
	"time"
)

// Debouncer helps prevent multiple rapid executions
type Debouncer struct {
	mu       sync.Mutex
	delay    time.Duration
	timer    *time.Timer
	pending  map[string]bool // Track pending files
}

// NewDebouncer creates a new debouncer with the specified delay
func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay:   delay,
		pending: make(map[string]bool),
	}
}

// Debounce calls the function after the delay, resetting if called again
func (d *Debouncer) Debounce(files []string, fn func([]string)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add files to pending set
	for _, file := range files {
		d.pending[file] = true
	}

	// Cancel existing timer
	if d.timer != nil {
		d.timer.Stop()
	}

	// Start new timer
	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		
		// Get all pending files
		pendingFiles := make([]string, 0, len(d.pending))
		for file := range d.pending {
			pendingFiles = append(pendingFiles, file)
		}
		
		// Clear pending set
		d.pending = make(map[string]bool)
		
		d.mu.Unlock()

		// Execute function with all pending files
		if len(pendingFiles) > 0 {
			fn(pendingFiles)
		}
	})
}