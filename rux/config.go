package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rsanheim/rux/rspec"
	"github.com/urfave/cli/v2"
)

// Config holds top level config for rux
type Config struct {
	Auto         bool
	ColorOutput  bool
	ConfigPaths  *ConfigPaths
	DryRun       bool
	TraceEnabled bool
	WorkerCount  int
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

func BuildConfig(ctx *cli.Context, paths *ConfigPaths) *Config {
	return &Config{
		Auto:         ctx.Bool("auto"),
		ColorOutput:  shouldUseColor(ctx),
		ConfigPaths:  paths,
		DryRun:       ctx.Bool("dry-run"),
		TraceEnabled: ctx.Bool("trace"),
		WorkerCount:  GetWorkerCount(ctx.Int("n")),
	}
}
