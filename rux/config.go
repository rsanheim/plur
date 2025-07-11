package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rsanheim/rux/rspec"
)

// TestFramework represents the test framework type
type TestFramework string

const (
	FrameworkRSpec    TestFramework = "rspec"
	FrameworkMinitest TestFramework = "minitest"
)

// GlobalConfig holds settings that are truly global across all commands
type GlobalConfig struct {
	Auto        bool
	ColorOutput bool
	ConfigPaths *ConfigPaths
	Debug       bool
	Verbose     bool
	DryRun      bool
	WorkerCount int
	RuntimeDir  string
	JSON        string // JSON output file
}

type ConfigPaths struct {
	RuxHome           string // ~/.rux or $RUX_HOME
	BinDir            string
	CacheDir          string
	RuntimeDir        string
	FormatterDir      string
	JSONRowsFormatter string
}

func InitConfigPaths() *ConfigPaths {
	ruxHome, ok := os.LookupEnv("RUX_HOME")
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot find home directory and RUX_HOME not set: %v\n", err)
			os.Exit(1)
		}
		ruxHome = filepath.Join(homeDir, ".rux")
	}

	err := os.MkdirAll(ruxHome, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create RUX_HOME directory: %v\n", err)
		os.Exit(1)
	}

	binDir := filepath.Join(ruxHome, "bin")
	cacheDir := filepath.Join(ruxHome, "cache")
	runtimeDir := filepath.Join(ruxHome, "runtime")
	formatterDir := filepath.Join(ruxHome, "formatter")

	paths := []string{binDir, cacheDir, runtimeDir, formatterDir}
	for _, path := range paths {
		if os.MkdirAll(path, 0755) != nil {
			fmt.Fprintf(os.Stderr, "failed to create %s directory: %v\n", path, err)
			os.Exit(1)
		}
	}

	jsonRowsFormatter, err := rspec.GetFormatterPath(formatterDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get JSON rows formatter path: %v\n", err)
		os.Exit(1)
	}

	configPaths := ConfigPaths{
		RuxHome:           ruxHome,
		BinDir:            binDir,
		CacheDir:          cacheDir,
		RuntimeDir:        runtimeDir,
		FormatterDir:      formatterDir,
		JSONRowsFormatter: jsonRowsFormatter,
	}

	return &configPaths
}

// ParseFrameworkType converts a string type to TestFramework enum
func ParseFrameworkType(frameworkType string) TestFramework {
	if frameworkType == "" {
		return DetectTestFramework()
	}
	switch frameworkType {
	case "rspec":
		return FrameworkRSpec
	case "minitest":
		return FrameworkMinitest
	default:
		// Default to RSpec for backward compatibility
		return FrameworkRSpec
	}
}

// DetectTestFramework attempts to detect the test framework based on directory structure
func DetectTestFramework() TestFramework {
	// Check for test/ directory (minitest)
	if _, err := os.Stat("test"); err == nil {
		return FrameworkMinitest
	}

	// Check for spec/ directory (rspec)
	if _, err := os.Stat("spec"); err == nil {
		return FrameworkRSpec
	}

	// Default to RSpec for backward compatibility
	return FrameworkRSpec
}
