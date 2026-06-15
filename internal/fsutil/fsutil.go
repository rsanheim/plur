package fsutil

import "os"

// IgnoredDirs are directory names that test detection and file discovery never
// descend into. They hold third-party or generated code whose test files must
// not drive detection or be handed to workers, and they dominate walk time when
// present. This is the single source of truth shared by the runtime detection
// walk and the fileset discovery filter.
var IgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"tmp":          true,
}

// FileExists checks if a file exists (not a directory)
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
