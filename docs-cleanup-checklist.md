# Documentation Cleanup Checklist

**Generated:** 2025-10-19
**Purpose:** Clean up outdated, unnecessary, and completed documentation before open-sourcing plur

---

## Group 1: Safe to Remove - Definitely Dead ✅

**STATUS: All 15 files moved to `docs/archive/` - awaiting final deletion decision**

These documents are completed implementation plans, removed features, or obsolete research that no longer provide value. They have been archived and can be safely deleted.

### Archive Directory - Completed Work (already in archive/)

- [x] `docs/archive/2025-05-28-rspec-package.md` ✅ ARCHIVED
  * **Reason:** Completed spike/refactoring from May 2025; RSpec code moved to plur/rspec package months ago
  * **Evidence:** Work fully merged and reflected in current codebase structure

- [x] `docs/archive/2025-06-04-plur-watch-multiple-watchers-plan.md` ✅ ARCHIVED
  * **Reason:** Implementation plan for WatcherManager - fully completed
  * **Evidence:** WatcherManager exists in plur/watch/; all 5 phases implemented

- [x] `docs/archive/2025-07-11-performance-tracing.md` ✅ ARCHIVED
  * **Reason:** Documents `--trace` flag that was explicitly removed in commit 83c8616
  * **Evidence:** Feature removed 3 months ago; no trace code in current codebase

### WIP Directory - Completed Work (moved to archive/)

- [x] `docs/archive/2025-08-17-consolidate-watch-and-spec-report.md` ✅ ARCHIVED
  * **Reason:** Planning document for watch/spec consolidation - completed in commit 4a239a9
  * **Evidence:** "feat: Complete Phase 5 - Watch Consolidation" merged Aug 17, 2025
  * **Was:** `docs/wip/consolidate-watch-and-spec-report.md`

- [x] `docs/archive/2025-08-27-watcher-packaging-strategy.md` ✅ ARCHIVED
  * **Reason:** Strategy document for watcher binary packaging - implementation complete
  * **Evidence:** All watcher binaries exist; strategy moved to release docs
  * **Was:** `docs/wip/watcher-packaging-strategy.md`

### Research Directory - Decided/Implemented (moved to archive/)

- [x] `docs/archive/2025-08-22-bundler-optimization.md` ✅ ARCHIVED
  * **Reason:** Proposed bundler auto-detection strategies never implemented; project took different approach
  * **Evidence:** Current approach uses `.plur.toml` config instead
  * **Was:** `docs/research/bundler-optimization.md`

- [x] `docs/archive/2025-turbo-tests-performance-analysis.md` ✅ ARCHIVED
  * **Reason:** Outdated performance analysis; benchmarks no longer relevant
  * **Evidence:** Performance gap addressed through other optimizations
  * **Was:** `docs/research/turbo-tests-performance-analysis.md`

- [x] `docs/archive/2025-file-pattern-matching-libraries.md` ✅ ARCHIVED
  * **Reason:** Research recommending doublestar library - decision made and implemented
  * **Evidence:** doublestar adopted in go.mod
  * **Was:** `docs/research/file-pattern-matching-libraries.md`

- [x] `docs/archive/2025-race-conditions-analysis.md` ✅ ARCHIVED
  * **Reason:** Identified race condition in GetJSONRowsFormatterPath() - addressed
  * **Evidence:** Likely fixed in recent cleanup commits
  * **Was:** `docs/research/race-conditions-analysis.md`

- [x] `docs/archive/2025-spec-test-command-analysis.md` ✅ ARCHIVED
  * **Reason:** Analysis of `plur spec` vs `plur test` behavior - implemented in PR #116
  * **Evidence:** Commits 52cec3c, 8e05f58 implement mixed framework selection
  * **Was:** `docs/research/spec-test-command-analysis.md`

### Root-Level Documents - Historical (moved to archive/)

- [x] `docs/archive/2025-08-16-code-review-interactive-config.md` ✅ ARCHIVED
  * **Reason:** Code review of interactive-config branch from Aug 2025 - branch merged/completed
  * **Evidence:** Historical review, not actionable for current development
  * **Was:** `docs/code-review-interactive-config.md`

