//go:build linux && arm64

// Package embedded carries platform assets compiled into the plur binary.
// Each platform embeds only its own watcher binary to keep plur small.
package embedded

import "embed"

//go:embed watcher/watcher-aarch64-unknown-linux-gnu
var Watcher embed.FS
