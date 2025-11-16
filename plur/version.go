package main

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"
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

// GetDetailedVersionInfo returns detailed version information for debugging
func GetDetailedVersionInfo() string {
	var parts []string

	// Start with basic version
	parts = append(parts, fmt.Sprintf("Version: %s", GetVersionInfo()))

	// Add build info if available
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.GoVersion != "" {
			parts = append(parts, fmt.Sprintf("Go: %s", info.GoVersion))
		}

		// Extract VCS details
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				parts = append(parts, fmt.Sprintf("Commit: %s", setting.Value))
			case "vcs.time":
				parts = append(parts, fmt.Sprintf("Built: %s", setting.Value))
			case "vcs.modified":
				if setting.Value == "true" {
					parts = append(parts, "Modified: true")
				}
			}
		}
	}

	// Add ldflags info if set by GoReleaser
	if builtBy != "unknown" {
		parts = append(parts, fmt.Sprintf("Built by: %s", builtBy))
	}

	return strings.Join(parts, "\n")
}

// GetBuildTime returns the build time if available
func GetBuildTime() string {
	// First check if set by ldflags
	if date != "unknown" && date != "" {
		return date
	}

	// Try to get from VCS info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				// Parse and format the time for display
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					return t.Format("2006-01-02 15:04:05")
				}
				return setting.Value
			}
		}
	}

	return "unknown"
}
