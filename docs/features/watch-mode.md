# Plur Watch Mode

`plur watch` watches project files and runs the tests that match a changed
file. It is meant for local feedback while editing.

## Basic Usage

```bash
# Start watching for file changes
plur watch

# Preview what a changed file would run
plur watch find spec/models/user_spec.rb

# Use JSON for scripts or agents
plur watch find --format=json spec/models/user_spec.rb
```

Useful options:

```bash
# Set debounce delay in milliseconds
plur watch --debounce 250

# Auto-exit after a timeout
plur watch --timeout 60

# Ignore noisy paths for this watch session
plur watch --ignore "vendor/**" --ignore "tmp/**"
```

For stable `watch find --format=json` fields and exit codes, see
[Output Contracts](../output-contracts.md#watch-find).

## What Gets Watched

By default, Plur watches common Ruby/Rails paths when they exist:

- `spec/**/*_spec.rb` - test files run themselves
- `lib/**/*.rb` - library files map to matching specs
- `app/**/*.rb` - Rails app files map to matching specs

Shared helper files only trigger tests when a watch rule maps them to existing
targets. Use `plur watch find spec/spec_helper.rb` to check what a change would
run in the current project. When no rule matches a shared helper, Plur prints a
hint to add a `[[watch]]` mapping if that change should run tests.

## File Mapping Examples

| Changed File | Runs |
| --- | --- |
| `lib/foo.rb` | `spec/foo_spec.rb` |
| `lib/foo/bar.rb` | `spec/foo/bar_spec.rb` |
| `app/models/user.rb` | `spec/models/user_spec.rb` |
| `app/controllers/posts_controller.rb` | `spec/controllers/posts_controller_spec.rb` |
| `spec/models/user_spec.rb` | `spec/models/user_spec.rb` |

If a matching target does not exist, watch preview reports the missing target
and exits 2:

```bash
plur watch find lib/missing.rb
```

## Ignore Paths

By default, events from `.git/**` and `node_modules/**` are ignored. Use
`--ignore` for a single watch session:

```bash
plur watch --ignore "vendor/**" --ignore "tmp/**"
```

Project-wide watch mappings and ignore rules live in `.plur.toml`. See
[Watch Configuration](../configuration.md#watch-configuration).

## Limitations

- Watch mode runs matching jobs serially.
- Rapid save bursts can still make terminal output busy.
- Built-in mappings follow Ruby/Rails conventions; custom mappings require
  `[[watch]]` config.

## Troubleshooting

### Watcher Binary Not Found

The watcher binary should auto-extract to `~/.plur/bin/` or
`$PLUR_HOME/bin/`. Run:

```bash
plur doctor
```

### Tests Not Running On File Change

Check the mapping before starting a live watch:

```bash
plur watch find path/to/changed_file.rb
```

Also check:

- the file is not ignored by gitignore or `--ignore`
- the expected target file exists
- the change modifies file content; metadata-only changes such as `touch` may
  not trigger events

Use debug output only for local troubleshooting:

```bash
plur --debug watch
```

## Architecture Details

For implementation details about watcher processes, embedded binaries,
debouncing, and event flow, see [Watch Architecture](../architecture/plur-watch-architecture.md).
