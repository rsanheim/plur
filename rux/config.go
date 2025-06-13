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
	SpecFiles    []string
	TraceEnabled bool
	WorkerCount  int
}

type ConfigPaths struct {
	RuxHome           string // ~/.rux or $RUX_HOME
	CacheDir          string
	RuntimeDir        string
	Formatters        string
	JSONRowsFormatter string
}

func InitConfigPaths() (*ConfigPaths, error) {
	ruxHome, ok := os.LookupEnv("RUX_HOME")
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find home directory and RUX_HOME not set: %w", err)
		}
		ruxHome = filepath.Join(homeDir, ".rux")
	}

	err := os.MkdirAll(ruxHome, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create RUX_HOME directory: %v", err)
	}

	paths := map[string]string{
		"cache":      filepath.Join(ruxHome, "cache"),
		"runtime":    filepath.Join(ruxHome, "runtime"),
		"formatters": filepath.Join(ruxHome, "formatter"),
	}

	for _, path := range paths {
		if os.MkdirAll(path, 0755) != nil {
			return nil, fmt.Errorf("failed to create %s directory: %v", path, err)
		}
	}

	formattersPath := filepath.Join(ruxHome, "formatters")

	jsonRowsFormatter, err := rspec.GetFormatterPath(formattersPath)
	if err != nil {
		return nil, err
	}

	configPaths := ConfigPaths{
		RuxHome:           ruxHome,
		CacheDir:          paths["cache"],
		RuntimeDir:        paths["runtime"],
		Formatters:        paths["formatter"],
		JSONRowsFormatter: jsonRowsFormatter,
	}

	return &configPaths, nil
}

func BuildConfig(ctx *cli.Context, paths *ConfigPaths) (*Config, error) {
	specFiles, err := discoverSpecFiles(ctx)
	if err != nil {
		return nil, err
	}

	return &Config{
		Auto:         ctx.Bool("auto"),
		ColorOutput:  shouldUseColor(ctx),
		ConfigPaths:  paths,
		DryRun:       ctx.Bool("dry-run"),
		SpecFiles:    specFiles,
		TraceEnabled: ctx.Bool("trace"),
		WorkerCount:  GetWorkerCount(ctx.Int("n")),
	}, nil
}
