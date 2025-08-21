package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime/debug"
	"strings"
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

		// For dev builds, extract VCS information
		var vcsRevision string
		var vcsModified bool
		
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				vcsRevision = setting.Value
				if len(vcsRevision) > 7 {
					commit = vcsRevision[:7] // Store short commit for compatibility
				}
			case "vcs.modified":
				vcsModified = (setting.Value == "true")
			}
		}

		// Build version string from VCS info
		if vcsRevision != "" {
			// Try to get a descriptive version using git describe
			versionStr := getGitDescribeVersion(vcsRevision[:7])
			
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

// getGitDescribeVersion attempts to get a descriptive version from git
func getGitDescribeVersion(shortCommit string) string {
	// Try to run git describe to get a nice version string
	cmd := exec.Command("git", "describe", "--tags", "--always", "--abbrev=7", "--match=v*")
	var out bytes.Buffer
	cmd.Stdout = &out
	
	if err := cmd.Run(); err == nil {
		// Got a git describe output like "v0.10.0-5-g3c0a135"
		described := strings.TrimSpace(out.String())
		if described != "" && described != shortCommit {
			// Check if we're exactly on a tag
			if !strings.Contains(described, "-g") {
				// We're on a tag, but building locally, so add -dev
				return described + "-dev"
			}
			// We have commits since tag
			return described
		}
	}
	
	// Fallback to simple format
	return fmt.Sprintf("dev-%s", shortCommit)
}
