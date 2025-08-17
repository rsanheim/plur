package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rsanheim/plur/rspec"
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
	FirstIs1    bool   // Start TEST_ENV_NUMBER at 1 instead of empty string
}

// IsSerial returns true if running in serial mode (single worker)
func (c *GlobalConfig) IsSerial() bool {
	return c.WorkerCount == 1
}

type ConfigPaths struct {
	PlurHome          string // ~/.plur or $PLUR_HOME
	BinDir            string
	CacheDir          string
	RuntimeDir        string
	FormatterDir      string
	JSONRowsFormatter string
}

// InitConfigPaths initializes PLUR_HOME if necessary, as well as subdirs inside it.
// By default this will be ~/.plur unless PLUR_HOME is set by the user.
func InitConfigPaths() *ConfigPaths {
	plurHome, ok := os.LookupEnv("PLUR_HOME")
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: cannot find home directory and PLUR_HOME not set: %v\n", err)
			os.Exit(1)
		}
		plurHome = filepath.Join(homeDir, ".plur")
	}

	err := os.MkdirAll(plurHome, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: failed to create PLUR_HOME directory: %v\n", err)
		os.Exit(1)
	}

	binDir := filepath.Join(plurHome, "bin")
	cacheDir := filepath.Join(plurHome, "cache")
	runtimeDir := filepath.Join(plurHome, "runtime")
	formatterDir := filepath.Join(plurHome, "formatter")

	paths := []string{binDir, cacheDir, runtimeDir, formatterDir}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: failed to create %s directory: %v\n", path, err)
			os.Exit(1)
		}
	}

	configPaths := ConfigPaths{
		PlurHome:          plurHome,
		BinDir:            binDir,
		CacheDir:          cacheDir,
		RuntimeDir:        runtimeDir,
		FormatterDir:      formatterDir,
		JSONRowsFormatter: "", // Will be initialized lazily when needed
	}

	return &configPaths
}

// GetJSONRowsFormatterPath returns the path to the JSON rows formatter,
// We initialize it if needed - this is called lazily only when running RSpec tests.
// This function will exit the program if the formatter cannot be initialized.
func (c *ConfigPaths) GetJSONRowsFormatterPath() string {
	if c.JSONRowsFormatter != "" {
		return c.JSONRowsFormatter
	}

	formatter, err := rspec.GetFormatterPath(c.FormatterDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: failed to initialize RSpec formatter: %v\n", err)
		os.Exit(1)
	}

	c.JSONRowsFormatter = formatter
	return c.JSONRowsFormatter
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