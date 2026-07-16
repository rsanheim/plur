package main

import (
	"fmt"
	"os"
	"os/exec"
	stdruntime "runtime"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/internal/testruntime"
	"github.com/rsanheim/plur/watch"
)

// runtimeStats reads the runtime cache file at path and returns a one-line
// summary suitable for the plur doctor "Runtime Data:" block. Falls back to
// the original "(file exists)" wording if the cache is unreadable or in an
// unexpected shape, so doctor never crashes on a malformed file.
func runtimeStats(path string, size int64) string {
	cache := testruntime.LoadCache(path)
	if cache == nil || len(cache.Files) == 0 {
		return "(file exists)"
	}
	var examples int
	for _, f := range cache.Files {
		examples += len(f.Examples)
	}
	return fmt.Sprintf("%s / %d files / %d examples", humanSize(size), len(cache.Files), examples)
}

// humanSize formats a byte count as KB/MB rounded to nearest unit for the
// doctor output. Matches the style of common CLI utilities (du, ls -h).
func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.0fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0fK", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%dB", n)
}

func runDoctorWithConfig(globalConfig *config.GlobalConfig, runtimeConfig *runtime.RuntimeConfig) error {
	fmt.Println("Plur Doctor")
	fmt.Println("==========")
	fmt.Println()

	// Plur version info
	fmt.Printf("Plur Version:    %s\n", buildinfo.GetVersionInfo())
	fmt.Printf("Build Date:      %s\n", buildinfo.Date)
	fmt.Printf("Git Commit:      %s\n", buildinfo.Commit)
	fmt.Printf("Built By:        %s\n", buildinfo.BuiltBy)
	fmt.Printf("Race Detector:   %v\n", buildinfo.RaceEnabled)
	fmt.Println()

	// System info
	fmt.Printf("Operating System: %s\n", stdruntime.GOOS)
	fmt.Printf("Architecture:     %s\n", stdruntime.GOARCH)
	fmt.Printf("CPU Count:        %d\n", stdruntime.NumCPU())
	fmt.Printf("Go Version:       %s\n", stdruntime.Version())
	fmt.Println()

	// Working directory
	pwd, err := os.Getwd()
	if err != nil {
		pwd = fmt.Sprintf("error: %v", err)
	}
	fmt.Printf("Working Dir:      %s\n", pwd)

	// Binary location
	exePath, err := os.Executable()
	if err != nil {
		exePath = fmt.Sprintf("error: %v", err)
	}
	fmt.Printf("Plur Binary:     %s\n", exePath)
	fmt.Println()

	// Ruby info
	fmt.Println("Ruby Environment:")
	rubyVersion, err := getCommandOutput("ruby", "--version")
	if err != nil {
		rubyVersion = fmt.Sprintf("error: %v", err)
	}
	fmt.Printf("  Ruby Version:   %s\n", strings.TrimSpace(rubyVersion))

	bundlerVersion, err := getCommandOutput("bundle", "--version")
	if err != nil {
		bundlerVersion = fmt.Sprintf("error: %v", err)
	}
	fmt.Printf("  Bundler:        %s\n", strings.TrimSpace(bundlerVersion))

	rspecVersion, err := getCommandOutput("bundle", "exec", "rspec", "--version")
	if err != nil {
		rspecVersion = "not found"
	}
	fmt.Printf("  RSpec:          %s\n", strings.TrimSpace(rspecVersion))
	fmt.Println()

	// Watcher info
	fmt.Println("File Watcher:")
	watcherPath, err := watch.GetWatcherBinaryPath(globalConfig.ConfigPaths.BinDir)
	if err != nil {
		fmt.Printf("  Status:         Not available (%v)\n", err)
		fmt.Printf("  Platform:       %s/%s\n", stdruntime.GOOS, stdruntime.GOARCH)
	} else {
		fmt.Printf("  Status:         Available\n")
		fmt.Printf("  Binary Path:    %s\n", watcherPath)

		// Get e-dant watcher version (supported since 0.13.7)
		edantWatcherVersion, err := getCommandOutput(watcherPath, "-v")
		if err != nil {
			fmt.Printf("  Version:        error: %v\n", err)
		} else {
			fmt.Printf("  Version:        %s\n", strings.TrimSpace(edantWatcherVersion))
		}
		fmt.Printf("  Platform:       %s/%s\n", stdruntime.GOOS, stdruntime.GOARCH)
	}
	fmt.Println()

	// Cache info
	cacheDir := globalConfig.ConfigPaths.CacheDir
	fmt.Printf("Cache Directory:  %s\n", cacheDir)

	// Runtime data
	var runtimePath string
	rt, err := testruntime.NewRuntimeTracker(globalConfig.RuntimeDir)
	if err != nil {
		runtimePath = fmt.Sprintf("error: %v", err)
	} else {
		runtimePath = rt.RuntimeFilePath()
	}
	fmt.Printf("Runtime Data:     %s\n", runtimePath)

	// Check if runtime file exists; on hit, show size / files / examples.
	if info, err := os.Stat(runtimePath); err == nil {
		fmt.Printf("                  %s\n", runtimeStats(runtimePath, info.Size()))
	} else {
		fmt.Printf("                  (file does not exist)\n")
	}
	fmt.Println()

	// Environment variables: the always-shown set, then any other PLUR_*
	// vars present in the environment.
	fmt.Println("Environment Variables:")
	important := []string{"PLUR_WORKERS", "PARALLEL_TEST_PROCESSORS", "NO_COLOR", "HOME", "GOPATH"}
	keys := slices.Clone(important)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "PLUR_") && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys[len(important):])
	for _, key := range keys {
		fmt.Printf("  %-25s %s\n", key+":", envDisplay(key))
	}
	fmt.Println()

	// Configuration
	fmt.Println("Configuration:")
	if err := checkConfiguration(globalConfig, runtimeConfig); err != nil {
		fmt.Printf("  Error checking configuration: %v\n", err)
	}

	return nil
}

func getCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// envDisplay shows an env var's value as a quoted Go literal, so empty and
// whitespace-only values are visible and distinct from unset.
func envDisplay(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "(not set)"
	}
	return strconv.Quote(value)
}

func checkConfiguration(globalConfig *config.GlobalConfig, runtimeConfig *runtime.RuntimeConfig) error {
	// Show which config files are actually in use
	fmt.Println("  Active Configuration Files:")
	if len(globalConfig.LoadedConfigs) == 0 {
		fmt.Println("    Using defaults (no configuration files found)")
	} else {
		envFile := os.Getenv("PLUR_CONFIG_FILE")
		for _, loaded := range globalConfig.LoadedConfigs {
			// Check if this was from PLUR_CONFIG_FILE
			if envFile != "" && loaded == kong.ExpandPath(envFile) {
				fmt.Printf("    - %s (via PLUR_CONFIG_FILE)\n", loaded)
			} else {
				fmt.Printf("    - %s\n", loaded)
			}
		}
	}

	// Show actual configuration values
	fmt.Println("\n  Active Settings:")
	fmt.Printf("    Workers:     %d\n", globalConfig.WorkerCount)
	fmt.Printf("    Color:       %v (%s)\n", globalConfig.ColorOutput, globalConfig.ColorSource)
	fmt.Printf("    Debug:       %v\n", globalConfig.Debug)
	fmt.Printf("    Verbose:     %v\n", globalConfig.Verbose)

	// Job resolution - use runtimeConfig
	fmt.Println("\n  Job Resolution:")
	selected, err := runtime.SelectJobFromRuntimeConfig(runtimeConfig, nil)
	if err != nil {
		fmt.Printf("    %v\n", err)
	} else {
		fmt.Printf("    Active Job:      %s\n", selected.Name)
		fmt.Printf("    Command:         %v\n", selected.Job.Cmd)
		patterns, _ := selected.Job.TargetPatterns()
		fmt.Printf("    Target Patterns: %s\n", strings.Join(patterns, ", "))
	}

	// Watch directories - use runtimeConfig.Watches
	fmt.Println("\n  Watch Directories:")
	if len(runtimeConfig.Watches) == 0 {
		fmt.Println("    Warning: No watch mappings available")
		return nil
	}
	var watchDirs []string
	for _, mapping := range runtimeConfig.Watches {
		dir := mapping.SourceDir()
		if _, err := os.Stat(dir); err == nil {
			watchDirs = append(watchDirs, dir)
		}
	}
	sort.Strings(watchDirs)
	watchDirs = slices.Compact(watchDirs)
	if len(watchDirs) == 0 {
		fmt.Println("    Warning: No watch directories found in watch mappings")
		return nil
	}
	for _, dir := range watchDirs {
		fmt.Printf("    %s/ (exists)\n", dir)
	}
	return nil
}
