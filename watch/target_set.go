package watch

import (
	"iter"
	"slices"
)

// TargetSet is a duplicate-free collection of targets in insertion order.
// It is immutable after construction and safe to copy between JobRuns.
type TargetSet struct {
	values []string
	index  map[string]struct{}
}

func NewTargetSet(targets ...string) TargetSet {
	if len(targets) == 0 {
		return TargetSet{}
	}

	set := TargetSet{
		values: make([]string, 0, len(targets)),
		index:  make(map[string]struct{}, len(targets)),
	}
	for _, target := range targets {
		if _, exists := set.index[target]; exists {
			continue
		}
		set.index[target] = struct{}{}
		set.values = append(set.values, target)
	}
	return set
}

func (s TargetSet) Len() int {
	return len(s.values)
}

func (s TargetSet) Contains(target string) bool {
	_, exists := s.index[target]
	return exists
}

// All yields targets in insertion order.
func (s TargetSet) All() iter.Seq[string] {
	return slices.Values(s.values)
}

// Values returns the targets in insertion order.
func (s TargetSet) Values() []string {
	return slices.Clone(s.values)
}
