//go:build !(darwin && arm64) && !(linux && amd64) && !(linux && arm64) && !(windows && amd64)

package main

import "embed"

// No watcher binary exists for this platform; watch install reports the
// unsupported platform via getPlatformBinaryName.
var watcherBinaries embed.FS
