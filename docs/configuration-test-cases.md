# Configuration Test Cases

Manual verification checklist for plur configuration behavior. Each case specifies input, expected behavior, and verification status.

*Verified on: 2026-04-02, plur version 0.45.0-dev-65071c8*

**Legend:** `[x]` verified passing, `[!]` unexpected behavior (see notes)

---

## 1. Config File Loading & Precedence

### 1.1 Discovery and loading

* [x] **No config files exist** — plur uses built-in defaults
  * *Input:* Empty directory with `spec/` and a spec file, no `.plur.toml`, no `~/.plur.toml`
  * *Expected:* `plur --dry-run` succeeds, autodetects rspec, cmd includes `bundle exec rspec`
  * *Actual:* Confirmed

* [x] **Project `.plur.toml` only** — loaded and applied
  * *Input:* `.plur.toml` with `workers = 2`, spec file present
  * *Expected:* Config is loaded (verify with `plur doctor`)
  * *Note:* Worker count is adjusted down to file count, so 1 file always shows 1 worker in dry-run output. Use `plur doctor` to see configured value.

* [x] **Home `~/.plur.toml` only** — loaded and applied
  * *Input:* `PLUR_CONFIG_FILE` pointing to a file with `workers = 3`, no project config
  * *Expected:* Shows 3 workers in doctor output
  * *Actual:* Confirmed via `PLUR_CONFIG_FILE` simulation

* [x] **Both home and project config** — later file wins
  * *Input:* `PLUR_CONFIG_FILE` with `workers = 3`, `.plur.toml` with `workers = 5`
  * *Expected:* `PLUR_CONFIG_FILE` wins (loaded last in Kong sequence)
  * *Actual:* Shows 3 (PLUR_CONFIG_FILE value), confirming last-loaded-wins semantics

### 1.2 PLUR_CONFIG_FILE

* [x] **PLUR_CONFIG_FILE set to valid file** — takes precedence over project config
  * *Input:* `.plur.toml` with `workers = 2`, env file with `workers = 7`
  * *Expected:* Doctor shows 7 workers
  * *Actual:* Confirmed

* [x] **PLUR_CONFIG_FILE set to nonexistent file** — exits immediately with error
  * *Input:* `PLUR_CONFIG_FILE=/no/such/file`
  * *Expected:* Exit 1, stderr includes "does not exist or is not readable"
  * *Actual:* `Error: Config file specified in PLUR_CONFIG_FILE does not exist or is not readable: /no/such/file`

* [x] **PLUR_CONFIG_FILE set to empty string** — ignored, normal loading
  * *Input:* `PLUR_CONFIG_FILE=""`
  * *Expected:* Normal behavior, no error
  * *Actual:* Exits 0, normal behavior

### 1.3 CLI flag precedence

* [x] **CLI flag overrides config file** — `-n` wins over `workers` in TOML
  * *Input:* `.plur.toml` with `workers = 2`, 8 spec files, `plur -n 4 --dry-run`
  * *Expected:* Shows 4 workers
  * *Actual:* `Running 4 specs [rspec] in parallel using 4 workers` (note: would show 2 without flag)

* [x] **CLI flag overrides env var** — `-n` wins over `PARALLEL_TEST_PROCESSORS`
  * *Input:* `PARALLEL_TEST_PROCESSORS=8`, 8 spec files, `plur -n 2 --dry-run`
  * *Expected:* Shows 2 workers
  * *Actual:* Confirmed

* [x] **Config file overrides env var** — `workers` in TOML wins over `PARALLEL_TEST_PROCESSORS`
  * *Input:* `.plur.toml` with `workers = 3`, `PARALLEL_TEST_PROCESSORS=8`, 8 spec files
  * *Expected:* Shows 3 workers
  * *Actual:* Confirmed

### 1.4 `-C` flag interaction

* [x] **`-C` loads config from target directory** — not current directory
  * *Input:* Current dir has `.plur.toml` with `workers = 2`, target dir has `.plur.toml` with `workers = 5`
  * *Expected:* `plur -C target --dry-run` shows 5 workers
  * *Actual:* Confirmed with 5 spec files

