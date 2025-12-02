# Refactor: Consolidate watch domain logic into watch package

## Summary

* Move watch domain logic (directory filtering, ignore patterns, job execution) from main package to `watch/` package
* Rename command files (`doctor.go` → `cmd_doctor.go`, `watch.go` → `cmd_watch.go`) for clarity
* Add global `--ignore` and `--use` flags at the `WatchCmd` level (shared by all subcommands)
* Introduce default ignore patterns (`.git/**`, `node_modules/**`)

## Motivation

The watch command implementation had grown organically with domain logic mixed into CLI orchestration. This made testing difficult and obscured the architectural boundaries. This refactor cleanly separates:

* **CLI layer** (`cmd_watch.go`): Kong command definitions, configuration loading, orchestration
* **Domain layer** (`watch/watcher.go`): Reusable logic for path validation, pattern matching, job execution

## Changes

### Architectural

* `FilterDirectories()` → moved to `watch/watcher.go`
  * Validates paths stay within project root (security)
  * Deduplicates symlinks pointing to same location
  * Removes subdirectories when parent is already watched
* `IsIgnored()` → moved to `watch/watcher.go`
* `ExecuteJob()` / `RunCommand()` → moved to `watch/watcher.go`
* `DefaultIgnorePatterns` constant added

### CLI Flags

```
# Before (flag on subcommand only)
plur watch run --use=rspec

# After (flags on parent, inherited by subcommands)
plur watch --use=rspec --ignore=vendor/** run
```

### File Renames

* `plur/doctor.go` → `plur/cmd_doctor.go`
* `plur/watch.go` → `plur/cmd_watch.go`
* `plur/watch_test.go` → `plur/watch/watcher_test.go`

## Test Plan

* [ ] Existing integration specs pass (`spec/integration/plur_watch/`)
* [ ] New ignore spec validates `--ignore` flag behavior
* [ ] Complexity test ensures path processing stays O(n)
* [ ] `bin/rake` passes all tests and lints
