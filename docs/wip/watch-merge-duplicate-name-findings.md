# Watch Merge Duplicate-Name Findings

## Question

Did we already have notes in the plan, git history, or other artifacts about how watch merge should behave when multiple user-defined `[[watch]]` entries share the same `name`?

## Short Answer

Yes and no:

- Yes: the existing plan doc explicitly treated `name` as the merge identity for named watches.
- No: I did not find any artifact that also said duplicate user watch names are invalid, must be rejected, or should remain distinct.
- Public config docs and examples show `name` as an optional field, but they do not document it as a unique key.

That means the current implementation follows the plan doc, but the reviewer is still right: the plan itself assumed uniqueness without the config contract ever clearly establishing it.

## What The Plan Doc Says

The clearest artifact is the existing planning doc:

- [2026-04-01-watch-config-merge-override.md](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md)

Relevant lines:

- [docs/plans/2026-04-01-watch-config-merge-override.md:7](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:7)
  - "Named user watches with a matching key replace built-ins"
- [docs/plans/2026-04-01-watch-config-merge-override.md:15](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:15)
  - `mergeKey()` returns `"name:" + w.Name` for named watches
- [docs/plans/2026-04-01-watch-config-merge-override.md:23](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:23)
  - "Named watches: keyed by name"
- [docs/plans/2026-04-01-watch-config-merge-override.md:42](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:42)
  - user watches overwrite by key
- [docs/plans/2026-04-01-watch-config-merge-override.md:77](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:77)
  - "Add new mapping | `name = \"custom\"` with new source | Appended after builtins"
- [docs/plans/2026-04-01-watch-config-merge-override.md:113](/Users/rsanheim/src/rsanheim/plur/docs/plans/2026-04-01-watch-config-merge-override.md:113)
  - "same name, different fields | same key (name wins)"

Interpretation:

- The plan absolutely assumed `name` was the identity key for named watches.
- The plan did not distinguish builtin-vs-user override from user-vs-user collision.
- The plan also did not add a validation rule that duplicate user names should be rejected.

So the reviewer found a real gap in the plan, not just in the implementation.

## What Public Docs And Examples Say

The public config docs and examples show `name` in watch examples, but they do not define it as unique.

Relevant references:

- [docs/configuration.md:210](/Users/rsanheim/src/rsanheim/plur/docs/configuration.md:210)
  - the watch field table lists `source`, `targets`, `jobs`, `ignore`, and `reload`
  - it does not define `name` semantics in that table
- [docs/configuration.md:227](/Users/rsanheim/src/rsanheim/plur/docs/configuration.md:227)
  - examples use `name = "lib-to-spec"` and `name = "spec-files"`
- [docs/examples/plur.toml.example:66](/Users/rsanheim/src/rsanheim/plur/docs/examples/plur.toml.example:66)
  - example watch mappings include `name`

Interpretation:

- Users can reasonably infer that `name` is descriptive or optional.
- There is no visible contract saying `name` must be unique across user-defined watches.
- There is no documented warning that duplicate names collapse or override.

That supports the review comment: collapsing same-name user mappings is surprising behavior under the current docs.

## What PR And Git History Say

### PR `#31`

The older PR `#31` did not use name-based merging at all. Its body only described additive merge via `slices.Concat`:

- "User-defined `[[watch]]` entries ... now augment the built-in watch mappings"
- "ensuring user watches and defaults coexist"

That means `#31` gives no evidence that duplicate user watch names were ever meant to collide.

### Later follow-on artifacts

The later runtime-boundary note points back to the name-based override plan:

- [2026-04-02-runtime-config-boundary-design.md:281](/Users/rsanheim/src/rsanheim/plur/docs/superpowers/specs/2026-04-02-runtime-config-boundary-design.md:281)
  - "Watch config name-based override (`MergeWatches`) — see `docs/plans/2026-04-01-watch-config-merge-override.md`"

This confirms the plan was reused as the follow-on design reference.

### Search result summary

I did not find any repo artifact that explicitly says one of the following:

- duplicate named user watches are invalid
- duplicate named user watches must be rejected during validation
- duplicate named user watches should remain separate and only builtin-vs-user should override

I also did not find a prior test or doc that exercised "two user watches with the same name but different sources."

## Overall Conclusion

The state of the artifacts is:

1. The plan doc explicitly encoded `name` as identity for named watch merges.
2. The plan doc did not account for user-vs-user duplicate-name collisions.
3. Public docs/examples did not establish `name` as a unique key.
4. PR `#31` history does not cover this issue because it was additive-only.

So the review comment is valid. The implementation matches the existing plan, but the plan itself appears incomplete relative to the public config contract.

## Practical Implication

Before merging this work, we should choose one of these paths explicitly:

1. Make `name` a true unique key.
   - Reject duplicate named user watches in validation.
   - Update config docs to say `name` must be unique.

2. Preserve distinct user watches with the same `name`.
   - Change merge behavior so builtin-vs-user override does not collapse unrelated user-vs-user mappings.
   - Keep `name` as descriptive rather than unique.

Right now we are in an inconsistent middle state:

- the plan and code treat `name` as unique
- the user-facing docs do not

That mismatch is the real issue.
