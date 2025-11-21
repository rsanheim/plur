# Task-to-Job PR Summary
_Timestamp: 2025-11-21 06:49:41Z | Author: Codex_
- Unified the runner around a single `job` model, retiring the old `task` pipeline; added templated watch mappings with richer tokens and a new defaults/autodetection layer (Ruby/Go) so `plur watch` can start without config (`plur/autodetect/*`, `plur/job/*`, `plur/watch/*`).
- CLI wiring now exposes `job`/`watch` config in `.plur.toml`, adds glob expansion helpers for job target patterns, and refreshes logging plus docs/checklists to guide the migration (`plur/main.go`, `plur/glob.go`, `plur/logger/*`, `docs/wip/*`).
- `plur watch find` and doctor fixtures updated to reflect the new mapping model and provide clearer feedback when resolving targets (`plur/watch_find.go`, `fixtures/projects/*`, `spec/integration/plur_watch/*`).

## Risks / Follow-ups
- `plur spec` ignores job definitions from `.plur.toml` unless `--use` is set, so user-configured commands/targets never run by default; autodetected defaults always win (`plur/main.go`).
- Autodetected defaults are only loaded when both jobs *and* watches are absent; supplying either one blocks defaults and produces “undefined job” errors unless users duplicate the other half (`plur/watch.go`).
- RSpec/Minitest-specific behavior is keyed to exact job names; any alias like `rspec-fast` loses the parser/formatter logic despite target conventions supporting name substrings (`plur/job/job.go`, `plur/runner.go`).
- Directory expansion for jobs without a target suffix resolves to `**/*`, which can accidentally scoop an entire repo when users pass a directory to `plur spec` for non-test jobs (`plur/glob.go`).
