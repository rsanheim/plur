package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig holds settings that are truly global across all commands
type GlobalConfig struct {
	ColorOutput bool
	ColorSource string // short source tag from term.ResolveColor, shown by doctor and --verbose
	ConfigPaths *ConfigPaths
	Debug       bool
	Verbose     bool
	DryRun      bool
	WorkerCount int
	RuntimeDir  string
	FirstIs1    bool // Start TEST_ENV_NUMBER at 1 instead of empty string
	RspecTrace  bool // Prefix stdout/stderr with source file path (RSpec only)
	RspecSplit  bool // EXPERIMENTAL: split long RSpec files into focused file:line targets

	// Configuration source tracking
	LoadedConfigs []string // List of config files that actually exist and were loaded
}

// IsSerial returns true if running in serial mode (single worker)
func (c *GlobalConfig) IsSerial() bool {
	return c.WorkerCount == 1
}

type ConfigPaths struct {
	PlurHome     string // ~/.plur or $PLUR_HOME
	BinDir       string
	CacheDir     string
	RuntimeDir   string
	FormatterDir string
	RubyLibDir   string // added to ruby $LOAD_PATH via -I for minitest plugin discovery
}

// InitConfigPaths initializes PLUR_HOME if necessary, as well as subdirs inside it.
// By default this will be ~/.plur unless PLUR_HOME is set by the user.
func InitConfigPaths() (*ConfigPaths, error) {
	plurHome, ok := os.LookupEnv("PLUR_HOME")
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("find home directory with PLUR_HOME unset: %w", err)
		}
		plurHome = filepath.Join(homeDir, ".plur")
	}

	if err := os.MkdirAll(plurHome, 0755); err != nil {
		return nil, fmt.Errorf("create PLUR_HOME directory %q: %w", plurHome, err)
	}

	binDir := filepath.Join(plurHome, "bin")
	cacheDir := filepath.Join(plurHome, "cache")
	runtimeDir := filepath.Join(plurHome, "runtime")
	formatterDir := filepath.Join(plurHome, "formatter")
	rubyLibDir := filepath.Join(plurHome, "config", "ruby")

	paths := []string{binDir, cacheDir, runtimeDir, formatterDir, rubyLibDir}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("create %s directory: %w", path, err)
		}
	}

	configPaths := ConfigPaths{
		PlurHome:     plurHome,
		BinDir:       binDir,
		CacheDir:     cacheDir,
		RuntimeDir:   runtimeDir,
		FormatterDir: formatterDir,
		RubyLibDir:   rubyLibDir,
	}

	return &configPaths, nil
}
