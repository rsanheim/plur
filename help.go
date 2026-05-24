package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/alecthomas/kong"
)

func customHelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	var buf bytes.Buffer
	restoreHiddenFlags := hideIrrelevantHelpFlags(ctx.Selected())
	defer restoreHiddenFlags()

	originalStdout := ctx.Stdout
	ctx.Stdout = &buf
	err := kong.DefaultHelpPrinter(options, ctx)
	ctx.Stdout = originalStdout
	if err != nil {
		return err
	}

	_, err = io.WriteString(ctx.Stdout, customizeHelpOutput(buf.String(), ctx))
	return err
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
	case "plur watch find":
		return map[string]bool{
			"dry-run":        true,
			"dry-run-format": true,
			"first-is-1":     true,
			"json":           true,
			"rspec-split":    true,
			"workers":        true,
		}
	case "plur watch", "plur watch run":
		return map[string]bool{
			"dry-run":        true,
			"dry-run-format": true,
			"first-is-1":     true,
			"rspec-split":    true,
			"workers":        true,
		}
	default:
		return nil
	}
}

func customizeHelpOutput(output string, ctx *kong.Context) string {
	selected := ctx.Selected()
	if selected == nil {
		output = strings.Replace(output,
			"Usage: plur <command> [flags]",
			"Usage: plur [patterns...] [flags]\n       plur <command> [flags]",
			1)
		return insertAfterDescription(output, topLevelWorkflowHelp())
	}

	if selected.FullPath() == "plur watch" {
		output = strings.Replace(output,
			"Usage: plur watch <command> [flags]",
			"Usage: plur watch [flags]\n       plur watch find <changed-file> [flags]\n       plur watch <command> [flags]",
			1)
		output = customizeWatchDryRunHelp(output)
		return insertAfterDescription(output, watchWorkflowHelp())
	}

	if selected.FullPath() == "plur watch run" {
		return customizeWatchDryRunHelp(output)
	}

	return output
}

func customizeWatchDryRunHelp(output string) string {
	output = strings.Replace(output,
		"--dry-run                  Print what would be executed without running",
		"--dry-run                  One-shot run preview only; watch mode rejects it",
		1)
	output = strings.Replace(output,
		"--dry-run-format=\"text\"    Dry-run output format: text or json",
		"--dry-run-format=\"text\"    One-shot dry-run output format: text or json",
		1)
	return output
}

func insertAfterDescription(output, insertion string) string {
	const marker = "\nFlags:\n"
	if !strings.Contains(output, marker) {
		return output
	}
	return strings.Replace(output, marker, "\n"+insertion+"\nFlags:\n", 1)
}

func topLevelWorkflowHelp() string {
	return `Common workflows:
  plur                                Run the detected test suite
  plur spec/calculator_spec.rb        Run one target
  plur test/calculator_test.rb        Run one Minitest target
  plur --dry-run                      Preview the one-shot test plan
  plur watch                          Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  Preview a watch file change
`
}

func watchWorkflowHelp() string {
	return `Common workflows:
  plur watch                          Watch files and run matching tests
  plur watch find spec/calculator_spec.rb  Preview which tests a change runs
  plur --dry-run [patterns...]        Preview a one-shot test run
`
}
