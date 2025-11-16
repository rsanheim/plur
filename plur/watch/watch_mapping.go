package watch

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
)

// WatchMapping represents a source->target mapping rule for watch mode
type WatchMapping struct {
	Name    string              `toml:"name,omitempty" json:"name,omitempty"`
	Source  string              `toml:"source" json:"source"`
	Targets *config.MultiString `toml:"targets,omitempty" json:"targets,omitempty"`
	Jobs    config.MultiString  `toml:"jobs" json:"jobs"`
	Exclude []string            `toml:"exclude,omitempty" json:"exclude,omitempty"`

	sourceDir string `toml:"-" json:"-"`
}

// SourceDir returns the directory part of the source pattern
func (w *WatchMapping) SourceDir() string {
	if w.sourceDir == "" {
		w.sourceDir = w.calculateSourceDir()
	}
	return w.sourceDir
}

func (w *WatchMapping) calculateSourceDir() string {
	base, _ := doublestar.SplitPattern(w.Source)
	return base
}
