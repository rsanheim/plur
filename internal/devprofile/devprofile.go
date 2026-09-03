// Package devprofile writes Go runtime profiles for a plur process. It backs
// the hidden --dev-profile flag. Profiles land in a per-process directory so a
// watch reload, which execs a fresh process, never overwrites the previous set.
package devprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
)

var (
	dir     string
	cpuFile *os.File
)

// Start creates <root>/plur-<pid> and begins CPU profiling into it.
func Start(root string) error {
	dir = filepath.Join(root, fmt.Sprintf("plur-%d", os.Getpid()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("dev-profile: %w", err)
	}
	f, err := os.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		return fmt.Errorf("dev-profile: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("dev-profile: %w", err)
	}
	cpuFile = f
	return nil
}

// Stop ends CPU profiling and writes the goroutine leak, goroutine, and heap
// profiles. It is a no-op unless Start succeeded, and only acts once.
func Stop() {
	if cpuFile == nil {
		return
	}
	pprof.StopCPUProfile()
	cpuFile.Close()
	cpuFile = nil

	// Writing goroutineleak runs a GC cycle, so the heap profile written after
	// it reflects live objects only.
	write("goroutineleak.txt", "goroutineleak", 1)
	write("goroutine.txt", "goroutine", 2)
	write("heap.pprof", "heap", 0)
	fmt.Fprintf(os.Stderr, "[dev-profile] wrote profiles to %s\n", dir)
}

func write(name, profile string, debug int) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[dev-profile] %v\n", err)
		return
	}
	defer f.Close()
	if err := pprof.Lookup(profile).WriteTo(f, debug); err != nil {
		fmt.Fprintf(os.Stderr, "[dev-profile] %s: %v\n", name, err)
	}
}
