# CLI-UX Assessment Artifacts

This directory contains measured evidence used by the CLI-UX assessment docs.

## Versioned Binaries

Built locally from git archives:

- `plur-v0.56.0-version.txt`
- `plur-main-version.txt`
- `plur-v0.60.0-rc.1-version.txt`

Temporary build inputs and binaries live under `tmp/goal-assessment/` and are
not part of the durable assessment artifact set.

## CLI Captures

Each command capture uses three files:

- `*.stdout.txt`
- `*.stderr.txt`
- `*.exit.txt`

Captured surfaces include:

- top-level help
- watch help
- watch-find help
- text dry-run
- JSON dry-run
- text watch-find preview
- JSON watch-find preview
- helper-file watch-find no-op preview

Prefixes:

- `v056-*`: `v0.56.0`
- `main-*`: `main`
- `v060-*`: `v0.60.0-rc.1`

## Metrics

- `git-diff-*.txt` and `git-diff-*.tsv`: git diff summaries.
- `cloc-*.txt` and `cloc-*.csv`: line-count summaries from `cloc`.
- `static-*.txt`: static check outputs and exit codes.
- `hyperfine-*.md`: small benchmark outputs from `hyperfine`.

## Caveats

The small benchmark results are useful as smoke evidence only. The dry-run
benchmark is below `hyperfine`'s ideal timing range, and the one-spec benchmark
uses the small local default-ruby fixture rather than a representative large
project.