* [x] **`-C` ignores config in current directory**
  * *Input:* Current dir has `.plur.toml` with `workers = 99`, target dir has no config
  * *Expected:* Uses auto-detected defaults, not 99
  * *Actual:* Doctor shows 18 workers (CPU count), confirming current-dir config ignored

* [x] **`-C` to nonexistent directory** — clear error
  * *Input:* `plur -C /no/such/dir`
  * *Expected:* Exit 1, stderr includes "failed to change directory"
  * *Actual:* `Error: failed to change directory to /no/such/dir: chdir /no/such/dir: no such file or directory`

* [x] **`-C` without argument** — error
  * *Input:* `plur -C`
  * *Expected:* Exit 1, error about missing argument
  * *Actual:* `Error: -C flag requires a directory argument`

### 1.5 Empty and minimal config files

* [x] **Empty config file** — treated as valid, defaults used
  * *Input:* `.plur.toml` is empty (0 bytes)
  * *Expected:* `plur --dry-run` succeeds with defaults
  * *Actual:* Succeeds

* [x] **Comments-only config file** — treated as valid
  * *Input:* `.plur.toml` contains only `# this is a comment`
  * *Expected:* `plur --dry-run` succeeds with defaults
  * *Actual:* Succeeds

---

## 2. Job Configuration

### 2.1 Builtin jobs

* [x] **Builtin rspec job available by default** — no config needed
  * *Input:* Directory with `spec/example_spec.rb`, no `.plur.toml`
  * *Expected:* `plur --dry-run` runs with rspec, cmd includes `bundle exec rspec`
  * *Actual:* Confirmed

* [x] **Builtin minitest job available by default**
  * *Input:* Directory with `test/example_test.rb`, no `.plur.toml`
  * *Expected:* `plur --dry-run` runs with minitest
  * *Actual:* `Running 1 test [minitest]`

* [x] **Builtin go-test job available by default**
  * *Input:* Directory with `go.mod` and `example_test.go`
  * *Expected:* `plur --dry-run` runs with go-test
  * *Actual:* `Running 1 test [go-test]`

### 2.2 User job overlay on builtins

* [x] **Override builtin command** — user cmd replaces builtin cmd
  * *Input:* `.plur.toml` with `[job.rspec]` and `cmd = ["bin/rspec"]`
  * *Expected:* `plur doctor` shows command as `[bin/rspec]`, framework still `rspec`
  * *Actual:* Doctor shows `Active Job: rspec`, `Command: [bin/rspec]`

* [x] **Override builtin — framework tracks as inherited** — debug output shows which fields inherited
  * *Input:* `.plur.toml` with `[job.rspec]` and `cmd = ["bin/rspec"]` (no framework specified)
  * *Expected:* Framework is `rspec` (inherited from builtin), `--debug` output shows `framework=true`
  * *Actual:* `job inherited defaults job="rspec" cmd=false env=false framework=true target_pattern=true`

* [x] **Override builtin target_pattern** — user pattern replaces builtin
  * *Input:* `.plur.toml` with `[job.rspec]` and `target_pattern = "spec/**/*_spec.rb"`
  * *Expected:* `plur doctor` shows the user-specified pattern
  * *Actual:* `Target Patterns: spec/**/*_spec.rb`

### 2.3 Custom jobs

* [!] **Custom job with passthrough framework** — defaults to passthrough
  * *Input:* `.plur.toml` with `[job.lint]` and `cmd = ["rubocop"]`
  * *Expected:* `plur -u lint --dry-run` runs rubocop
  * *Actual:* ERROR logged: `job "lint" has no target_pattern and framework "passthrough" has no detect patterns` — **but exit code is 0**. A custom passthrough job without `target_pattern` can't discover files, so it errors at runtime but the error isn't propagated. This is likely a bug — should either exit nonzero or require `target_pattern` for passthrough jobs.

* [x] **Custom job with explicit framework**
  * *Input:* `.plur.toml` with `[job.custom]`, `cmd = ["my-rspec"]`, `framework = "rspec"`, `target_pattern = "spec/**/*_spec.rb"`
  * *Expected:* `plur -u custom --dry-run` uses rspec output parsing with custom command
  * *Actual:* Shows `my-rspec` in dry-run command output

### 2.4 Job selection via `use`

