# Plur Watch Mode

## Overview

`plur watch` provides automatic test/spec execution when files change. Imagine [guard](https://github.com/guard/guard), but much faster, zero-config (by default), and no gem/ruby setup necessary. It's designed to be a "one stop shop" - just run `plur watch` in any Ruby project and get instant feedback as you code.

It uses a [fast, lean embedded C++ watcher](https://github.com/e-dant/watcher) to monitor file changes and trigger test/spec execution, using the best platform-specific fsevent library. (FSEvents, inotify, ReadDirectoryChangesW, etc.)

## Usage

```bash
# Start watching for file changes
plur watch

# Preview the jobs a file change would trigger
plur watch find lib/foo.rb

# Set custom debounce delay (milliseconds)
plur watch --debounce 250
```

### What Gets Watched

For an RSpec project, the default watch mappings monitor:

- `spec/**/*_spec.rb` - Test files (runs the changed spec)
- `lib/**/*.rb` - Library files (runs corresponding spec)
- `app/**/*.rb` - Rails app files (runs corresponding spec)

Minitest projects use corresponding `test/**/*_test.rb` mappings. Go projects
map Go source changes to package test commands.

Default watch mappings do not include helper files such as
`spec/spec_helper.rb` or `spec/rails_helper.rb`. Add a project-specific
`[[watch]]` rule if helper changes should run tests.

### File Mapping Examples

| Changed File | Runs |
|--------------|------|
| `lib/foo.rb` | `spec/foo_spec.rb` |
| `lib/foo/bar.rb` | `spec/foo/bar_spec.rb` |
| `app/models/user.rb` | `spec/models/user_spec.rb` |
| `app/controllers/posts_controller.rb` | `spec/controllers/posts_controller_spec.rb` |
| `spec/models/user_spec.rb` | `spec/models/user_spec.rb` (itself) |

### Global Exclusions

By default, events from certain directories are ignored to reduce noise:

* `.git/**` - Git internal files
* `node_modules/**` - JavaScript dependencies

These patterns are applied globally before any watch rules are evaluated. You can customize them in `.plur.toml` with the `watch-ignore` option:

```toml
watch-ignore = [".git/**", "node_modules/**", "vendor/**", ".bundle/**"]
```

Or customize a single watch session with the repeatable `--ignore` flag:

```bash
plur watch --ignore ".git/**" --ignore "node_modules/**" --ignore "vendor/**" --ignore ".bundle/**"
```

Setting either `watch-ignore` or `--ignore` replaces the defaults entirely - include `.git/**` and `node_modules/**` if you still want them ignored.

## Platform Support

Watcher binaries are embedded for these platforms:

- macOS ARM64 (Apple Silicon)
- Linux x86_64
- Linux ARM64
- Windows x86_64 (experimental)

Binaries are extracted on first use to `~/.plur/bin/` (or `$PLUR_HOME/bin/`)
and automatically replaced when Plur ships a newer watcher version.

## Process Lifecycle

- Plur tracks direct child jobs and waits for them to exit
- The first Ctrl-C lets Plur and test runners stop normally; a second force-stops remaining jobs
- Without a terminal, SIGINT stops remaining jobs after a short grace period
- All ordinary shutdown paths reap active jobs; SIGKILL cannot and may leave jobs running

### Event Types

Plur considers `create` and `modify` events for a test run (see the effect-type
filter in `internal/watch/controller.go`); events with other effect types are skipped before
the usual ignore and watch-mapping rules are applied.

On macOS and Linux, this makes watch mode driven by **content** changes, not
timestamps. A bare `touch` that only bumps a file's modification time is not
reported as `create` or `modify`, so it does *not* trigger a run. That is
deliberate: modern editors, formatters, build tools, and sync agents churn file
timestamps constantly, and reacting to every mtime bump would make watch mode
far too noisy.

### Debouncing

* Default 30ms delay to batch related changes
* Prevents test runs from overlapping file saves
* Configurable via `--debounce` flag

## Known Issues and Limitations

### Concurrent Output
Watch runs independent work concurrently. A target already running in the same
job is skipped and reported:

```text
[plur] skipped spec/user_spec.rb reason=running
```

When only part of a run overlaps, the free targets still start. Different jobs
do not block each other. A `no_targets = true` run only blocks another
no-targets run of the same job.

Concurrent runs currently share the terminal, which can lead to:

- Interleaved output from different test runs
- Output that is harder to attribute to one run

The run and completion log lines identify the job and targets when debug output
is enabled.

### Current Limitations

- No output panes or per-run output grouping
- Limited to Ruby/Rails conventions by default (custom mappings available via `[[watch]]` config)

See [Watch Configuration](../configuration.md#watch-configuration) for custom file mapping options.

See [Watch Architecture](../architecture/plur-watch-architecture.md) for implementation details.
