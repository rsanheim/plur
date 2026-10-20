# Plur Watch Mode

`plur watch` reruns your tests when files change. It knows the usual Ruby
layouts out of the box, and you can add your own mappings in `.plur.toml`.

```bash
plur watch                    # start watching
plur watch --debounce 250     # wait 250ms for changes to settle (default is 30ms)
plur watch find lib/foo.rb    # show what a change to lib/foo.rb would run
```

## While It's Running

| Type | To |
|------|----|
| Enter | Run all tests |
| `reload` | Restart plur, picking up any config changes |
| `debug` | Toggle debug output |
| `help` | List these commands |
| `exit` or Ctrl-C | Quit |

Ctrl-C once lets running tests wrap up. Press it again to force-stop them.

With Minitest, Enter runs every test file your job matches (minus its
excludes), including files you just added.

## What Gets Watched

For RSpec projects:

| Changed File | Runs |
|--------------|------|
| `lib/foo/bar.rb` | `spec/foo/bar_spec.rb` |
| `app/models/user.rb` | `spec/models/user_spec.rb` |
| `spec/models/user_spec.rb` | itself |

Minitest projects work the same way with `test/**/*_test.rb`, and Go projects
run the changed package's tests.

Helpers like `spec/spec_helper.rb` aren't watched by default. Add a `[[watch]]`
rule if you want changes there to run tests. See
[Watch Configuration](../configuration.md#watch-configuration).

## Ignoring Files

Plur ignores `.git/**` and `node_modules/**` by default. To change that, set
`watch-ignore` in `.plur.toml`:

```toml
watch-ignore = [".git/**", "node_modules/**", "vendor/**"]
```

Or use `--ignore` for a single session:

```bash
plur watch --ignore ".git/**" --ignore "node_modules/**" --ignore "vendor/**"
```

Either one replaces the defaults, so list `.git/**` and `node_modules/**` again
if you still want them ignored.

## Good to Know

- Creating or editing a file triggers a run. A bare `touch` doesn't, so change
  the contents.
- If a test is already running when its file changes again, plur skips it
  instead of queueing it up:

  ```text
  [plur] skipped spec/user_spec.rb reason=running
  ```

- Different jobs run side by side, so their output can interleave.

## Platform Support

File watching is powered by the excellent
[e-dant/watcher](https://github.com/e-dant/watcher). Plur bundles it for macOS (Apple Silicon), Linux x86_64 and
ARM64, and Windows x86_64 (experimental). It gets unpacked to `~/.plur/bin/`
the first time you run `plur watch`.

For how it all works under the hood, see
[Watch Architecture](../architecture/plur-watch-architecture.md).
