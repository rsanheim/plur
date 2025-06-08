package watch

import (
	"fmt"
	"sync"
	"time"
)

// ManagerConfig holds configuration for the watcher manager
type ManagerConfig struct {
	Directories    []string
	DebounceDelay  time.Duration
	TimeoutSeconds int
}

// WatcherManager manages multiple watcher processes
type WatcherManager struct {
	watchers   []*Watcher
	eventChan  chan Event
	errorChan  chan error
	stopChan   chan struct{}
	config     *ManagerConfig
	binaryPath string
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewWatcherManager creates a new watcher manager instance
func NewWatcherManager(config *ManagerConfig, binaryPath string) *WatcherManager {
	return &WatcherManager{
		config:     config,
		binaryPath: binaryPath,
		eventChan:  make(chan Event, 100),
		errorChan:  make(chan error, 10),
		stopChan:   make(chan struct{}),
		watchers:   make([]*Watcher, 0, len(config.Directories)),
	}
}

// Start begins watching all configured directories
func (wm *WatcherManager) Start() error {
	// For each directory, create and start a watcher
	for _, dir := range wm.config.Directories {
		// Create a config for single directory
		singleDirConfig := &WatcherConfig{
			Directory:      dir,
			DebounceDelay:  wm.config.DebounceDelay,
			TimeoutSeconds: wm.config.TimeoutSeconds,
		}

		watcher := NewWatcher(singleDirConfig, wm.binaryPath)
		if err := watcher.Start(); err != nil {
			// Clean up already started watchers
			wm.cleanup()
			return fmt.Errorf("failed to start watcher for %s: %w", dir, err)
		}

		wm.mu.Lock()
		wm.watchers = append(wm.watchers, watcher)
		wm.mu.Unlock()

		// Aggregate events from this watcher
		wm.wg.Add(1)
		go wm.aggregateEvents(watcher)
	}

	return nil
}

// Stop stops all watchers
func (wm *WatcherManager) Stop() {
	// Signal all goroutines to stop
	close(wm.stopChan)

	// Clean up all watchers
	wm.cleanup()

	// Wait for all goroutines to finish
	wm.wg.Wait()

	// Close channels
	close(wm.eventChan)
	close(wm.errorChan)
}

// Events returns the aggregated event channel
func (wm *WatcherManager) Events() <-chan Event {
	return wm.eventChan
}

// Errors returns the aggregated error channel
func (wm *WatcherManager) Errors() <-chan error {
	return wm.errorChan
}

// aggregateEvents collects events from a single watcher
func (wm *WatcherManager) aggregateEvents(w *Watcher) {
	defer wm.wg.Done()

	for {
		select {
		case event := <-w.Events():
			select {
			case wm.eventChan <- event:
			case <-wm.stopChan:
				return
			}
		case err := <-w.Errors():
			select {
			case wm.errorChan <- err:
			case <-wm.stopChan:
				return
			}
		case <-wm.stopChan:
			return
		}
	}
}

// cleanup stops all running watchers
func (wm *WatcherManager) cleanup() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, watcher := range wm.watchers {
		watcher.Stop()
	}
	wm.watchers = nil
}
