package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

type WatchCmd struct {
	Timeout  int  `help:"Exit after specified seconds (default: run until Ctrl-C)"`
	Debounce int  `help:"Debounce delay in milliseconds" default:"100"`
	Verbose  bool `help:"Enable verbose output for debugging"`
}

func (w *WatchCmd) Run() error {
	Logger.Info("rux-kong watch starting",
		"timeout", w.Timeout,
		"debounce", w.Debounce,
		"verbose", w.Verbose)

	// TODO: Call the actual runWatch logic here
	// For now, just show it would work
	Logger.Warn("kong watch not fully implemented yet")
	return nil
}

type SpecCmd struct {
	Glob string `arg:"" help:"Glob pattern to run" default:"spec/**/*_spec.rb"`
}

func (cmd *SpecCmd) Run() error {
	Logger.Info("rux-kong spec starting", "glob", cmd.Glob)

	files, err := ExpandGlobPatterns([]string{cmd.Glob})
	if err != nil {
		return err
	}
	Logger.Info("found files", "files", files)
	return nil
}

var KongCLI struct {
	Spec  SpecCmd  `cmd:"" help:"Run tests" default:"withargs"`
	Watch WatchCmd `cmd:"" help:"Watch for file changes and run tests automatically"`

	// Global flags
	Auto       bool   `help:"Automatically run bundle install before tests" default:"false"`
	Verbose    bool   `help:"Enable verbose output for debugging" default:"false"`
	Debug      bool   `help:"Enable debug output (includes verbose)" default:"false"`
	DryRun     bool   `help:"Print what would be executed without running" default:"false"`
	JSON       string `help:"Save detailed test results as JSON to the specified file" default:""`
	Color      bool   `help:"Force colorized output (auto-detected by default)" negatable:"" default:"true"`
	RuntimeDir string `help:"Custom directory for runtime data" default:""`
	CacheDir   string `help:"Directory for caching runtime data" default:"${cache_dir}"`
	Trace      bool   `help:"Enable performance tracing (saves to ./rux_trace_*.json)" default:"false"`
	Workers    int    `short:"n" help:"Number of parallel workers (default: auto-detect CPUs)" default:"0"`
}

func runKongCLI() {
	// Get cache directory early - fail if environment is broken

	paths, configErr := InitConfigPaths()
	if configErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", configErr)
		os.Exit(1)
	}

	ctx := kong.Parse(&KongCLI,
		kong.Vars{
			"cache_dir": paths.CacheDir,
		})

	// Initialize logging before running any command (same as main.go Before hook)
	debug := KongCLI.Debug || os.Getenv("RUX_DEBUG") == "1"
	InitLogger(KongCLI.Verbose, debug)

	if KongCLI.DryRun {
		Logger.Info("kong dry run mode - exiting")
		return
	}

	err := ctx.Run()
	if err != nil {
		Logger.Error("Command failed", "error", err)
	}
}
