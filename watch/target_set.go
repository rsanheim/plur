package watch

import (
	"iter"
	"slices"
)

// TargetSet is an immutable, duplicate-free collection of watch paths or job
// targets in insertion order. It is safe to share across debounce, planning,
// and JobRun boundaries.
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

// Union returns the targets in either set. Targets from s come first, followed
// by targets unique to other in their insertion order.
func (s TargetSet) Union(other TargetSet) TargetSet {
	values := make([]string, 0, s.Len()+other.Len())
	values = append(values, s.values...)
	for target := range other.All() {
		if !s.Contains(target) {
			values = append(values, target)
		}
	}
	return NewTargetSet(values...)
}

// Difference returns the targets in s that are not in other, preserving the
// insertion order of s.
func (s TargetSet) Difference(other TargetSet) TargetSet {
	values := make([]string, 0, s.Len())
	for target := range s.All() {
		if !other.Contains(target) {
			values = append(values, target)
		}
	}
	return NewTargetSet(values...)
}

// Intersection returns the targets shared by s and other, preserving the
// insertion order of s.
func (s TargetSet) Intersection(other TargetSet) TargetSet {
	values := make([]string, 0, s.Len())
	for target := range s.All() {
		if other.Contains(target) {
			values = append(values, target)
		}
	}
	return NewTargetSet(values...)
}

// All yields targets in insertion order.
func (s TargetSet) All() iter.Seq[string] {
	return slices.Values(s.values)
}

// Values returns the targets in insertion order.
func (s TargetSet) Values() []string {
	return slices.Clone(s.values)
}
