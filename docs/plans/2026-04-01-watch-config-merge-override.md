# Watch Config Merge: Name-Based Override

## Context

Branch `fix/watch-config-merge` changed watch mapping merges from "user replaces all defaults" to "user appends to defaults" via `slices.Concat`. This fixed the bug where adding one custom `[[watch]]` wiped all built-in mappings, but introduced a gap: users can't override or remove a specific built-in watch.

**Solution:** Use a `map[string]WatchMapping` keyed by `mergeKey()` during resolve/merge time to enforce identity semantics. Named user watches with a matching key replace built-ins; unnamed/new watches are additive. The map is flattened to `[]WatchMapping` for all downstream consumers (processors, handlers, find) — they don't change at all.

## Design

### `mergeKey()` method on `WatchMapping`

```go
func (w WatchMapping) mergeKey() string {
    if w.Name != "" {
        return "name:" + w.Name
    }
    return fmt.Sprintf("source:%s|targets:%v|jobs:%v|ignore:%v|reload:%v",
        w.Source, w.Targets, w.Jobs, w.Ignore, w.Reload)
}
```

* Named watches: keyed by name (enables override by name)
* Unnamed watches: keyed by full field composite (only exact duplicates collapse)

### `MergeWatches` function

```go
func MergeWatches(builtins, user []WatchMapping) []WatchMapping {
    merged := make(map[string]WatchMapping)
    order := make([]string, 0)

    // Add builtins first, tracking insertion order
    for _, b := range builtins {
        key := b.mergeKey()
        if _, exists := merged[key]; !exists {
            order = append(order, key)
        }
        merged[key] = b
    }

    // Add user watches — overwrite by key (name match = replace)
    for _, u := range user {
        key := u.mergeKey()
        if _, exists := merged[key]; !exists {
            order = append(order, key)
        }
        merged[key] = u
    }

    // Flatten in insertion order
    result := make([]WatchMapping, 0, len(order))
    for _, key := range order {
        result = append(result, merged[key])
    }
    return result
}
```

The map enforces identity during merge. Insertion order is preserved via the `order` slice. When a user watch has the same key as a built-in, the built-in's slot gets overwritten with the user's version — keeping the built-in's position in the list.

### Data flow

```
getWatchesForJob(name) → []WatchMapping (built-ins filtered by job)
cli.WatchMappings      → []WatchMapping (user config)
        ↓
MergeWatches(builtins, user) → map internally → []WatchMapping
        ↓
processors, handlers, find (unchanged — just iterate the slice)
```

## Override behavior by scenario

| Scenario | User config | Result |
|----------|-------------|--------|
| Add new mapping | `name = "custom"` with new source | Appended after builtins |
| Override built-in | `name = "lib-to-spec"` with different targets | Replaces built-in, keeps its position |
| Unnamed new mapping | No name, unique source/targets/jobs | Appended after builtins |
| Exact unnamed duplicate | No name, identical fields to a built-in | Collapsed to one (via composite key) |
| No user watches | (empty) | All builtins returned unchanged |

## Implementation steps

### Step 1: Add `mergeKey()` and `MergeWatches` to `watch/watch_mapping.go`

* `mergeKey()` — unexported method on `WatchMapping`
* `MergeWatches(builtins, user []WatchMapping) []WatchMapping` — exported function

### Step 2: Add data-driven unit tests in `watch/watch_mapping_test.go`

**`TestMergeWatches`** table-driven test:

| Case | Description |
|------|-------------|
| both empty | nil inputs → empty output |
| builtins only | no user watches → all builtins returned |
| user only | no builtins → all user watches returned |
| additive, new name | user watch with unique name → appended |
| additive, unnamed | unnamed user watch → appended |
| override by name | user `name = "lib-to-spec"` replaces built-in `lib-to-spec` |
| override preserves position | overridden watch keeps built-in's position in order |
| override one, keep others | 3 builtins, 1 overridden → 2 original + 1 replaced |
| mix of override and additive | 1 override + 1 new name → correct merge |
| exact unnamed duplicate collapses | two unnamed watches with identical fields → one kept |

**`TestMergeKey`**:

| Case | Description |
|------|-------------|
| named watch | returns `"name:<name>"` |
| unnamed watch | returns composite of all fields |
| same name, different fields | same key (name wins) |
| same fields, no name | same key (field hash matches) |
| different fields, no name | different keys |

### Step 3: Update call sites

**`internal/runtime/config.go`** — in `BuildRuntimeConfig`, the watch fallback path currently either uses user watches or falls back to builtin watches. Replace this with `MergeWatches(builtinWatches, userWatches)` so user watches override by name rather than replacing entirely.

### Step 4: Update/add call-site tests

**`internal/runtime/config_test.go`** — add test where user watch has `Name: "lib-to-spec"` with different targets, verify the built-in is replaced.

### Step 5: Add `ProcessPath` behavior tests in `watch/processor_test.go`

**`TestEventProcessorMergedWatches`** — data-driven test exercising `ProcessPath` with watch slices that represent already-merged output (the way processors receive them):

| Case | Description |
|------|-------------|
| same source, different targets, same job | both targets produced |
| same source, same target, same job | deduplicated to one |
| same source, different jobs | each job gets its target |
| user watches only | works without builtins |
| builtins only | regression guard |
| nested path, additive | `{{match}}` resolves correctly for both |
| non-matching file | empty result |

### Step 6: Add integration test

**`spec/integration/watch/watch_integration_spec.rb`** — test that a `.plur.toml` with `name = "lib-to-spec"` pointing to different targets overrides the built-in when running `plur watch find lib/calculator.rb`.

## Critical files

* `watch/watch_mapping.go` — add `mergeKey()` and `MergeWatches`
* `watch/watch_mapping_test.go` — unit tests
* `watch/processor_test.go` — behavior tests for `ProcessPath` with merged watches
* `internal/runtime/config.go` — update watch fallback to use `MergeWatches`
* `internal/runtime/config_test.go` — add override test
* `spec/integration/watch/watch_integration_spec.rb` — integration test

## Verification

```bash
go test ./watch/ -run "TestMergeKey|TestMergeWatches" -v
go test ./watch/ -run TestEventProcessorMergedWatches -v
go test ./... -v
bin/rake test
bin/rake
```
