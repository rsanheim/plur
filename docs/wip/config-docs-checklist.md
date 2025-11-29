# Configuration Documentation Overhaul

## Problem Summary

**docs/configuration.md documents a config API that doesn't exist:**

| What docs say | What code actually uses |
|---------------|------------------------|
| `[task.rspec]` | `[job.rspec]` |
| `run = "cmd"` | `cmd = ["cmd", "args"]` |
| `test_glob = "..."` | `target_pattern = "..."` |
| `source_dirs = [...]` | (not a field) |
| `[watch.run]` | `[[watch]]` array |

## Actual Config API (verified from Go code + defaults.toml)

### Job Configuration
```toml
[job.<name>]
cmd = ["command", "args", "{{target}}"]  # Command array with target substitution
target_pattern = "spec/**/*_spec.rb"      # Glob pattern for test files
env = ["VAR=value"]                       # Optional environment variables
```

### Watch Configuration
```toml
[[watch]]
name = "rule-name"           # Optional identifier
source = "lib/**/*.rb"       # File pattern to watch
targets = ["spec/{{match}}_spec.rb"]  # Target patterns ({{match}}, {{dir_relative}})
jobs = ["rspec"]             # Jobs to trigger
exclude = [".git/**"]        # Patterns to exclude
```

### Global Settings
```toml
workers = 4
color = true
use = "rspec"  # Default job
```

## Checklist

### Phase 1: Verify Current Implementation
- [ ] Confirm job struct fields in plur/job/job.go
- [ ] Confirm watch struct fields in plur/watch/watch_mapping.go
- [ ] Review defaults.toml for built-in jobs/watch mappings
- [ ] Test a sample config with `plur doctor`

### Phase 2: Update docs/configuration.md
- [ ] Replace all `[task.*]` with `[job.*]`
- [ ] Replace `run = "..."` with `cmd = [...]`
- [ ] Replace `test_glob` with `target_pattern`
- [ ] Remove `source_dirs` references (not a real field)
- [ ] Fix watch config section to use `[[watch]]` array syntax
- [ ] Update all code examples to match actual API
- [ ] Keep convention-based patterns section (still valid concept)

### Phase 3: Update Other Docs
- [ ] docs/usage.md - verify config examples (currently just `use=` which is fine)
- [ ] docs/getting-started.md - verify config examples (currently just `use=` which is fine)
- [ ] docs/features/watch-mode.md - update any config references

### Phase 4: Archive Files
- [ ] docs/archive/* - leave as historical, don't update

### Phase 5: Verification
- [ ] Run `script/check-links`
- [ ] Run `script/docs` and verify rendered output
- [ ] Test example configs from docs actually work with plur

## Files To Modify

1. **docs/configuration.md** - Complete rewrite of config sections

## Source of Truth Files (read-only reference)

* `plur/job/job.go` - Job struct definition
* `plur/watch/watch_mapping.go` - WatchMapping struct
* `plur/autodetect/defaults.toml` - Built-in defaults
* `.plur.toml` - Working example config
* `CLAUDE.md` - Already has correct syntax
