package embedded

import (
	_ "embed"
	"strings"
)

//go:embed watcher.version
var watcherVersion string

// WatcherVersion returns the version of the embedded watcher binaries.
func WatcherVersion() string {
	return strings.TrimSpace(watcherVersion)
}
