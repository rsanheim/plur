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
	Ignore []string `toml:"ignore,omitempty" json:"ignore,omitempty"`
}

// SourceDir returns the directory part of the source pattern
func (w WatchMapping) SourceDir() string {
	base, _ := doublestar.SplitPattern(w.Source)
	return base
}

func (w WatchMapping) String() string {
	return fmt.Sprintf("WatchMapping{Name: %s, Source: %s, Targets: %s, Jobs: %s, Ignore: %s, SourceDir: %s}", w.Name, w.Source, w.Targets, w.Jobs, w.Ignore, w.SourceDir())
}
