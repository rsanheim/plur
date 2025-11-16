package watch

import (
	"fmt"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
	"github.com/stretchr/testify/assert"
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
			Targets: &config.MultiString{glob},
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
			Targets: &config.MultiString{source},
			Jobs:    config.MultiString{"rspec"},
		}
		assert.Equal(t, "test", watchMapping.SourceDir())
	}
}

func TestSourceDirForDirectory(t *testing.T) {
	watchMapping := WatchMapping{
		Source: "test/package/stuff/things/",
		Jobs:   config.MultiString{"rspec"},
	}
	assert.Equal(t, "test/package/stuff/things", watchMapping.SourceDir())
}

func TestSourceDirForRootDirectory(t *testing.T) {
	watchMapping := WatchMapping{
		Source: ".",
		Jobs:   config.MultiString{"rspec"},
	}
	assert.Equal(t, ".", watchMapping.SourceDir())
}
