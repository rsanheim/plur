package watch

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu      sync.Mutex
	delay   time.Duration
	timer   *time.Timer
	pending map[string]bool // Track pending files
}

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

	for _, file := range files {
		d.pending[file] = true
	}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()

		pendingFiles := make([]string, 0, len(d.pending))
		for file := range d.pending {
			pendingFiles = append(pendingFiles, file)
		}

		d.pending = make(map[string]bool)

		d.mu.Unlock()

		if len(pendingFiles) > 0 {
			fn(pendingFiles)
		}
	})
}
