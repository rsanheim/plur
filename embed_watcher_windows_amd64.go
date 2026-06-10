//go:build windows && amd64

package main

import "embed"

// Embed only this platform's watcher binary to keep plur small.
//
//go:embed embedded/watcher/watcher-x86_64-pc-windows-msvc
var watcherBinaries embed.FS
