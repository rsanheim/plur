# Runner jobs implementation plan

This plan implements the behavior specified in:
- docs/architecture/runner-jobs-rfc.md

Each phase is scoped to fit a single coding session. Use this as the checklist
for incremental changes and verification.

## Phase 1: Resolver + validation refactor (core behavior)

**Tasks**
- [x] Implement a single, linear "resolved jobs" function that merges built-ins
      and user jobs per the RFC (no early abstractions).
- [x] Apply framework defaulting and target pattern defaulting as specified.
- [x] Add config validation at load time (resolved jobs must have `Cmd`, known
      frameworks; watch mappings reference known jobs).
- [x] Ensure autodetect uses resolved jobs only and follows the RFC priority
      (rspec → minitest → go-test).
- [x] Remove any directory fallback logic in resolver.
- [x] Update debug/verbose logs to indicate inherited fields (cmd/framework/
      target_pattern) when resolved from built-ins.

**Success criteria**
- [x] `plur` in this repo selects rspec (no explicit config changes).
- [x] `plur -n4 --dry-run --debug` shows resolved rspec job, with inherited
      target_pattern when user overrides only `cmd`.
- [x] `bin/rake test:go` passes.

## Phase 2: Explicit pattern inference + expansion rules

**Tasks**
- [x] Implement framework inference for explicit patterns using `DetectPatterns`
      (directory/file/glob rules as in RFC).
- [x] If multiple frameworks match, return a clear error instructing the user
      to split the command or pass `--use`.
- [x] Implement simplified pattern expansion rules after selection:
      directories append `TargetPattern` or framework `DetectPatterns`.
- [x] Ensure explicit pattern expansion errors when it yields zero files.

**Success criteria**
- [x] `plur spec/models` selects rspec and expands under `spec/models`.
- [x] `plur test/models` selects minitest and expands under `test/models`.
- [x] Mixed explicit patterns (spec + test) raise a clear error.
- [x] `bin/rake test:go` passes.

## Phase 3: Integration specs + docs cleanup

**Tasks**
- [x] Add integration specs for resolver/autodetect flows described in the RFC.
- [x] Update or remove any outdated docs; ensure references point to the RFC.
- [x] Verify `bin/rake` passes (Ruby + Go).
- [x] Manual spot checks:
      - `plur` picks rspec in this repo.
      - `plur --use fast` respects framework defaults when TargetPattern unset.

**Success criteria**
- [x] All new integration specs pass.
- [x] `bin/rake` passes in a clean environment.
- [x] RFC and implementation plan are consistent and up to date.

Notes:
- `bin/rake` currently passes with existing pending specs (DB setup + flaky watch).

## Phase 4: Simplification + dead code cleanup

**Tasks**
- [x] Audit resolver/runner code for now-unnecessary helpers and abstractions.
- [x] Remove dead code paths, legacy conventions, and duplicated logic.
- [x] Simplify glob/pattern helpers that are no longer used after refactor.
- [x] Ensure debug logging remains clear after cleanup.
- [x] Remove all backward-compatibility shims, aliases, or deprecated code paths.

**Success criteria**
- [x] `bin/rake` passes after cleanup.
- [x] No unused helpers or unreachable branches in the resolver path.
- [x] Docs remain accurate post-cleanup.
- [x] No backward-compatibility code paths remain.
- [x] Resolver flow is deterministic and shows resolved job inheritance in
      debug/verbose output.

## Phase 5: Follow-up refactors enabled by framework/job work

This phase is for investigating opportunities made possible by the new
framework/job resolver and pattern logic. It should result in concrete
proposals and scoped follow-up work.

**Tasks**
- [ ] Identify additional single sources of truth to consolidate (e.g., unify
      framework detect patterns and defaults.toml where feasible).
- [ ] Assess config schema cleanup now that job merging is explicit (remove any
      remaining unused/legacy fields from templates and docs).
- [ ] Review watch/run parity: consider sharing more path expansion logic
      between run mode and watch mode where safe.
- [ ] Consider centralizing more glob helpers under `internal/patterns` to
      reduce duplication across autodetect and run expansion.
- [ ] Evaluate user-facing error messages when no tests exist but watch mappings
      do (clearer guidance for new repos).

**Success criteria**
- [ ] A short RFC or issue list capturing the above opportunities with scope,
      constraints, and proposed tests.
- [ ] Any chosen follow-up items are prioritized with explicit owners.
