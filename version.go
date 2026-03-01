package main

import (
	"fmt"
	"runtime/debug"
)

// Build variables set by ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

// GetVersionInfo returns the full version information
func GetVersionInfo() string {
	// If version was set by ldflags (GoReleaser), use it
	if version != "dev" {
		return version
	}

	// Try to get version info from runtime
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if this is a versioned module install (go install pkg@version)
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}

		// For dev builds, extract VCS information from BuildSettings
		var vcsRevision, vcsTime string
		var vcsModified bool

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				vcsRevision = setting.Value
				if len(vcsRevision) > 7 {
					commit = vcsRevision[:7] // Store short commit for compatibility
				}
			case "vcs.time":
				vcsTime = setting.Value
				if vcsTime != "" {
					date = vcsTime // Store for compatibility
				}
			case "vcs.modified":
				vcsModified = (setting.Value == "true")
			}
		}

		// Build version string from VCS info
		if vcsRevision != "" {
			shortCommit := vcsRevision
			if len(shortCommit) > 7 {
				shortCommit = shortCommit[:7]
			}

			// Use simple dev version format with commit hash
			versionStr := fmt.Sprintf("dev-%s", shortCommit)

			// Add dirty flag if working tree has modifications
			if vcsModified {
				versionStr += "-dirty"
			}

			return versionStr
		}
	}

	// Fallback to simple "dev"
	return version
}
