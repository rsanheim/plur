package runtime

import (
	"fmt"
	"testing"
)

// BenchmarkAnyFileMatches walks a project-shaped tree: app/ and lib/ sources
// sit ahead of spec/ in walk order, and node_modules/ plus a hidden directory
// are full of would-be matches that pruning must never visit.
func BenchmarkAnyFileMatches(b *testing.B) {
	b.Chdir(b.TempDir())
	for top, suffix := range map[string]string{
		"app":          ".rb",
		"lib":          ".rb",
		"spec":         "_spec.rb",
		"node_modules": "_test.go",
		".hidden":      "_test.go",
	} {
		for dir := range 40 {
			for file := range 25 {
				writeDetectFile(b, fmt.Sprintf("%s/d%02d/f%02d%s", top, dir, file, suffix))
			}
		}
	}

	for _, bc := range []struct {
		name    string
		pattern string
		want    bool
	}{
		{"match", "**/*_spec.rb", true},
		{"no match", "**/*_test.go", false},
	} {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				found, err := anyFileMatches(bc.pattern)
				if err != nil || found != bc.want {
					b.Fatalf("anyFileMatches(%q) = %v, %v; want %v", bc.pattern, found, err, bc.want)
				}
			}
		})
	}
}