* [x] **`use` in config selects job** — skips autodetection
  * *Input:* `.plur.toml` with `use = "minitest"`, both `spec/` and `test/` dirs exist
  * *Expected:* `plur --dry-run` uses minitest (not rspec despite priority)
  * *Actual:* `Running 1 test [minitest]`

* [x] **`-u` CLI flag overrides `use` in config**
  * *Input:* `.plur.toml` with `use = "minitest"`, run `plur -u rspec --dry-run`
  * *Expected:* Uses rspec
  * *Actual:* `Running 1 spec [rspec]`

* [x] **`use` references nonexistent job** — clear error
  * *Input:* `.plur.toml` with `use = "nope"`
  * *Expected:* Exit 1, error includes "job 'nope' not found" and lists available jobs
  * *Actual:* `job 'nope' not found. Available jobs: go-test, minitest, rspec`

### 2.5 Job validation failures

* [x] **Custom job without command** — rejected at config time
  * *Input:* `.plur.toml` with `[job.broken]` and `framework = "rspec"` (no cmd)
  * *Expected:* Exit 1, error includes `job "broken" must define a command`
  * *Actual:* `configuration error in [/.../plur.toml]: job "broken" must define a command`

* [x] **Job with unknown framework** — rejected at config time
  * *Input:* `.plur.toml` with `[job.bad]`, `cmd = ["echo"]`, `framework = "unknown"`
  * *Expected:* Exit 1, error includes `unknown framework "unknown"`
  * *Actual:* `job "bad" has unknown framework "unknown"`

---

## 3. Watch Configuration

### 3.1 Valid watch mappings

* [x] **User-defined watch mapping loaded**
  * *Input:* `[[watch]]` with `source = "lib/**/*.rb"`, `targets = ["spec/{{match}}_spec.rb"]`, `jobs = ["rspec"]`
  * *Expected:* `plur watch find lib/models/user.rb` shows the user mapping and resolved target
  * *Actual:* Shows `found rules name=lib-to-spec` and `found files files=spec/models/user_spec.rb`

* [x] **Multiple watch mappings** — all loaded
  * *Input:* Two `[[watch]]` entries (lib-watch and app-watch)
  * *Expected:* Each matches its source pattern independently
  * *Actual:* `lib/foo.rb` matches lib-watch, `app/bar.rb` matches app-watch

* [x] **Watch with name field** — name appears in output
  * *Input:* `[[watch]]` with `name = "lib-watch"`
  * *Expected:* Output includes the name
  * *Actual:* `found rules name=lib-watch`

* [x] **Watch with ignore patterns** — ignored files don't trigger
  * *Input:* `[[watch]]` with `source = "**/*.rb"`, `ignore = ["vendor/**"]`
  * *Expected:* `plur watch find vendor/bar.rb` shows 0 rules
  * *Actual:* `found rules count=0` (exit 2)

* [x] **Watch with reload flag** — parses without error
  * *Input:* `[[watch]]` with `reload = true`
  * *Expected:* Config loads, watch find works (reload effect only visible during `watch run`)
  * *Actual:* Parses and finds targets normally

### 3.2 Builtin watch fallback

* [x] **No user watches, rspec detected** — builtin rspec watches used
  * *Input:* `.plur.toml` with `use = "rspec"` and `[job.rspec]` but no `[[watch]]`
  * *Expected:* `plur watch find lib/foo.rb` maps to `spec/foo_spec.rb` via builtin `lib-to-spec`
  * *Actual:* Shows `found rules name=lib-to-spec` and `found files files=spec/foo_spec.rb`

* [!] **No user watches, no detectable framework** — errors instead of gracefully showing no watches
  * *Input:* Empty project, no spec/test dirs, no config
  * *Expected:* Shows no mappings, no crash
  * *Actual:* `watch find` exits 1 with `failed to select watch job: no default spec/test files found`. The `selectJobFromRuntimeConfig` call in `watch_find.go` runs before the empty-watches check. `buildRuntimeConfig` correctly falls back to empty watches, but `watch find` then tries to select a job and fails. Minor issue — could check for empty watches first.

### 3.3 Watch validation failures

* [x] **Watch references undefined job** — rejected at config time
  * *Input:* `[[watch]]` with `jobs = ["nonexistent"]`
  * *Expected:* Exit 1, error includes `watch "..." references undefined job "nonexistent"`
  * *Actual:* `configuration error in [/.../.plur.toml]: watch "bad-watch" references undefined job "nonexistent"`

