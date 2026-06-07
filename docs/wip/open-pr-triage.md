# Open PR Triage

This document is a rolling triage note for open pull requests in `rsanheim/plur`.

Keep it focused on current recommendations, not one-time review snapshots. Update it when PRs open, close, merge, or materially change direction.

## Triage Rules

Use these categories when deciding what to do with a PR:

- `refresh`: the idea is still valuable, but the branch is too stale to merge directly
- `split/archive`: the branch mixes too many concerns or contains spike work that should not merge as one PR
- `close`: the branch is superseded, obsolete, or no longer the cleanest expression of the change
- `merge-ready`: the branch is current, focused, and can be reviewed directly

## Current Recommendations

| PR | Action | Why |
| --- | --- | --- |
| `#31` Merge user watch mappings with built-in defaults | `refresh` as a new PR on current `main` | This is the smallest high-value unresolved watch bug. The underlying problem is still live on `main`, but the old branch targets pre-refactor files and uses append-only merging. |
| `#29` Watch mode: fix config merging, terminal output, and color | `split/archive` | This branch mixes multiple concerns: config merge, terminal abstraction, prompt behavior, color/TTY fixes, and docs. Treat it as a spike branch and salvage follow-on PRs only if the individual fixes are still wanted. |
| `#17` Exclude patterns | `refresh` after the watch merge work | Old, but still coherent. The feature is isolated and the touched files still exist on `main`, so it remains a good candidate after the higher-priority watch fix. |

## Recently Closed

| PR | Outcome | Why |
| --- | --- | --- |
| `#19` Fix watch mode config merge, duplicate warnings, and prompt rendering | `closed as superseded` | Overlapped by `#31` for the config-merge bug and by `#29` for the broader watch-output work. It no longer represented the cleanest unit of work. |

## Current Code Findings

### Watch merge behavior still needs work

Current `main` still treats user-defined watch mappings as a replacement for built-in watch mappings:

```go
if len(cli.WatchMappings) > 0 {
    rc.Watches = cli.WatchMappings
} else {
    // fallback to built-in watches
}
```

This behavior lives in `internal/runtime/config.go` and means adding any user `[[watch]]` entries drops the built-in mappings entirely.

### The desired follow-on behavior is already documented

`docs/plans/2026-04-01-watch-config-merge-override.md` describes the better final behavior:

- user watches should be additive by default
- named user watches should be able to override built-ins by name
- downstream watch processing should continue to consume a plain `[]WatchMapping`

That design is a better replacement target than simply rebasing the old append-only fix from `#31`.

### The old watch branches target outdated file layout

The watch branches modify files from an older runtime and integration-spec
layout. Current `main` has moved this logic into:

- `internal/runtime/config.go`
- `internal/runtime/defaults.go`
- `spec/integration/watch/...`

That is the main reason the watch PRs are stale even when their original CI runs were green.

## Priority Queue

1. Replace `#31` with a fresh PR on top of current `main`.
2. Decide whether any pieces of `#29` should be split into focused follow-on PRs.
3. Refresh `#17` after the watch merge work lands.

## Replacement Scope For `#31`

The replacement for `#31` should:

- implement watch merging in `internal/runtime/config.go`
- add unit coverage in `internal/runtime/config_test.go`
- add merge behavior tests near `watch/watch_mapping.go` and `watch/watch_mapping_test.go`
- add or refresh an integration test under `spec/integration/watch/`
- prefer the override-by-name design from `docs/plans/2026-04-01-watch-config-merge-override.md`

## Maintenance Notes

- Keep this document evergreen. Avoid review-date language, one-time counts, and branch-ahead/behind numbers.
- Update the `Current Recommendations` table whenever PR state changes.
- Move closed or merged items into `Recently Closed` only if the decision is still useful context.
