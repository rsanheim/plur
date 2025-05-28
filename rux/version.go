package main

import (
	"runtime/debug"
)

// Build variables set by ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

// GetVersion returns the version string
func GetVersion() string {
	return version
}

// GetVersionInfo returns the full version information
func GetVersionInfo() string {
	// Try to get module version from runtime if not set by ldflags
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "(devel)" && info.Main.Version != "" {
				version = info.Main.Version
			}
		}
	}

	// Return just the version string
	return version
}
