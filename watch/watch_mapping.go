package watch

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

// WatchMapping represents a source->target mapping rule for watch mode
type WatchMapping struct {
	Name    string   `toml:"name,omitempty" json:"name,omitempty"`
	Source  string   `toml:"source" json:"source"`
	Targets []string `toml:"targets,omitempty" json:"targets,omitempty"`
	Jobs    []string `toml:"jobs" json:"jobs"`
	Ignore  []string `toml:"ignore,omitempty" json:"ignore,omitempty"`
	Reload  bool     `toml:"reload,omitempty" json:"reload,omitempty"` // Reload plur after jobs complete
}

// SourceDir returns the directory part of the source pattern
func (w WatchMapping) SourceDir() string {
	base, _ := doublestar.SplitPattern(w.Source)
	return base
}

func (w WatchMapping) String() string {
	return fmt.Sprintf("WatchMapping{Name: %s, Source: %s, Targets: %s, Jobs: %s, Ignore: %s, Reload: %t, SourceDir: %s}", w.Name, w.Source, w.Targets, w.Jobs, w.Ignore, w.Reload, w.SourceDir())
}

func (w WatchMapping) mergeKey() string {
	if w.Name != "" {
		return "name:" + w.Name
	}
	return fmt.Sprintf("source:%s|targets:%v|jobs:%v|ignore:%v|reload:%t", w.Source, w.Targets, w.Jobs, w.Ignore, w.Reload)
}

// MergeWatches combines built-in and user watch mappings.
// Named user watches override built-ins with the same name, preserving position.
// Callers are expected to reject duplicate named user watches before merging.
// Unnamed watches remain additive unless they are exact duplicates.
func MergeWatches(builtins, user []WatchMapping) []WatchMapping {
	result := make([]WatchMapping, 0, len(builtins)+len(user))
	indexByKey := make(map[string]int)
	for _, list := range [][]WatchMapping{builtins, user} {
		for _, w := range list {
			key := w.mergeKey()
			if i, ok := indexByKey[key]; ok {
				result[i] = w
			} else {
				indexByKey[key] = len(result)
				result = append(result, w)
			}
		}
	}
	return result
}
