package main

import (
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
	Verbose bool `help:"Enable verbose output for debugging"`
	Debug   bool `help:"Enable debug output (includes verbose)"`
	DryRun  bool `help:"Print what would be executed without running"`
}

func runKongCLI() {
	ctx := kong.Parse(&KongCLI)

	// Initialize logging before running any command (same as main.go Before hook)
	debug := KongCLI.Debug || os.Getenv("RUX_DEBUG") == "1"
	InitLogger(KongCLI.Verbose, debug)

	if KongCLI.DryRun {
		Logger.Info("kong dry run", "args", ctx.Args)
		return
	}

	Logger.Info("kong ct", "args", ctx.Args)

	err := ctx.Run()
	if err != nil {
		Logger.Error("Command failed", "error", err)
	}
}
