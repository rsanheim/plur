// Package kongtoml provides a Kong configuration resolver for TOML files.
//
// It parses TOML configuration files and resolves their values as Kong CLI flags.
// This package is designed to be self-contained with no application-specific
// dependencies, making it suitable for extraction as a standalone module.
package kongtoml

import (
	"io"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kong"
)

// Loader is a kong.ConfigurationLoader that reads TOML configuration.
func Loader(r io.Reader) (kong.Resolver, error) {
	var tree map[string]any
	md, err := toml.NewDecoder(r).Decode(&tree)
	if err != nil {
		return nil, err
	}
	var filename string
	if named, ok := r.(interface{ Name() string }); ok {
		filename = named.Name()
	}
	return &Resolver{filename: filename, tree: normalizeTree(tree), meta: md}, nil
}

// normalizeTree converts []map[string]any values (produced by BurntSushi/toml
// for array-of-tables) to []any so Kong's JSON-based unmarshaling works correctly.
func normalizeTree(m map[string]any) map[string]any {
	for k, v := range m {
		m[k] = normalizeValue(v)
	}
	return m
}

func normalizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return normalizeTree(val)
	case []map[string]any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = normalizeTree(item)
		}
		return result
	case []any:
		for i, item := range val {
			val[i] = normalizeValue(item)
		}
		return val
	}
	return v
}

var _ kong.Resolver = (*Resolver)(nil)

// Resolver resolves kong flags from a parsed TOML tree.
type Resolver struct {
	filename string
	tree     map[string]any
	meta     toml.MetaData
}

func (r *Resolver) Resolve(kctx *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
	value, ok := r.findValue(parent, flag)
	if !ok {
		return nil, nil
	}
	return value, nil
}

func (r *Resolver) Validate(app *kong.Application) error {
	return nil
}

func (r *Resolver) findValue(parent *kong.Path, flag *kong.Flag) (any, bool) {
	keys := []string{
		strings.ReplaceAll(parent.Node().Path(), " ", "-") + "-" + flag.Name,
		flag.Name,
	}
	for _, key := range keys {
		parts := strings.Split(key, "-")
		if value, ok := r.findValueParts(parts[0], parts[1:], r.tree); ok {
			return value, ok
		}
	}
	return nil, false
}

func (r *Resolver) findValueParts(prefix string, suffix []string, tree map[string]any) (any, bool) {
	if value, ok := tree[prefix]; ok {
		if len(suffix) == 0 {
			return value, true
		}
		if branch, ok := value.(map[string]any); ok {
			return r.findValueParts(suffix[0], suffix[1:], branch)
		}
	} else if len(suffix) > 0 {
		return r.findValueParts(prefix+"-"+suffix[0], suffix[1:], tree)
	}
	return nil, false
}
