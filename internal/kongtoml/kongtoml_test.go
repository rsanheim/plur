package kongtoml

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedReader struct {
	*strings.Reader
	name string
}

func (n *namedReader) Name() string { return n.name }

func TestLoaderParsesValidTOML(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)
	assert.NotNil(t, resolver)
}

func TestLoaderReturnsErrorForInvalidTOML(t *testing.T) {
	input := `[broken`
	resolver, err := Loader(strings.NewReader(input))
	assert.Error(t, err)
	assert.Nil(t, resolver)
}

func TestLoaderExtractsFilename(t *testing.T) {
	input := `workers = 4`
	r := &namedReader{Reader: strings.NewReader(input), name: ".plur.toml"}
	resolver, err := Loader(r)
	require.NoError(t, err)
	res := resolver.(*Resolver)
	assert.Equal(t, ".plur.toml", res.filename)
}

func TestLoaderNoFilenameForPlainReader(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)
	res := resolver.(*Resolver)
	assert.Empty(t, res.filename)
}

// Helper to build a minimal kong.Path and kong.Flag for testing Resolve.
// The flag name and parent path are the only fields the resolver uses.
func makeParentAndFlag(parentPath, flagName string) (*kong.Path, *kong.Flag) {
	var path *kong.Path
	if parentPath == "" {
		// For root-level flags, use an Application path so Node() returns a node with empty Path()
		path = &kong.Path{App: &kong.Application{Node: &kong.Node{}}}
	} else {
		parts := strings.Fields(parentPath)
		var parent *kong.Node
		for _, part := range parts {
			node := &kong.Node{Name: part, Type: kong.CommandNode, Parent: parent}
			parent = node
		}
		path = &kong.Path{Command: parent}
	}
	flag := &kong.Flag{Value: &kong.Value{Name: flagName}}
	return path, flag
}

func TestResolveFlatKey(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "workers")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Equal(t, int64(4), val)
}

func TestResolveNestedKey(t *testing.T) {
	input := `
[job]
[job.rspec]
cmd = ["bin/rspec"]
`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "job")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.NotNil(t, val)

	jobMap, ok := val.(map[string]any)
	require.True(t, ok)
	rspec, ok := jobMap["rspec"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"bin/rspec"}, rspec["cmd"])
}

func TestResolveMissingKey(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "nonexistent")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestResolveStringValue(t *testing.T) {
	input := `use = "rspec"`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "use")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Equal(t, "rspec", val)
}

func TestResolveBoolValue(t *testing.T) {
	input := `color = true`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "color")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Equal(t, true, val)
}

func TestResolveArrayOfTables(t *testing.T) {
	input := `
[[watch]]
source = "**/*.go"
jobs = ["build"]

[[watch]]
source = "**/*_spec.rb"
jobs = ["rspec"]
`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("", "watch")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)

	// BurntSushi/toml returns []map[string]any for array-of-tables, but Kong
	// needs []any for JSON unmarshaling. normalizeTree handles this conversion.
	arr, ok := val.([]any)
	require.True(t, ok, "expected []any, got %T", val)
	assert.Len(t, arr, 2)

	first, ok := arr[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "**/*.go", first["source"])
}

func TestResolveCommandPathWithSpaces(t *testing.T) {
	input := `
[watch.run]
debounce = 250
`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("watch run", "debounce")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Equal(t, int64(250), val)
}

func TestResolveHyphenatedFlattenedKey(t *testing.T) {
	input := `watch-run-timeout = 5`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	parent, flag := makeParentAndFlag("watch run", "timeout")
	val, err := resolver.Resolve(nil, parent, flag)
	require.NoError(t, err)
	assert.Equal(t, int64(5), val)
}

func TestTopLevelKeys(t *testing.T) {
	input := `
workers = 2
use = "rspec"

[job.rspec]
cmd = ["bin/rspec"]

[[watch]]
source = "spec/**/*_spec.rb"
jobs = ["rspec"]
`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)
	r := resolver.(*Resolver)

	assert.Equal(t, []string{"job", "use", "watch", "workers"}, topLevelKeys(r.meta))
}

