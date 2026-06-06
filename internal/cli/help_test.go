package cli

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type helpTestCLI struct {
	Spec  helpTestSpecCmd  `cmd:"" group:"daily" help:"Run tests" default:"withargs"`
	Watch helpTestWatchCmd `cmd:"" help:"Watch for file changes and run tests automatically"`

	DryRun  bool `help:"Print what would be executed without running"`
	Workers int  `short:"n" help:"Number of parallel workers" default:"4"`
}

type helpTestSpecCmd struct {
	Patterns []string `arg:"" optional:""`
}

type helpTestWatchCmd struct {
	Run  helpTestWatchRunCmd  `cmd:"" default:"withargs" group:"daily" help:"Run watch mode"`
	Find helpTestWatchFindCmd `cmd:"" group:"daily" help:"Show what would be executed for a given file change"`
}

type helpTestWatchRunCmd struct{}

type helpTestWatchFindCmd struct {
	FilePath string `arg:"" required:"true"`
}

func TestConfigureHelpDetailsSetsApplicationAndWatchDetails(t *testing.T) {
	var cli helpTestCLI

	parser, err := kong.New(&cli,
		kong.Name("plur"),
		ConfigureHelpDetails(),
	)
	require.NoError(t, err)

	assert.Contains(t, parser.Model.Detail, "plur --dry-run")

	watch := childCommand(parser.Model.Node, "watch")
	require.NotNil(t, watch)
	assert.Contains(t, watch.Detail, "plur watch find spec/calculator_spec.rb")
}

func TestHelpPrinterAddsTopLevelUsageAndNormalizesExamplesSpacing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var exitCode int
	var cli helpTestCLI

	parser, err := kong.New(&cli,
		kong.Name("plur"),
		kong.Description("test description"),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true, FlagsLast: true}),
		ConfigureHelpDetails(),
		kong.Help(HelpPrinter),
		kong.Writers(&stdout, &stderr),
		kong.Exit(func(code int) {
			exitCode = code
			panic(helpExit{code: code})
		}),
	)
	require.NoError(t, err)

	assertHelpExit(t, func() {
		_, err = parser.Parse([]string{"--help"})
		require.NoError(t, err)
	})

	assert.Equal(t, 0, exitCode)
	assert.Empty(t, stderr.String())
	assert.Contains(t, stdout.String(), "Usage: plur [patterns...] [flags]")
	assert.Contains(t, stdout.String(), "       plur <command> [flags]")
	assert.Contains(t, stdout.String(), "Examples:\n    plur")
	assert.NotContains(t, stdout.String(), "Examples:\n\n")
}

type helpExit struct {
	code int
}

func assertHelpExit(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		require.Equal(t, helpExit{code: 0}, recovered)
	}()

	fn()
}
