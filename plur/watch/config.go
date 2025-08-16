package watch

import (
	"os"
	"time"
)

// DefaultDebounceDelay is the default debounce delay
const DefaultDebounceDelay = 100 * time.Millisecond

// GetWatchDirectories determines which directories to watch based on framework
func GetWatchDirectories(framework string) []string {
	dirs := []string{}

	// Watch test directory based on framework
	switch framework {
	case "minitest":
		if _, err := os.Stat("test"); err == nil {
			dirs = append(dirs, "test")
		}
	case "rspec":
		if _, err := os.Stat("spec"); err == nil {
			dirs = append(dirs, "spec")
		}
	default:
		// Auto-detect: watch both if they exist
		if _, err := os.Stat("spec"); err == nil {
			dirs = append(dirs, "spec")
		}
		if _, err := os.Stat("test"); err == nil {
			dirs = append(dirs, "test")
		}
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
