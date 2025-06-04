package watch

import (
	"os"
	"time"
)

// DefaultDebounceDelay is the default debounce delay
const DefaultDebounceDelay = 100 * time.Millisecond

// GetWatchDirectories determines which directories to watch based on what exists
func GetWatchDirectories() []string {
	dirs := []string{}

	// Always watch spec directory if it exists
	if _, err := os.Stat("spec"); err == nil {
		dirs = append(dirs, "spec")
	}

	// Watch lib directory if it exists
	if _, err := os.Stat("lib"); err == nil {
		dirs = append(dirs, "lib")
	}

	// Watch app directory if it exists (Rails apps)
	if _, err := os.Stat("app"); err == nil {
		dirs = append(dirs, "app")
	}

	return dirs
}
