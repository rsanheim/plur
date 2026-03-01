package watch

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebouncer_BasicDebounce(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)

	var calls [][]string
	var mu sync.Mutex

	// Call debounce multiple times rapidly
	d.Debounce([]string{"a.rb"}, func(files []string) {
		mu.Lock()
		calls = append(calls, files)
		mu.Unlock()
	})
	d.Debounce([]string{"b.rb"}, func(files []string) {
		mu.Lock()
		calls = append(calls, files)
		mu.Unlock()
	})
	d.Debounce([]string{"c.rb"}, func(files []string) {
		mu.Lock()
		calls = append(calls, files)
		mu.Unlock()
	})

	// Wait for debounce to fire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 1, "Should only call callback once")
	assert.ElementsMatch(t, []string{"a.rb", "b.rb", "c.rb"}, calls[0])
}

func TestDebouncer_FileDeduplication(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)

	var receivedFiles []string
	var mu sync.Mutex

	// Add the same file multiple times
	d.Debounce([]string{"user.rb"}, func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	})
	d.Debounce([]string{"user.rb"}, func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	})
	d.Debounce([]string{"user.rb"}, func(files []string) {
		mu.Lock()
		receivedFiles = files
		mu.Unlock()
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, receivedFiles, 1, "Duplicate files should be deduplicated")
	assert.Equal(t, "user.rb", receivedFiles[0])
}

func TestDebouncer_TimerReset(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)

	var callTime time.Time
	var mu sync.Mutex
	startTime := time.Now()

	// First call
	d.Debounce([]string{"a.rb"}, func(files []string) {
		mu.Lock()
		callTime = time.Now()
		mu.Unlock()
	})

	// Wait a bit, then call again (should reset timer)
	time.Sleep(30 * time.Millisecond)
	d.Debounce([]string{"b.rb"}, func(files []string) {
		mu.Lock()
		callTime = time.Now()
		mu.Unlock()
	})

	// Wait for debounce to fire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	elapsed := callTime.Sub(startTime)
	mu.Unlock()

	// Should have waited ~80ms total (30ms + 50ms delay), not 50ms
	assert.Greater(t, elapsed.Milliseconds(), int64(70), "Timer should have been reset")
}

func TestDebouncer_NoCallbackIfEmpty(t *testing.T) {
	d := NewDebouncer(20 * time.Millisecond)

	callCount := 0
	var mu sync.Mutex

	// Call with empty slice
	d.Debounce([]string{}, func(files []string) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 0, callCount, "Should not call callback when no files pending")
}

func TestDebouncer_MultipleDelays(t *testing.T) {
	// Test that separate debouncers work independently
	d1 := NewDebouncer(30 * time.Millisecond)
	d2 := NewDebouncer(60 * time.Millisecond)

	var d1Called, d2Called bool
	var mu sync.Mutex

	d1.Debounce([]string{"a.rb"}, func(files []string) {
		mu.Lock()
		d1Called = true
		mu.Unlock()
	})
	d2.Debounce([]string{"b.rb"}, func(files []string) {
		mu.Lock()
		d2Called = true
		mu.Unlock()
	})

	// After 45ms, d1 should have fired, d2 should not
	time.Sleep(45 * time.Millisecond)
	mu.Lock()
	assert.True(t, d1Called, "d1 should have fired")
	assert.False(t, d2Called, "d2 should not have fired yet")
	mu.Unlock()

	// After another 30ms, d2 should have fired
	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	assert.True(t, d2Called, "d2 should have fired")
	mu.Unlock()
}
