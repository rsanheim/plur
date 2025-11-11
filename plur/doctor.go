package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/watch"
)

func runDoctorWithConfig(globalConfig *config.GlobalConfig) error {
	fmt.Println("Plur Doctor")
	fmt.Println("==========")
	fmt.Println()

	// Plur version info
	fmt.Printf("Plur Version:    %s\n", GetVersionInfo())
	fmt.Printf("Build Date:      %s\n", date)
	fmt.Printf("Git Commit:      %s\n", commit)
	fmt.Printf("Built By:        %s\n", builtBy)
	fmt.Println()

	// System info
	fmt.Printf("Operating System: %s\n", runtime.GOOS)
	fmt.Printf("Architecture:     %s\n", runtime.GOARCH)
	fmt.Printf("CPU Count:        %d\n", runtime.NumCPU())
	fmt.Printf("Go Version:       %s\n", runtime.Version())
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
		fmt.Printf("  Platform:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	} else {
		fmt.Printf("  Status:         Available\n")
		fmt.Printf("  Binary Path:    %s\n", watcherPath)

		// Try to get e-dant watcher version
		edantWatcherVersion, err := getCommandOutput(watcherPath, "--version")
		if err != nil {
			// e-dant/watcher doesn't have --version, so check if binary exists
			if _, statErr := os.Stat(watcherPath); statErr == nil {
				fmt.Printf("  Version:        Binary exists (no version info available)\n")
			} else {
				fmt.Printf("  Version:        error: %v\n", err)
			}
		} else {
			fmt.Printf("  Version:        %s\n", strings.TrimSpace(edantWatcherVersion))
		}
		fmt.Printf("  Platform:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}
	fmt.Println()

	// Cache info
	cacheDir := globalConfig.ConfigPaths.CacheDir
	fmt.Printf("Cache Directory:  %s\n", cacheDir)

	// Runtime data
	runtimePath, err := GetRuntimeFilePath()
	if err != nil {
		runtimePath = fmt.Sprintf("error: %v", err)
	}
	fmt.Printf("Runtime Data:     %s\n", runtimePath)

	// Check if runtime file exists
	if _, err := os.Stat(runtimePath); err == nil {
		fmt.Printf("                  (file exists)\n")
	} else {
		fmt.Printf("                  (file does not exist)\n")
	}
	fmt.Println()

	// Environment variables
	fmt.Println("Environment Variables:")
	fmt.Printf("  PARALLEL_TEST_PROCESSORS: %s\n", getEnvOrDefault("PARALLEL_TEST_PROCESSORS", "(not set)"))
	fmt.Printf("  FORCE_COLOR:              %s\n", getEnvOrDefault("FORCE_COLOR", "(not set)"))
	fmt.Printf("  NO_COLOR:                 %s\n", getEnvOrDefault("NO_COLOR", "(not set)"))
	fmt.Printf("  HOME:                     %s\n", getEnvOrDefault("HOME", "(not set)"))
	fmt.Printf("  GOPATH:                   %s\n", getEnvOrDefault("GOPATH", "(not set)"))
	fmt.Println()

	// Configuration
	fmt.Println("Configuration:")
	if err := checkConfiguration(globalConfig); err != nil {
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func checkConfiguration(globalConfig *config.GlobalConfig) error {
	// Show which config files are actually in use
	fmt.Println("  Active Configuration Files:")
	if len(globalConfig.LoadedConfigs) == 0 {
		fmt.Println("    Using defaults (no configuration files found)")
	} else {
		envFile := os.Getenv("PLUR_CONFIG_FILE")
		for i, loaded := range globalConfig.LoadedConfigs {
			// Check if this was from PLUR_CONFIG_FILE (first in list and env var is set)
			if i == 0 && envFile != "" {
				fmt.Printf("    - %s (via PLUR_CONFIG_FILE)\n", loaded)
			} else {
				fmt.Printf("    - %s\n", loaded)
			}
		}
	}

	// Show actual configuration values
	fmt.Println("\n  Active Settings:")
	fmt.Printf("    Workers:     %d\n", globalConfig.WorkerCount)
	fmt.Printf("    Color:       %v\n", globalConfig.ColorOutput)
	fmt.Printf("    Debug:       %v\n", globalConfig.Debug)
	fmt.Printf("    Verbose:     %v\n", globalConfig.Verbose)

	// Show active task if set
	// Note: We'd need to pass the active task name from PlurCLI if we want to show it here

	// Check for watch directories
	fmt.Println("\n  Watch Directories:")
	_, watchMappings := autodetect.GetAutodetectedDefaults()
	// Extract watch directories from watch mappings
	var watchDirs []string
	for _, mapping := range watchMappings {
		dir := mapping.SourceDir()
		if _, err := os.Stat(dir); err == nil {
			watchDirs = append(watchDirs, dir)
		}
	}
	sort.Strings(watchDirs)
	watchDirs = slices.Compact(watchDirs) // Remove duplicates from sorted slice
	if len(watchDirs) == 0 {
		fmt.Println("    Warning: No watch directories found in watch mappings")
	} else {
		for _, dir := range watchDirs {
			fmt.Printf("    %s/ (exists)\n", dir)
		}
	}

	return nil
}
