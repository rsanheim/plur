package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

func customHelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	restoreHiddenFlags := hideIrrelevantHelpFlags(ctx.Selected())
	defer restoreHiddenFlags()

	if usage, ok := customUsage(ctx); ok {
		fmt.Fprintln(ctx.Stdout, usage)
		options.NoAppSummary = true
	}

	stdout := ctx.Stdout
	var output bytes.Buffer
	ctx.Stdout = &output
	defer func() {
		ctx.Stdout = stdout
	}()

	if err := kong.DefaultHelpPrinter(options, ctx); err != nil {
		return err
	}

	_, err := fmt.Fprint(stdout, normalizeHelpSpacing(output.String()))
	return err
}

func normalizeHelpSpacing(help string) string {
	// Kong formats Detail/HelpProvider text through go/doc, which inserts a
	// blank line between an "Examples:" heading and the indented example block.
	return strings.ReplaceAll(help, "Examples:\n\n", "Examples:\n")
}

func customUsage(ctx *kong.Context) (string, bool) {
	selected := ctx.Selected()
	if selected == nil {
		return "Usage: plur [patterns...] [flags]\n       plur <command> [flags]", true
	}

	if selected.FullPath() == "plur watch" {
		return "Usage: plur watch [flags]\n       plur watch find <changed-file> [flags]\n       plur watch <command> [flags]", true
	}

	return "", false
}

func hideIrrelevantHelpFlags(selected *kong.Node) func() {
	if selected == nil {
		return func() {}
	}

	hiddenNames := irrelevantHelpFlagNames(selected.FullPath())
	if len(hiddenNames) == 0 {
		return func() {}
	}

	type hiddenState struct {
		flag   *kong.Flag
		hidden bool
	}

	states := []hiddenState{}
	for _, group := range selected.AllFlags(false) {
		for _, flag := range group {
			if hiddenNames[flag.Name] {
				states = append(states, hiddenState{flag: flag, hidden: flag.Hidden})
				flag.Hidden = true
			}
		}
	}

	return func() {
		for _, state := range states {
			state.flag.Hidden = state.hidden
		}
	}
}

func irrelevantHelpFlagNames(fullPath string) map[string]bool {
	switch fullPath {
	case "plur watch", "plur watch run", "plur watch find":
		return map[string]bool{
			"dry-run":     true,
			"first-is-1":  true,
			"json":        true,
			"rspec-split": true,
			"workers":     true,
		}
	default:
		return nil
	}
}

func configureHelpDetails() kong.Option {
	return kong.PostBuild(func(k *kong.Kong) error {
		k.Model.Detail = topLevelWorkflowHelp()
		return nil
	})
}

func (w *WatchCmd) Help() string {
	return watchWorkflowHelp()
}

func topLevelWorkflowHelp() string {
	return `Examples:
  plur                                  # Run the detected test suite
  plur spec/calculator_spec.rb          # Run one file
  plur --dry-run                        # Preview the test plan
  plur watch find spec/calculator_spec.rb  # Preview a watch file change`
}

func watchWorkflowHelp() string {
	return `Examples:
  plur watch                            # Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  # Preview which tests a change runs`
}