* [x] **Watch with invalid target template** — rejected at config time
  * *Input:* `[[watch]]` with `targets = ["spec/{{invalid_token}}_spec.rb"]`
  * *Expected:* Exit 1, error includes "invalid target template" and lists available tokens
  * *Actual:* Error lists all 8 available tokens

* [x] **Watch with unclosed template braces** — rejected
  * *Input:* `[[watch]]` with `targets = ["spec/{{match"]`
  * *Expected:* Exit 1, error about invalid template
  * *Actual:* `failed to parse template "spec/{{match": template: target:1: unclosed action`

### 3.4 Watch edge cases

* [x] **Watch with empty jobs array** — valid but triggers nothing
  * *Input:* `[[watch]]` with `jobs = []`
  * *Expected:* `plur watch find` shows mapping with `jobs=[]` but exit 2 (nothing to run)
  * *Actual:* `found rules name=empty-jobs source=lib/**/*.rb jobs=[] target="[source file]"` — exit 2

* [x] **Watch without targets** — source file itself is the target
  * *Input:* `[[watch]]` with `source = "spec/**/*_spec.rb"`, `jobs = ["rspec"]`, no `targets`
  * *Expected:* `plur watch find spec/foo_spec.rb` shows the source file as target
  * *Actual:* `found rules name=self-target ... target="[source file]"`, `found files files=spec/foo_spec.rb`

---

## 4. Runtime Config Validation

### 4.1 Error message format

* [x] **Validation errors include source files** — config file paths in error
  * *Input:* `.plur.toml` with invalid watch job reference
  * *Expected:* Error message includes the config file path
  * *Actual:* `configuration error in [/full/path/.plur.toml]: watch "bad-watch" references undefined job "nonexistent"`

* [x] **Same error for spec and doctor** — both fail through AfterApply
  * *Input:* `.plur.toml` with invalid config
  * *Expected:* Both `plur spec --dry-run` and `plur doctor` show the same error
  * *Actual:* Identical error messages from both commands

### 4.2 Validation scope

* [x] **All jobs validated, not just selected one** — even unused jobs checked
  * *Input:* `.plur.toml` with `use = "rspec"`, `[job.rspec]` valid, `[job.broken]` missing cmd
  * *Expected:* Exit 1, error about broken job even though rspec is selected
  * *Actual:* `configuration error in [/.../.plur.toml]: job "broken" must define a command`

* [x] **All watches validated** — even if no watch command is used
  * *Input:* `.plur.toml` with valid job but watch referencing undefined job
  * *Expected:* `plur spec --dry-run` fails (validation happens in AfterApply before any command)
  * *Actual:* Confirmed — spec command fails with watch validation error

---

## 5. Framework Detection

### 5.1 Autodetection priority

* [x] **Both spec/ and test/ exist** — rspec wins (higher priority)
  * *Input:* Directory with both `spec/example_spec.rb` and `test/example_test.rb`
  * *Expected:* `plur --dry-run` autodetects rspec
  * *Actual:* Confirmed

* [x] **Only test/ exists** — minitest detected
  * *Input:* Directory with only `test/example_test.rb`
  * *Expected:* `plur --dry-run` autodetects minitest
  * *Actual:* `Running 1 test [minitest]`

* [x] **Only go test files exist** — go-test detected
  * *Input:* Directory with `go.mod` and `example_test.go`
  * *Expected:* `plur --dry-run` autodetects go-test
  * *Actual:* `Running 1 test [go-test]`

### 5.2 Pattern inference

* [x] **Spec file pattern infers rspec** — `spec/foo_spec.rb` → rspec
  * *Input:* `plur spec/foo_spec.rb --dry-run` (file exists)
  * *Expected:* Selects rspec framework
  * *Actual:* `Running 1 spec [rspec]`

* [x] **Test file pattern infers minitest** — `test/foo_test.rb` → minitest
  * *Input:* `plur test/foo_test.rb --dry-run` (file exists)
  * *Expected:* Selects minitest framework
  * *Actual:* `Running 1 test [minitest]`

