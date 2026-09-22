# Plur Watch Mode

## Overview

`plur watch` reruns tests when files change. Built-in mappings connect source
files to their tests, and you can add mappings for your project.

Plur bundles [e-dant/watcher](https://github.com/e-dant/watcher) to monitor changes
using the operating system's file events.

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

Plur ignores events from these directories by default:

* `.git/**` - Git internal files
* `node_modules/**` - JavaScript dependencies

Plur applies these exclusions before matching watch rules. Set `watch-ignore` in
`.plur.toml` to change them:

```toml
watch-ignore = [".git/**", "node_modules/**", "vendor/**", ".bundle/**"]
```

Or customize a single watch session with the repeatable `--ignore` flag:

```bash
plur watch --ignore ".git/**" --ignore "node_modules/**" --ignore "vendor/**" --ignore ".bundle/**"
```

Both `watch-ignore` and `--ignore` replace the default list. Include `.git/**` and
`node_modules/**` to keep ignoring them.

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
- On shutdown, Plur waits for child jobs to exit; SIGKILL prevents cleanup and may leave jobs running

## File Changes

Plur runs tests for `create` and `modify` events that match a watch rule.
Other event types are ignored.

On macOS and Linux, `touch` alone does not trigger a run. Change the file's
contents to trigger its watch rules.

### Debouncing

Plur batches changes over a 30ms window before running tests. Set `--debounce`
to adjust the delay in milliseconds.

## Known Issues and Limitations

### Concurrent Output
Watch mode runs independent jobs and targets concurrently. If a target is already
running in the same job, Plur skips it and reports:

```text
[plur] skipped spec/user_spec.rb reason=running
```

If only some targets are already running, Plur starts the rest. Different jobs
do not block each other. A `no_targets = true` run only blocks another run
without targets in the same job.

Concurrent runs share the terminal, so their output can interleave. Enable debug
logging to see the job and targets when each run starts and finishes.

See [Watch Configuration](../configuration.md#watch-configuration) for custom
mappings and [Watch Architecture](../architecture/plur-watch-architecture.md)
for implementation details.
