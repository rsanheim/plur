//go:build linux && amd64

package main

import "embed"

// Embed only this platform's watcher binary to keep plur small.
//
//go:embed embedded/watcher/watcher-x86_64-unknown-linux-gnu
var watcherBinaries embed.FS
