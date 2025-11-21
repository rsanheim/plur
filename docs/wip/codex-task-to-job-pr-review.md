# Task-to-Job PR Review

_Timestamp: 2025-11-21 06:49:41Z | Author: Codex_

## High
- `plur spec` bypasses configured jobs unless `--use` is provided; it always autodetects and pulls embedded defaults, so a custom `.plur.toml` job never runs by default and its command/target overrides are ignored (`plur/main.go`). Consider preferring an explicit config job (or a declared default) before falling back to autodetection.
- Defaults only load when *both* jobs and watches are absent; if a user supplies watches but expects built-in jobs (or vice versa), `ValidateConfig` fails with undefined jobs instead of merging defaults. This blocks incremental adoption of the new structure and forces duplication of defaults to make small tweaks (`plur/watch.go` loadWatchConfiguration, `plur/watch/processor.go` validation).

## Medium
- Framework-specific behavior depends on exact job names: RSpec/Minitest parsing/formatter injection and minitest multi-file handling only trigger for `rspec`/`minitest` (exact match) while target-pattern conventions accept substrings. Aliases like `rspec-fast` or `minitest-ci` will lose parsing and custom flags even though patterns resolve (`plur/job/job.go`, `plur/runner.go`). Recommend normalizing by job type or regex instead of exact equality.
- Directory expansion without a target suffix expands to `**/*`, so pointing `plur spec` at a directory for a job with no `target_pattern` (or a pattern without `_` and `.`) will scoop the whole repo and run whatever matches instead of returning a clear error (`plur/glob.go`). Guard against empty suffixes or require an explicit pattern for non-test jobs.

## Low
- Manual “[Enter] run all tests” in watch mode picks the “first” job (rspec then minitest, else first map entry), ignoring `--use` and any notion of a default watch job (`plur/watch.go`). This is non-deterministic when multiple jobs exist and could surprise users; consider aligning with the configured/default job selection logic.
- `FindTargetsForFile` drops missing targets with info-level logs only, so watch mode skips generating missing test files silently (`plur/watch/find.go`). If creating missing tests on change is desired, consider a flag or clearer warning path.

## Nice-to-have
- Job/watch autodetect only covers Ruby/Go; if OSS users expect more frameworks, add a minimal plug-in/adapters surface so new profiles can be added without editing core code (`plur/autodetect/*`).
