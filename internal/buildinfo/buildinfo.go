package buildinfo

import (
	"fmt"
	"runtime/debug"
)

// Build variables set by ldflags
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// GetVersionInfo returns the full version information
func GetVersionInfo() string {
	// If version was set by ldflags (GoReleaser), use it
	if Version != "dev" {
		return Version
	}

	// Try to get version info from runtime
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if this is a versioned module install (go install pkg@version)
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}

		// For dev builds, extract VCS information from BuildSettings
		var vcsRevision string
		var vcsModified bool

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				vcsRevision = setting.Value
			case "vcs.time":
				if setting.Value != "" {
					Date = setting.Value
				}
			case "vcs.modified":
				vcsModified = (setting.Value == "true")
			}
		}

		if vcsRevision != "" {
			shortCommit := vcsRevision
			if len(shortCommit) > 7 {
				shortCommit = shortCommit[:7]
			}
			Commit = shortCommit

			versionStr := fmt.Sprintf("dev-%s", shortCommit)
			if vcsModified {
				versionStr += "-dirty"
			}
			return versionStr
		}
	}

	// Fallback to simple "dev"
	return Version
}