- [x] `docs/archive/2025-08-22-performance-regression-analysis.md` ✅ ARCHIVED
  * **Reason:** Analysis of interactive-config branch performance issue - historical
  * **Evidence:** Issue resolved; tied to specific point-in-time problem
  * **Was:** `docs/performance-regression-analysis.md`

- [x] `docs/archive/2025-10-13-wip-help-improvements.md` ✅ ARCHIVED
  * **Reason:** WIP document marked COMPLETED ✅ - all improvements implemented
  * **Evidence:** Document states "All improvements have been successfully implemented"
  * **Was:** `docs/wip-help-improvements.md`

### Release Documentation - Implementation Plans (moved to archive/)

- [x] `docs/archive/2025-10-goreleaser-prd.md` ✅ ARCHIVED
  * **Reason:** PRD for GoReleaser - marked "✅ IMPLEMENTED (October 2025)"
  * **Evidence:** Implementation complete; valuable as historical reference only
  * **Was:** `docs/development/releases/goreleaser-prd.md`

- [x] `docs/archive/2025-10-goreleaser-implementation-summary.md` ✅ ARCHIVED
  * **Reason:** PR summary describing changes - already merged
  * **Evidence:** Labeled "What This PR Does" - historical PR documentation
  * **Was:** `docs/development/releases/goreleaser-implementation-summary.md`

---

## Group 2: Require Review 🔍

These documents may have ongoing value but need your decision on whether to keep, update, or remove.

### Archive Directory - Potentially Valuable

- [ ] `docs/archive/2025-05-28-single-stream-full-migration.md`
  * **Status:** Contains valuable performance insights and decision rationale
  * **Issue:** Migration completed months ago; could be condensed
  * **Options:**
    1. Keep as-is for historical reference
    2. Distill key lessons into main architecture docs
    3. Remove if performance rationale is well-documented elsewhere

- [ ] `docs/archive/2025-07-11-minitest-refactoring-summary.md`
  * **Status:** Analysis with "recommendations for further improvement"
  * **Issue:** Unclear if recommendations were addressed or ignored
  * **Action needed:** Verify if recommendations were implemented before deciding

- [ ] `docs/archive/2025-07-30-minitest-flow-sequence-diagram.md`
  * **Status:** Documents CURRENT architecture with useful mermaid diagram
  * **Issue:** Should be in main docs (architecture/), not archive
  * **Options:**
    1. Move to `docs/architecture/minitest-flow.md`
    2. Keep in archive if superseded by other docs
    3. Remove if diagram is outdated

- [ ] `docs/archive/index.md`
  * **Status:** Index for archive directory
  * **Issue:** Only needed if keeping archived docs
  * **Action:** Remove if archive directory is emptied

### WIP Directory - Active Work

- [ ] `docs/wip/interactive-plur.md`
  * **Status:** Documents `plur watch find` (experimental but working)
  * **Issue:** Contains outdated TODOs; Minitest support may be complete
  * **Action needed:** Update completion status and remove stale TODOs

- [ ] `docs/wip/index.md`
  * **Status:** Index for WIP directory
  * **Issue:** References completed work as in-progress
  * **Action needed:** Update or remove based on what stays in WIP

### Research Directory - Ongoing Value

- [ ] `docs/research/verbosity-levels-plan.md`
  * **Status:** Excellent design for progressive verbosity (-v, -vv, -vvv)
  * **Issue:** Not implemented yet; valid proposal
  * **Options:**
    1. Keep in research/
    2. Move to docs/planning/
    3. Move to roadmap if not prioritized

- [ ] `docs/research/optimization-opportunities.md`
  * **Status:** Mix of completed and pending optimizations
  * **Issue:** Needs update to mark completed items
  * **Action needed:** Update status of Phase 1 items (done), review Phase 2-4

- [ ] `docs/research/performance-analysis-2025-08.md`
  * **Status:** Active optimization roadmap from Aug 2025
  * **Issue:** Phases 1-4 need status verification
  * **Action needed:** Verify current status, mark completed phases