func TestUnknownLeafKeys(t *testing.T) {
	input := `
workers = 2
wokers = 3
use = "rspec"

[job.rspec]
cmd = ["bin/rspec"]
`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)
	r := resolver.(*Resolver)

	var cli struct {
		Workers int
		Use     string
		Job     map[string]any
		Watch   []map[string]any
	}
	parser, err := kong.New(&cli)
	require.NoError(t, err)

	assert.Equal(t, []string{"wokers"}, unknownLeafKeys(r.meta, parser.Model))
}

func TestValidateReturnsNil(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	err = resolver.(*Resolver).Validate(nil)
	assert.NoError(t, err)
}

func TestKeyResolutionEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		toml       string
		parentPath string
		flagName   string
		wantValue  any
	}{
		{
			name:      "hyphenated flag resolves flat key",
			toml:      `dry-run = true`,
			flagName:  "dry-run",
			wantValue: true,
		},
		{
			name:      "hyphenated flag resolves nested table",
			toml:      "[dry]\nrun = true",
			flagName:  "dry-run",
			wantValue: true,
		},
		{
			name:      "nested table takes precedence over flat key",
			toml:      "dry-run = false\n[dry]\nrun = true",
			flagName:  "dry-run",
			wantValue: true,
		},
		{
			// findValueParts finds tree["dry"] (a scalar), then tries to descend
			// into it for "run" — but it's not a map, so resolution fails entirely.
			// The flat key "dry-run" is never tried because the else-if branch
			// only runs when tree[prefix] does NOT exist.
			name:      "scalar key shadows same-prefix hyphenated key",
			toml:      "dry = \"something\"\ndry-run = true",
			flagName:  "dry-run",
			wantValue: nil,
		},
		{
			name:       "command-scoped takes precedence over global",
			toml:       "timeout = 50\n[watch.run]\ntimeout = 100",
			parentPath: "watch run",
			flagName:   "timeout",
			wantValue:  int64(100),
		},
		{
			name:       "falls back to global when no command-scoped key exists",
			toml:       `timeout = 50`,
			parentPath: "watch run",
			flagName:   "timeout",
			wantValue:  int64(50),
		},
		{
			name:       "three-level command nesting",
			toml:       "[watch.run.debug]\nverbose = true",
			parentPath: "watch run debug",
			flagName:   "verbose",
			wantValue:  true,
		},
		{
			name:       "hyphenated flag under command context",
			toml:       "[watch]\ndry-run = true",
			parentPath: "watch",
			flagName:   "dry-run",
			wantValue:  true,
		},
		{
			name:      "multi-hyphen flag resolves flat key",
			toml:      `some-very-long-flag = "value"`,
			flagName:  "some-very-long-flag",
			wantValue: "value",
		},
		{
			name:      "multi-hyphen flag resolves fully nested",
			toml:      "[some.very.long]\nflag = \"nested\"",
			flagName:  "some-very-long-flag",
			wantValue: "nested",
		},
		{
			name:      "multi-hyphen flag resolves partially nested",
			toml:      "[some.very]\nlong-flag = \"partial\"",
			flagName:  "some-very-long-flag",
			wantValue: "partial",
		},
		{
			// A command with hyphens in its own name: the command path "my-cmd"
			// gets split on hyphens too, so [my-cmd] and [my.cmd] are both
			// reachable. Flat form works here.
			name:       "hyphenated command name with flat table key",
			toml:       "[my-cmd]\nopt = 42",
			parentPath: "my-cmd",
			flagName:   "opt",
			wantValue:  int64(42),
		},
		{
			// Same hyphenated command, but using dot-separated tables.
			// findValueParts splits "my-cmd-opt" into ["my","cmd","opt"]
			// and walks [my.cmd] to find opt.
			name:       "hyphenated command name with nested tables",
			toml:       "[my.cmd]\nopt = 42",
			parentPath: "my-cmd",
			flagName:   "opt",
			wantValue:  int64(42),
		},
		{
			name:      "empty config returns nil for any flag",
			toml:      "",
			flagName:  "workers",
			wantValue: nil,
		},
		{
			name:      "comments-only config returns nil for any flag",
			toml:      "# just a comment\n# another comment",
			flagName:  "workers",
			wantValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := Loader(strings.NewReader(tt.toml))
			require.NoError(t, err)

			parent, flag := makeParentAndFlag(tt.parentPath, tt.flagName)
			val, err := resolver.Resolve(nil, parent, flag)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, val)
		})
	}
}
