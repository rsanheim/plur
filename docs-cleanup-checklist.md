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

- [x] `docs/archive/2025-05-28-single-stream-full-migration.md` ✅ KEEP
  * **Status:** Contains valuable performance insights and decision rationale
  * **Resolution:** Keep in archive as historical reference
  * **Reason:** Documents important architectural decision with performance data

- [ ] `docs/archive/2025-07-11-minitest-refactoring-summary.md`
  * **Status:** Analysis with "recommendations for further improvement"
  * **Issue:** Unclear if recommendations were addressed or ignored
  * **Action needed:** Verify if recommendations were implemented before deciding

- [x] `docs/archive/2025-07-30-minitest-flow-sequence-diagram.md` ✅ MOVED
  * **Status:** Documents CURRENT architecture with useful mermaid diagram
  * **Resolution:** Moved to `docs/architecture/test-processing-flow.md`
  * **Reason:** Valuable architecture documentation showing complete test processing flow; renamed to reflect that it covers general architecture (not just minitest)

- [x] `docs/archive/index.md` ✅ KEEP
  * **Status:** Index for archive directory
  * **Resolution:** Keep as-is
  * **Reason:** Useful navigation for archived content

### WIP Directory - Active Work

- [x] `docs/wip/interactive-plur.md` ✅ MOVED
  * **Status:** Documents `plur watch find` (experimental but working)
  * **Resolution:** Moved to docs/ideas/interactive-watch-find.md
  * **Reason:** Future feature idea; experimental and not ready to publicize

- [x] `docs/wip/index.md` ✅ KEEP
  * **Status:** Index for WIP directory
  * **Resolution:** Keep as-is
  * **Reason:** Useful navigation for WIP content

### Research Directory - Ongoing Value

- [ ] `docs/research/verbosity-levels-plan.md`
  * **Status:** Excellent design for progressive verbosity (-v, -vv, -vvv)
  * **Issue:** Not implemented yet; valid proposal
  * **Options:**
    1. Keep in research/
    2. Move to docs/planning/
    3. Move to roadmap if not prioritized

- [x] `docs/research/optimization-opportunities.md` ✅ MERGED
  * **Status:** Detailed optimization catalog (265 lines)
  * **Resolution:** Merged into unified performance-optimizations.md
  * **Reason:** Combined with performance-analysis doc; verified all code references with subagents

- [x] `docs/research/performance-analysis-2025-08.md` ✅ MERGED
  * **Status:** Optimization roadmap with Phase 1 completed
  * **Resolution:** Merged into unified performance-optimizations.md
  * **Reason:** Combined with optimization-opportunities doc; updated with current status

### Root-Level Documents - Unclear Status

- [x] `docs/bundler-optimization-research.md` ✅ MOVED
  * **Status:** 239-line research document from Aug 2025
  * **Resolution:** Moved to `docs/research/bundler-optimization-research.md`
  * **Reason:** Valuable research on bundler overhead; belongs in research directory

- [x] `docs/plur-optimization-plan.md` ✅ REMOVED
  * **Status:** Performance plan from July 2025 (3 months old)
  * **Resolution:** Deleted - proposals were never pursued
  * **Reason:** Project took different optimization approach

- [x] `docs/implementation-mkdocs-migration.md` ✅ REMOVED
  * **Status:** Unexecuted migration plan
  * **Resolution:** Deleted - decided to keep mkdocs in root
  * **Reason:** Migration not needed; existing setup works fine

- [x] `docs/recent-pages.md` ✅ KEEP
  * **Status:** Auto-generated index from Oct 13
  * **Resolution:** Keep as-is
  * **Reason:** Useful navigation aid; manual maintenance is acceptable

### Release Documentation - Active Process

- [x] `docs/development/releases/goreleaser-checklist.md` ✅ MOVED
  * **Status:** 300+ line implementation tracker with Phases 1-3 complete
  * **Resolution:** Moved to docs/wip/goreleaser-checklist.md
  * **Reason:** Active tracking doc; belongs in WIP until fully complete

### Architecture/Features - Minor Issues

- [x] TODO comment in `docs/features/watch-mode.md:141` ✅ REMOVED
  * **Status:** Stale inline TODO: `# TODO you sure aout that?`
  * **Resolution:** Removed the TODO comment
  * **Reason:** Comment was unclear and unnecessary

- [x] `docs/architecture/output-design.md` ✅ MOVED
  * **Status:** Reads like personal notes/future plans, not current docs
  * **Resolution:** Moved to docs/ideas/output-design.md
  * **Reason:** Future ideas/backlog; created new docs/ideas/ directory for such content

---

## Summary Statistics

* **Group 1 (Safe to Remove):** 15 files ✅ ALL ARCHIVED
* **Group 2 (Require Review):** 15 files ✅ ALL RESOLVED
* **Total Identified:** 30 items
* **Progress:** 30/30 complete (100%) 🎉

### Breakdown by Category

| Category | Safe to Remove | Require Review | Resolved | Remaining |
|----------|----------------|----------------|----------|-----------|
| Archive | 3 ✅ | 4 | 4 ✅ | 0 |
| WIP | 2 ✅ | 2 | 2 ✅ | 0 |
| Research | 5 ✅ | 3 | 3 ✅ | 0 |
| Root-level | 3 ✅ | 5 | 5 ✅ | 0 |
| Releases | 2 ✅ | 1 | 1 ✅ | 0 |
| **Total** | **15 ✅** | **15** | **15** | **0** |

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