* [x] **Directory pattern infers framework** — `spec/models` → rspec
  * *Input:* `plur spec/models --dry-run` (directory with spec files)
  * *Expected:* Selects rspec
  * *Actual:* `Running 1 spec [rspec]`

* [x] **Glob pattern infers framework** — `spec/**/*_spec.rb` → rspec
  * *Input:* `plur "spec/**/*_spec.rb" --dry-run` (matching files exist)
  * *Expected:* Selects rspec
  * *Actual:* `Running 1 spec [rspec]`

### 5.3 Detection failures

* [x] **No test files found** — clear error
  * *Input:* Empty directory, run `plur --dry-run`
  * *Expected:* Exit 1, error "no default spec/test files found using default patterns"
  * *Actual:* Confirmed

* [x] **Mixed framework patterns** — error listing frameworks
  * *Input:* `plur spec/foo_spec.rb test/bar_test.rb`
  * *Expected:* Exit 1, error "explicit patterns match multiple frameworks (minitest, rspec)"
  * *Actual:* Confirmed

* [x] **Non-matching patterns fall back to autodetect**
  * *Input:* `spec/` exists with spec files, run `plur README.md --dry-run`
  * *Expected:* Falls back to rspec autodetect (README.md doesn't match any framework)
  * *Actual:* `Running 1 spec [rspec]`

---

## 6. TOML Parsing

### 6.1 Syntax errors

* [x] **Invalid TOML syntax** — immediate error
  * *Input:* `.plur.toml` with `workers =` (incomplete value)
  * *Expected:* Exit 1, error about TOML parse failure
  * *Actual:* `Configuration error: .plur.toml: toml: line 1 (last key "workers"): expected value but found '\n' instead`

* [x] **Duplicate keys** — TOML parser rejects
  * *Input:* `.plur.toml` with `workers = 2` twice
  * *Expected:* Exit 1, error includes "already been defined"
  * *Actual:* `Key 'workers' has already been defined.`

* [x] **Duplicate tables** — TOML parser rejects
  * *Input:* `.plur.toml` with `[job.rspec]` defined twice
  * *Expected:* Exit 1, error includes "has already been defined"
  * *Actual:* `Key 'job.rspec' has already been defined.`

### 6.2 Type mismatches

* [x] **String where int expected** — Kong rejects
  * *Input:* `.plur.toml` with `workers = "hello"`
  * *Expected:* Exit 1, error about type mismatch
  * *Actual:* `--workers: expected a valid 64 bit int but got "hello"`

* [x] **Int where bool expected** — Kong rejects
  * *Input:* `.plur.toml` with `color = 42`
  * *Expected:* Exit 1, error about type mismatch
  * *Actual:* `--color: expected bool but got '*' (int64)`

### 6.3 Unknown keys

* [x] **Unknown top-level key** — logged in debug, not an error
  * *Input:* `.plur.toml` with `bogus_key = true`
  * *Expected:* `plur --debug --dry-run` succeeds, debug output mentions unknown key
  * *Actual:* `unknown config keys file=".../.plur.toml" keys=[bogus_key]`

* [x] **Unknown nested job key** — logged in debug
  * *Input:* `[job.rspec]` with `bogus = "value"` alongside valid `cmd`
  * *Expected:* Succeeds, debug output mentions unknown key
  * *Actual:* `unknown config keys file=".../.plur.toml" keys=[job.rspec.bogus]`

### 6.4 TOML 1.1 features

* [x] **Multiline inline tables** — supported
  * *Input:* `cmd` defined with multiline array syntax
  * *Expected:* Parses correctly
  * *Actual:* Confirmed

* [x] **Trailing commas in arrays** — supported
  * *Input:* `cmd = ["rspec", "{{target}}",]` (trailing comma)
  * *Expected:* Parses correctly
  * *Actual:* Confirmed

### 6.5 Hyphenated key resolution

* [x] **Flat hyphenated key** — resolves correctly
  * *Input:* `.plur.toml` with `dry-run = true`
  * *Expected:* Equivalent to `--dry-run` flag
  * *Actual:* Shows `[dry-run]` output

* [x] **Nested table equivalent** — also works
  * *Input:* `.plur.toml` with `[dry]` / `run = true`
  * *Expected:* Equivalent to `dry-run = true`
  * *Actual:* Shows `[dry-run]` output

---

## 7. Watch Template Tokens

### 7.1 Valid tokens

Given source `lib/**/*.rb` matching file `lib/models/user.rb`:

* [x] **`{{match}}`** → `models/user` (relative path without extension)
  * *Actual:* Target `spec/{{match}}_spec.rb` resolves to `spec/models/user_spec.rb`

* [x] **`{{path}}`** → `lib/models/user.rb` (full path)
  * *Actual:* Target `{{path}}` resolves to `lib/models/user.rb`

* [x] **`{{dir}}`** → `lib/models/` (directory with trailing slash)
  * *Actual:* Target `{{dir}}test.rb` resolves to `lib/models/test.rb`

* [x] **`{{dir_relative}}`** → `models/` (directory relative to source base)
  * *Actual:* Target `{{dir_relative}}test.rb` resolves to `models/test.rb`

* [x] **`{{base}}`** → `user.rb` (filename with extension)
  * *Actual:* Target `{{base}}` resolves to `user.rb`

* [x] **`{{name}}`** → `user` (filename without extension)
  * *Actual:* Target `spec/{{name}}_spec.rb` resolves to `spec/user_spec.rb`

* [x] **`{{ext}}`** → `.rb` (extension with dot)
  * *Actual:* Target `spec/user{{ext}}` resolves to `spec/user.rb`

* [x] **`{{ext_no_dot}}`** → `rb` (extension without dot)
  * *Actual:* Target `spec/user.{{ext_no_dot}}` resolves to `spec/user.rb`

### 7.2 Token edge cases

* [x] **File without extension** — `{{ext}}` is empty, `{{name}}` is full filename
  * *Input:* Source matching `lib/Makefile`
  * *Expected:* `{{name}}` = `Makefile`, `{{ext}}` = empty, `{{ext_no_dot}}` = empty
  * *Actual:* Target `{{name}}-{{ext}}-{{ext_no_dot}}` resolves to `Makefile--` (both empty)

* [x] **File with multiple dots** — extension is last segment only
  * *Input:* Source matching `lib/foo.bar.rb`
  * *Expected:* `{{name}}` = `foo.bar`, `{{ext}}` = `.rb`
  * *Actual:* Target `{{name}}-{{ext}}-{{ext_no_dot}}` resolves to `foo.bar-.rb-rb`

* [x] **Deeply nested file** — all path tokens correct
  * *Input:* Source `lib/**/*.rb` matching `lib/a/b/c/deep.rb`
  * *Expected:* `{{dir_relative}}` = `a/b/c/`, `{{match}}` = `a/b/c/deep`
  * *Actual:* Target `{{dir_relative}}--{{match}}` resolves to `a/b/c/--a/b/c/deep`

### 7.3 Invalid tokens

* [x] **Unknown token name** — rejected with helpful error
  * *Input:* `targets = ["spec/{{unknown}}_spec.rb"]`
  * *Expected:* Error lists available tokens
  * *Actual:* `function "unknown" not defined` + lists all 8 available tokens

* [x] **Unclosed braces** — rejected
  * *Input:* `targets = ["spec/{{match"]`
  * *Expected:* Error about invalid template
  * *Actual:* `unclosed action`

* [x] **Typo in token** — rejected with available tokens listed
  * *Input:* `targets = ["spec/{{mach}}_spec.rb"]`
  * *Expected:* Error includes "Available tokens:"
  * *Actual:* `function "mach" not defined` + `Available tokens: {{match}}, {{path}}, ...`

---

## Findings

### Unexpected behaviors

1. **Case 2.3 — Passthrough job without target_pattern exits 0 despite error.** A custom passthrough job with `cmd` but no `target_pattern` logs an ERROR but returns exit code 0. The error occurs during file discovery in `SpecCmd.Run` but isn't propagated as a nonzero exit. Should either exit nonzero or require `target_pattern` for passthrough jobs at validation time.

2. **Case 3.2 — `watch find` with no framework errors instead of showing empty watches.** When there's no detectable framework and no user watches, `buildRuntimeConfig` correctly produces empty watches, but `watch_find.go` then calls `selectJobFromRuntimeConfig` before checking for empty watches. This causes an unnecessary error. The fix would be to check `len(watches) == 0` before selecting a job.
