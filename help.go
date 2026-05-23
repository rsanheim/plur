package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/alecthomas/kong"
)

func customHelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	var buf bytes.Buffer
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
		return insertAfterDescription(output, watchWorkflowHelp())
	}

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
  plur test                           Run Minitest targets
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