### Root-Level Documents - Unclear Status

- [x] `docs/bundler-optimization-research.md` ✅ MOVED
  * **Status:** 239-line research document from Aug 2025
  * **Resolution:** Moved to `docs/research/bundler-optimization-research.md`
  * **Reason:** Valuable research on bundler overhead; belongs in research directory

- [x] `docs/plur-optimization-plan.md` ✅ REMOVED
  * **Status:** Performance plan from July 2025 (3 months old)
  * **Resolution:** Deleted - proposals were never pursued
  * **Reason:** Project took different optimization approach

- [ ] `docs/implementation-mkdocs-migration.md`
  * **Status:** ⚠️ **CRITICAL** - Detailed plan that was NOT executed
  * **Issue:** Looks like active docs but mkdocs.yml still in root, not migrated
  * **Options:**
    1. Delete if migration plan is abandoned
    2. Move to `docs/wip/` if still planned
    3. **Do not leave in root** - misleading

- [ ] `docs/recent-pages.md`
  * **Status:** Auto-generated index from Oct 13
  * **Issue:** Not automatically updated; duplicates git-revision-date plugin
  * **Action needed:** Decide if manual maintenance is worth it

### Release Documentation - Active Process

- [ ] `docs/development/releases/goreleaser-checklist.md`
  * **Status:** 300+ line implementation tracker with Phases 1-3 complete
  * **Issue:** Reads like planning doc; Phases 4-5 are speculative
  * **Options:**
    1. Archive as implementation log
    2. Replace with simpler status document
    3. Update to reflect only current actionable items

### Architecture/Features - Minor Issues

- [ ] TODO comment in `docs/features/watch-mode.md:141`
  * **Status:** Stale inline TODO: `# TODO you sure aout that?`
  * **Action:** Remove or clarify

- [ ] `docs/architecture/output-design.md`
  * **Status:** Reads like personal notes/future plans, not current docs
  * **Issue:** Contains "Logging needs an overhaul" and proposed `plur watch-config` command
  * **Options:**
    1. Move to roadmap
    2. Rewrite as current implementation status
    3. Move to WIP if plans are active

---

## Summary Statistics

* **Group 1 (Safe to Remove):** 15 files ✅ ALL ARCHIVED
* **Group 2 (Require Review):** 15 files → **13 remaining** (2 resolved)
* **Total Identified:** 30 items
* **Progress:** 17/30 complete (57%)

### Breakdown by Category

| Category | Safe to Remove | Require Review | Resolved | Remaining |
|----------|----------------|----------------|----------|-----------|
| Archive | 3 ✅ | 4 | 1 ✅ | 3 |
| WIP | 2 ✅ | 2 | 0 | 2 |
| Research | 5 ✅ | 3 | 0 | 3 |
| Root-level | 3 ✅ | 5 | 2 ✅ | 3 |
| Releases | 2 ✅ | 1 | 0 | 1 |
| **Total** | **15 ✅** | **15** | **3** | **12** |

---

## Recommended Actions

### Immediate (Group 1)
1. Review the 15 "Safe to Remove" files
2. Delete confirmed obsolete docs
3. Optionally move valuable historical docs to `docs/archive/history/` before deleting

### Short-term (Group 2)
1. **CRITICAL:** Decide on `implementation-mkdocs-migration.md` - delete or move to WIP
2. Update WIP docs with current status
3. Mark completed optimization phases
4. Move misplaced docs to correct directories

### Organizational
1. Create `docs/archive/history/` for truly valuable historical reference
2. Create `docs/planning/` for future work that's not in roadmap
3. Establish clear guidelines: root-level = user-facing, subdirs = research/planning/archive

---

## Notes

* Analysis based on git history, file contents, and codebase verification
* Several docs violate CLAUDE.md guideline: "Document what exists and works today, not future plans"
* Many implementation plans remain after work is complete
* Consider: Do we need an archive/ at all, or should completed work just be deleted?

---

**Next Step:** Review Group 1 items and delete confirmed obsolete documentation.
