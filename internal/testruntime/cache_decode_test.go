package testruntime

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_ParallelDecodePreservesAllExamples(t *testing.T) {
	cache := buildRspecCache(120, 4000)
	cache.Meta.PlurVersion = "parallel-test"
	entry := cache.Files["spec/rubocop/cop/generated_000_spec.rb"]
	entry.Examples[0].ID = "./spec/\"escaped\\世界_spec.rb[1:1]"
	entry.Examples[0].LocationRerunArgument = "./spec/\"escaped\\世界_spec.rb:10"
	cache.Files["spec/\"escaped\\世界_spec.rb"] = entry
	data, err := json.Marshal(struct {
		Files map[string]FileEntry `json:"files"`
		Meta  CacheMeta            `json:"meta"`
		Run   CacheRun             `json:"run"`
	}{cache.Files, cache.Meta, cache.Run})
	require.NoError(t, err)
	data = append([]byte(" \n\t"), data...)
	data = append(data, '\n')
	assert.Equal(t, cache, decodeCacheParallel(data))
}

func TestCache_IgnoresMalformedExampleColumns(t *testing.T) {
	for name, examples := range map[string]string{
		"unequal lengths": `{"id":["x"],"line_number":[],"location_rerun_argument":["spec/x.rb:1"],"runtime_seconds":[1]}`,
		"invalid runtime": `{"id":["x"],"line_number":[1],"location_rerun_argument":["spec/x.rb:1"],"runtime_seconds":["invalid"]}`,
		"wrong shape":     `[{"id":"x","line_number":1,"runtime_seconds":1}]`,
	} {
		t.Run(name, func(t *testing.T) {
			for _, padding := range []string{"", strings.Repeat(" ", 2<<20)} {
				path := filepath.Join(t.TempDir(), "runtime.json")
				data := fmt.Sprintf(`{"meta":{"schema_version":%d},"files":{"spec/x.rb":{"examples":%s}}}`, SchemaVersion, examples)
				require.NoError(t, os.WriteFile(path, []byte(padding+data), 0644))
				assert.Equal(t, NewCache(), LoadCache(path))
			}
		})
	}
}

func TestCache_ParallelDecodeRejectsIncompleteDocument(t *testing.T) {
	cache := buildRspecCache(100, 1000)
	data, err := json.Marshal(cache)
	require.NoError(t, err)
	assert.Nil(t, decodeCacheParallel(data[:len(data)-1]))
	assert.Nil(t, decodeCacheParallel(append(data, []byte(" null")...)))
}

func TestCache_ParallelDecodeNullFile(t *testing.T) {
	data := fmt.Sprintf(`{"meta":{"schema_version":%d},"files":{"spec/null.rb":null}}`, SchemaVersion)
	cache := decodeCacheParallel([]byte(data))
	require.NotNil(t, cache)
	assert.Equal(t, FileEntry{}, cache.Files["spec/null.rb"])
}

func TestCache_ParallelDecodeRejectsDuplicateFiles(t *testing.T) {
	for _, records := range []string{
		`"spec/x.rb":{"source_cwd":"/project"},"spec/x.rb":{"runtime_seconds":2}`,
		`"spec/x.rb":{"runtime_seconds":1},"spec/x.rb":null`,
		`"spec/x.rb":null,"spec/x.rb":{"runtime_seconds":2}`,
		`"spec/x.rb":{"runtime_seconds":"invalid"},"spec/x.rb":{"runtime_seconds":2}`,
	} {
		t.Run(records, func(t *testing.T) {
			data := fmt.Sprintf(`{"meta":{"schema_version":%d},"files":{%s}}`, SchemaVersion, records)
			assert.Nil(t, decodeCacheParallel([]byte(data)))
		})
	}
}

func TestCache_IgnoresPreviousSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"meta":{"schema_version":4},"files":{"spec/x.rb":{"runtime_seconds":1}}}`), 0644))
	assert.Equal(t, NewCache(), LoadCache(path))
}

func TestCache_ParallelDecodePermissiveOptions(t *testing.T) {
	data := fmt.Sprintf("{\"meta\":{\"schema_version\":0,\"schema_version\":%d,\"plur_version\":\"v\xff\"},\"files\":{\"spec/x.rb\":{\"examples\":{\"id\":[\"x\xff\"],\"line_number\":[1],\"location_rerun_argument\":[\"spec/x.rb:1\"],\"runtime_seconds\":[0],\"runtime_seconds\":[1]}}}}", SchemaVersion)
	cache := decodeCacheParallel([]byte(data))
	require.NotNil(t, cache)
	assert.Equal(t, "v\ufffd", cache.Meta.PlurVersion)
	examples := cache.Files["spec/x.rb"].Examples
	require.Len(t, examples, 1)
	assert.Equal(t, "x\ufffd", examples[0].ID)
	assert.Equal(t, float64(1), examples[0].RuntimeSeconds)
}
