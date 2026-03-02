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

func TestValidateReturnsNil(t *testing.T) {
	input := `workers = 4`
	resolver, err := Loader(strings.NewReader(input))
	require.NoError(t, err)

	err = resolver.(*Resolver).Validate(nil)
	assert.NoError(t, err)
}
