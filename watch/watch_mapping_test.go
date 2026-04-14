package watch

import (
	"fmt"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceDirForGlobs(t *testing.T) {
	globs := []string{
		"test/**/file*.go",
		"test/**/*.rb",
		"test/file*.rb",
		"test/**/*.html",
	}
	for _, glob := range globs {
		watchMapping := WatchMapping{
			Source:  glob,
			Targets: []string{glob},
		}
		assert.Equal(t, "test", watchMapping.SourceDir())
	}
}

func TestSourceDirForSingleFile(t *testing.T) {
	sources := []string{
		"test/foo-baz.go",
		"test/test.go",
		"test/whatever",
	}
	for _, source := range sources {
		base, _ := doublestar.SplitPattern(source)
		fmt.Println(base)

		watchMapping := WatchMapping{
			Source:  source,
			Targets: []string{source},
			Jobs:    []string{"rspec"},
		}
		assert.Equal(t, "test", watchMapping.SourceDir())
	}
}

func TestSourceDirForDirectory(t *testing.T) {
	watchMapping := WatchMapping{
		Source: "test/package/stuff/things/",
		Jobs:   []string{"rspec"},
	}
	assert.Equal(t, "test/package/stuff/things", watchMapping.SourceDir())
}

func TestSourceDirForRootDirectory(t *testing.T) {
	watchMapping := WatchMapping{
		Source: ".",
		Jobs:   []string{"rspec"},
	}
	assert.Equal(t, ".", watchMapping.SourceDir())
}

func TestMergeKey(t *testing.T) {
	tests := []struct {
		name     string
		watch    WatchMapping
		expected string
	}{
		{
			name: "named watch uses name only",
			watch: WatchMapping{
				Name:    "lib-to-spec",
				Source:  "lib/**/*.rb",
				Targets: []string{"spec/{{match}}_spec.rb"},
				Jobs:    []string{"rspec"},
			},
			expected: "name:lib-to-spec",
		},
		{
			name: "unnamed watch uses composite fields",
			watch: WatchMapping{
				Source:  "config/**/*.yml",
				Targets: []string{"spec/config_spec.rb"},
				Jobs:    []string{"rspec"},
				Ignore:  []string{"config/credentials/**"},
				Reload:  true,
			},
			expected: "source:config/**/*.yml|targets:[spec/config_spec.rb]|jobs:[rspec]|ignore:[config/credentials/**]|reload:true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.watch.mergeKey())
		})
	}
}

func TestMergeWatches(t *testing.T) {
	builtinLibToSpec := WatchMapping{
		Name:    "lib-to-spec",
		Source:  "lib/**/*.rb",
		Targets: []string{"spec/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}
	builtinSpecFiles := WatchMapping{
		Name:   "spec-files",
		Source: "spec/**/*_spec.rb",
		Jobs:   []string{"rspec"},
	}
	userCustom := WatchMapping{
		Name:    "custom-config",
		Source:  "config/**/*.yml",
		Targets: []string{"spec/config_spec.rb"},
		Jobs:    []string{"rspec"},
	}
	userOverride := WatchMapping{
		Name:    "lib-to-spec",
		Source:  "lib/**/*.rb",
		Targets: []string{"spec/overrides/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}
	unnamedDuplicate := WatchMapping{
		Source:  "lib/**/*.rb",
		Targets: []string{"spec/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}
	unnamedDifferent := WatchMapping{
		Source:  "lib/**/*.rb",
		Targets: []string{"spec/lib/{{match}}_spec.rb"},
		Jobs:    []string{"rspec"},
	}

	tests := []struct {
		name      string
		builtins  []WatchMapping
		user      []WatchMapping
		assertion func(t *testing.T, got []WatchMapping)
	}{
		{
			name: "both empty",
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Empty(t, got)
			},
		},
		{
			name:     "builtins only",
			builtins: []WatchMapping{builtinLibToSpec, builtinSpecFiles},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{builtinLibToSpec, builtinSpecFiles}, got)
			},
		},
		{
			name: "user only",
			user: []WatchMapping{userCustom},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{userCustom}, got)
			},
		},
		{
			name:     "additive named user watch appends",
			builtins: []WatchMapping{builtinLibToSpec, builtinSpecFiles},
			user:     []WatchMapping{userCustom},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{builtinLibToSpec, builtinSpecFiles, userCustom}, got)
			},
		},
		{
			name:     "override by name preserves position",
			builtins: []WatchMapping{builtinLibToSpec, builtinSpecFiles},
			user:     []WatchMapping{userOverride},
			assertion: func(t *testing.T, got []WatchMapping) {
				require.Len(t, got, 2)
				assert.Equal(t, userOverride, got[0])
				assert.Equal(t, builtinSpecFiles, got[1])
			},
		},
		{
			name:     "mix override and additive",
			builtins: []WatchMapping{builtinLibToSpec, builtinSpecFiles},
			user:     []WatchMapping{userOverride, userCustom},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{userOverride, builtinSpecFiles, userCustom}, got)
			},
		},
		{
			name:     "exact unnamed duplicate collapses",
			builtins: []WatchMapping{unnamedDuplicate},
			user:     []WatchMapping{unnamedDuplicate},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{unnamedDuplicate}, got)
			},
		},
		{
			name:     "unnamed different watch remains additive",
			builtins: []WatchMapping{unnamedDuplicate},
			user:     []WatchMapping{unnamedDifferent},
			assertion: func(t *testing.T, got []WatchMapping) {
				assert.Equal(t, []WatchMapping{unnamedDuplicate, unnamedDifferent}, got)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeWatches(tt.builtins, tt.user)
			tt.assertion(t, got)
		})
	}
}
