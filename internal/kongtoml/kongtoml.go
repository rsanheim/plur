// Package kongtoml provides a Kong configuration resolver for TOML files.
//
// It parses TOML configuration files and resolves their values as Kong CLI flags.
// Validation is intentionally narrower than the full CLI model: only documented
// persistent config keys are accepted from TOML.
package kongtoml

import (
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
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
	slog.Debug("loaded config",
		"file", configName(filename),
		"key_count", len(md.Keys()),
		"top_level_keys", topLevelKeys(md),
	)
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

func (r *Resolver) Validate(_ *kong.Application) error {
	unknown := unknownLeafKeys(r.meta)
	if len(unknown) == 0 {
		cliOnly := cliOnlyConfigKeys(r.meta)
		if len(cliOnly) == 0 {
			return nil
		}

		slog.Debug("cli-only config keys",
			"file", configName(r.filename),
			"keys", cliOnly,
		)

		label := "CLI-only config key"
		if len(cliOnly) > 1 {
			label = "CLI-only config keys"
		}
		return fmt.Errorf("Configuration error: %s contains %s: %s; pass these as command-line flags instead", configName(r.filename), label, strings.Join(cliOnly, ", "))
	}

	slog.Debug("unknown config keys",
		"file", configName(r.filename),
		"keys", unknown,
	)

	label := "unknown config key"
	if len(unknown) > 1 {
		label = "unknown config keys"
	}
	return fmt.Errorf("Configuration error: %s contains %s: %s", configName(r.filename), label, strings.Join(unknown, ", "))
}

var persistentFlatConfigKeys = []string{
	"workers",
	"color",
	"verbose",
	"use",
}

var cliOnlyFlatConfigKeys = []string{
	"dry-run",
	"dry-run-format",
}

func cliOnlyConfigKeys(md toml.MetaData) []string {
	cliOnly := make(map[string]struct{}, len(cliOnlyFlatConfigKeys))
	for _, key := range cliOnlyFlatConfigKeys {
		cliOnly[key] = struct{}{}
	}
	set := make(map[string]struct{})
	for _, key := range md.Keys() {
		keyStr := key.String()
		if _, ok := cliOnly[keyStr]; ok {
			set[keyStr] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func configName(filename string) string {
	if filename != "" {
		return filename
	}
	return "<unknown>"
}

func topLevelKeys(md toml.MetaData) []string {
	set := make(map[string]struct{})
	for _, key := range md.Keys() {
		if len(key) == 0 {
			continue
		}
		set[key[0]] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func unknownLeafKeys(md toml.MetaData) []string {
	allowed := recognizedConfigKeys()
	set := make(map[string]struct{})
	for _, key := range md.Keys() {
		keyStr := key.String()
		if keyStr == "" {
			continue
		}
		if allowed.matches(keyStr) {
			continue
		}
		set[keyStr] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

type configKeySet struct {
	flat   map[string]struct{}
	nested map[string]nestedKeySpec
}

type nestedKeySpec struct {
	dynamicName bool
	allowed     map[string]struct{}
}

func (c configKeySet) matches(key string) bool {
	if _, ok := c.flat[key]; ok {
		return true
	}

	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return false
	}

	spec, ok := c.nested[parts[0]]
	if !ok {
		return false
	}
	return spec.matches(parts[1:])
}

func (s nestedKeySpec) matches(parts []string) bool {
	if len(parts) == 0 {
		return true
	}
	if s.dynamicName {
		if len(parts) == 1 {
			return true
		}
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return true
	}
	_, ok := s.allowed[strings.Join(parts, ".")]
	return ok
}

func recognizedConfigKeys() configKeySet {
	allowed := persistentConfigKeys()
	for _, key := range cliOnlyFlatConfigKeys {
		allowed.flat[key] = struct{}{}
	}
	return allowed
}

func persistentConfigKeys() configKeySet {
	allowed := configKeySet{
		flat:   make(map[string]struct{}),
		nested: make(map[string]nestedKeySpec),
	}

	for _, key := range persistentFlatConfigKeys {
		allowed.flat[key] = struct{}{}
	}

	allowed.nested["job"] = nestedKeySpec{
		dynamicName: true,
		allowed:     structFieldKeys(reflect.TypeOf(job.Job{})),
	}
	allowed.nested["watch"] = nestedKeySpec{
		allowed: structFieldKeys(reflect.TypeOf(watch.WatchMapping{})),
	}
	return allowed
}

func structFieldKeys(typ reflect.Type) map[string]struct{} {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}

	allowed := make(map[string]struct{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		name := tomlFieldName(field)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	return allowed
}

func tomlFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("toml")
	if tag == "-" {
		return ""
	}
	if tag != "" {
		name := strings.Split(tag, ",")[0]
		if name != "" {
			return name
		}
	}
	return field.Name
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
			if nested, ok := r.findValueParts(suffix[0], suffix[1:], branch); ok {
				return nested, true
			}
		}
	}
	if len(suffix) > 0 {
		return r.findValueParts(prefix+"-"+suffix[0], suffix[1:], tree)
	}
	return nil, false
}
