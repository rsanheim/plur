package testruntime

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"runtime"
	"sync"
	"sync/atomic"
)

// Scan file boundaries while a bounded worker pool decodes their contents.
// Offsets refer to the original input, so records do not need to be copied.
func decodeCacheParallel(data []byte) *Cache {
	type task struct {
		data  []byte
		entry *FileEntry
	}
	jobs := make(chan task, 64)
	var group sync.WaitGroup
	var failed atomic.Bool
	duplicateFiles := false
	for range min(4, runtime.GOMAXPROCS(0)) {
		group.Go(func() {
			for job := range jobs {
				if err := json.Unmarshal(job.data, job.entry, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true)); err != nil {
					failed.Store(true)
				}
			}
		})
	}
	unmarshaler := json.UnmarshalFromFunc(func(dec *jsontext.Decoder, out **FileEntry) error {
		value, err := dec.ReadValue()
		if err != nil {
			return err
		}
		end := dec.InputOffset()
		start := end - int64(len(value))
		duplicateFiles = duplicateFiles || *out != nil
		// A duplicate file key gets a separate result; workers never share a destination.
		entry := new(FileEntry)
		*out = entry
		jobs <- task{data: data[start:end], entry: entry}
		return nil
	})
	var raw *struct {
		Meta  CacheMeta             `json:"meta"`
		Run   CacheRun              `json:"run"`
		Files map[string]*FileEntry `json:"files"`
	}
	err := json.Unmarshal(data, &raw, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true), json.WithUnmarshalers(unmarshaler))
	close(jobs)
	group.Wait()
	if err != nil || raw == nil || failed.Load() {
		return nil
	}
	if duplicateFiles {
		// Let json/v2 merge repeated file records using the same rules as serial loads.
		var cache *Cache
		if err := json.Unmarshal(data, &cache, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true)); err != nil {
			return nil
		}
		return cache
	}
	cache := &Cache{Meta: raw.Meta, Run: raw.Run, Files: make(map[string]FileEntry, len(raw.Files))}
	for path, entry := range raw.Files {
		cache.Files[path] = *entry
	}
	return cache
}
