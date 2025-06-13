package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
)

func runDoctor(ctx *cli.Context) error {
	fmt.Println("Rux Doctor")
	fmt.Println("==========")
	fmt.Println()

	// Rux version info
	fmt.Printf("Rux Version:     %s\n", GetVersionInfo())
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
	fmt.Printf("Rux Binary:       %s\n", exePath)
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
	watcherPath, err := getWatcherBinaryPath()
	if err != nil {
		fmt.Printf("  Status:         Not available (%v)\n", err)
		fmt.Printf("  Platform:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	} else {
		fmt.Printf("  Status:         Available\n")
		fmt.Printf("  Binary Path:    %s\n", watcherPath)

		// Try to get watcher version
		watcherVersion, err := getCommandOutput(watcherPath, "--version")
		if err != nil {
			// e-dant/watcher doesn't have --version, so check if binary exists
			if _, statErr := os.Stat(watcherPath); statErr == nil {
				fmt.Printf("  Version:        Binary exists (no version info available)\n")
			} else {
				fmt.Printf("  Version:        error: %v\n", err)
			}
		} else {
			fmt.Printf("  Version:        %s\n", strings.TrimSpace(watcherVersion))
		}
		fmt.Printf("  Platform:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}
	fmt.Println()

	// Cache info
	cacheDir := ruxConfig.ConfigPaths.CacheDir
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
