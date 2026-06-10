//go:build darwin && arm64

package main

import "embed"

// Embed only this platform's watcher binary to keep plur small.
//
//go:embed embedded/watcher/watcher-aarch64-apple-darwin
var watcherBinaries embed.FS
