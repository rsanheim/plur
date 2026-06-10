//go:build !(darwin && arm64) && !(linux && amd64) && !(linux && arm64) && !(windows && amd64)

// Package embedded carries platform assets compiled into the plur binary.
// No watcher binary exists for this platform; watch install reports the
// unsupported platform via getPlatformBinaryName.
package embedded

import "embed"

var Watcher embed.FS
