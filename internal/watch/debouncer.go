package watch

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu      sync.Mutex
	delay   time.Duration
	timer   *time.Timer
	pending []string
	seen    map[string]struct{}
}

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
		seen:  make(map[string]struct{}),
	}
}

// Debounce calls the function after the delay, resetting if called again
func (d *Debouncer) Debounce(files []string, fn func(TargetSet)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, file := range files {
		if _, exists := d.seen[file]; exists {
			continue
		}
		d.seen[file] = struct{}{}
		d.pending = append(d.pending, file)
	}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()

		pending := NewTargetSet(d.pending...)
		d.pending = nil
		d.seen = make(map[string]struct{})

		d.mu.Unlock()

		if pending.Len() > 0 {
			fn(pending)
		}
	})
}
