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
// Named user watches override built-ins with the same name.
// Unnamed watches remain additive unless they are exact duplicates.
func MergeWatches(builtins, user []WatchMapping) []WatchMapping {
	merged := make(map[string]WatchMapping, len(builtins)+len(user))
	order := make([]string, 0, len(builtins)+len(user))

	for _, watch := range builtins {
		key := watch.mergeKey()
		if _, exists := merged[key]; !exists {
			order = append(order, key)
		}
		merged[key] = watch
	}

	for _, watch := range user {
		key := watch.mergeKey()
		if _, exists := merged[key]; !exists {
			order = append(order, key)
		}
		merged[key] = watch
	}

	result := make([]WatchMapping, 0, len(order))
	for _, key := range order {
		result = append(result, merged[key])
	}

	return result
}
