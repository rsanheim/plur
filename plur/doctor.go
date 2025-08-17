package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/watch"
)

func runDoctorWithConfig(globalConfig *config.GlobalConfig) error {
	fmt.Println("Plur Doctor")
	fmt.Println("==========")
	fmt.Println()

	// Plur version info
	fmt.Printf("Plur Version:     %s\n", GetVersionInfo())
	fmt.Printf("Build Date:      %s\n", date)
	fmt.Printf("Git Commit:      %s\n", commit)
	fmt.Printf("Built By:        %s\n", builtBy)

	// CLI Framework info
	fmt.Printf("CLI Framework:   Kong\n")
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
	fmt.Printf("Plur Binary:       %s\n", exePath)
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
	if err := checkConfiguration(); err != nil {
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

func checkConfiguration() error {
	// Check for local .plur.toml
	localConfig := ".plur.toml"
	if configInfo, err := checkConfigFile(localConfig); err == nil {
		fmt.Printf("  Local Config:   %s\n", configInfo)
	} else {
		fmt.Printf("  Local Config:   %s (not found)\n", localConfig)
	}

	// Check for global ~/.plur.toml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	globalConfig := filepath.Join(homeDir, ".plur.toml")
	if configInfo, err := checkConfigFile(globalConfig); err == nil {
		fmt.Printf("  Global Config:  %s\n", configInfo)
	} else {
		fmt.Printf("  Global Config:  %s (not found)\n", globalConfig)
	}

	// Try to load and validate active configuration
	fmt.Println("\n  Active Configuration:")
	if err := validateActiveConfig(localConfig, globalConfig); err != nil {
		fmt.Printf("    Error: %v\n", err)
	}

	return nil
}

func checkConfigFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Try to parse the TOML file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("%s (exists but unreadable: %v)", path, err), nil
	}

	var tempCfg map[string]interface{}
	if err := toml.Unmarshal(data, &tempCfg); err != nil {
		return fmt.Sprintf("%s (invalid TOML: %v)", path, err), nil
	}

	return fmt.Sprintf("%s (valid, %d bytes)", path, info.Size()), nil
}

func validateActiveConfig(localPath, globalPath string) error {
	// Load config in precedence order
	var cfg map[string]interface{}
	var configSource string

	// Try local first
	if data, err := os.ReadFile(localPath); err == nil {
		if err := toml.Unmarshal(data, &cfg); err == nil {
			configSource = "local .plur.toml"
		}
	}

	// If no local config, try global
	if cfg == nil {
		if data, err := os.ReadFile(globalPath); err == nil {
			if err := toml.Unmarshal(data, &cfg); err == nil {
				configSource = "global ~/.plur.toml"
			}
		}
	}

	if cfg == nil {
		fmt.Printf("    Using defaults (no configuration files found)\n")
		return nil
	}

	fmt.Printf("    Source: %s\n", configSource)

	// Display key configuration values
	if command, ok := cfg["command"].(string); ok {
		fmt.Printf("    Command: %s\n", command)
	}

	if workers, ok := cfg["workers"].(int64); ok {
		fmt.Printf("    Workers: %d\n", workers)
	}

	if color, ok := cfg["color"].(bool); ok {
		fmt.Printf("    Color: %v\n", color)
	}

	// Check for command-specific configs
	if specConfig, ok := cfg["spec"].(map[string]interface{}); ok {
		fmt.Println("    [spec] section:")
		if specCommand, ok := specConfig["command"].(string); ok {
			fmt.Printf("      Command: %s\n", specCommand)
		}
	}

	if watchConfig, ok := cfg["watch"].(map[string]interface{}); ok {
		if runConfig, ok := watchConfig["run"].(map[string]interface{}); ok {
			fmt.Println("    [watch.run] section:")
			if watchCommand, ok := runConfig["command"].(string); ok {
				fmt.Printf("      Command: %s\n", watchCommand)
			}
			if debounce, ok := runConfig["debounce"].(int64); ok {
				fmt.Printf("      Debounce: %dms\n", debounce)
				if debounce <= 0 || debounce > 10000 {
					fmt.Printf("      Warning: debounce value %dms seems unusual (recommended: 50-500ms)\n", debounce)
				}
			}
		}
	}

	// Check for watch directories
	fmt.Println("\n  Watch Directories:")
	framework := config.DetectTestFramework()
	watchDirs := watch.GetWatchDirectories(string(framework))
	if len(watchDirs) == 0 {
		dirList := "spec/, lib/, app/"
		if framework == config.FrameworkMinitest {
			dirList = "test/, lib/, app/"
		}
		fmt.Printf("    Warning: No watch directories found (checked: %s)\n", dirList)
	} else {
		for _, dir := range watchDirs {
			fmt.Printf("    %s/ (exists)\n", dir)
		}
	}

	return nil
}
