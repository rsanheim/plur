# Move Framework Adapters Under `framework/` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `minitest/`, `rspec/`, and `passthrough/` to `framework/minitest/`, `framework/rspec/`, and `framework/passthrough/` so adapter packages live under their parent dispatcher.

**Architecture:** Move the three adapter directories under `framework/`, then update every repo-local path reference found by a repo-wide scan. Go package names (`minitest`, `rspec`, `passthrough`) stay identical; only repo-local import/path references change. No behavior changes. Validated by unchanged Go tests, the formatter Ruby spec, and full `bin/rake`.

**Tech Stack:** Go modules, git, bin/rake (Ruby-driven build+test orchestration)

**Overall Intention:** Use this reorg to move the repo toward clearer package ownership and a cleaner top level. Related packages should live with the code that owns them, and follow-up refactors should keep reducing root-level leaf packages and random one-off directories/files.

---

## Facts Established During Planning

**Go importers of these three packages (all under plur root, no external consumers):**

| Package | Importers |
|---------|-----------|
| `plur/minitest` | `framework/framework.go` only |
| `plur/rspec` | `framework/framework.go`, `complexity_test.go`, `benchmark_test.go` |
| `plur/passthrough` | `framework/framework.go` only |

**Repo-wide scan findings (excluding `tmp/`, `vendor/`, and `docs/plans/`):**

* Build-affecting non-Go repo-local path reference: `spec/integration/spec/json_rows_formatter_spec.rb:3` has `require_relative "../../../rspec/formatter"`.
* Comment/doc references describing the old layout: `rspec/parser.go:90`, `docs/architecture/test-processing-flow.md:118`, `docs/architecture/go-concurrency-and-data-structures-review.md:88`.
* Ignore matches that only name Ruby gems/frameworks rather than repo paths, e.g. `require "rspec/core/rake_task"` or `require 'minitest/autorun'`.

**go:embed check:** `rspec/formatter.go:12` has `//go:embed formatter.rb`. Embed uses relative paths; moves cleanly with the directory.

**Package declarations:** All files in each adapter use `package minitest` / `package rspec` / `package passthrough`. These don't change — only the import path does.

---

### Task 1: Baseline — confirm green state before touching anything

**Files:** None (verification only)

- [x] **Step 1: Confirm working tree clean and note current branch**

Run: `git status && git rev-parse --abbrev-ref HEAD`
Expected: clean tree. If already on `refactor/framework-adapters-subpackages`, continue in place. If on `main`, create the refactor branch in Task 2. If on some other branch, stop and decide whether to continue there or create the planned refactor branch.

- [x] **Step 2: Run full build + test baseline**

Run: `bin/rake`
Expected: PASS. If this fails, STOP — resolve pre-existing failures before refactoring.

- [x] **Step 3: Record baseline output for later comparison**

Run: `bin/rake test:go 2>&1 | tail -20`
Save the PASS/FAIL summary mentally (or scroll-back). You'll compare post-move.

- [x] **Step 4: Run a repo-wide scan for repo-local adapter path references**

Run:

```bash
rg -n -F \
  -e 'github.com/rsanheim/plur/minitest' \
  -e 'github.com/rsanheim/plur/rspec' \
  -e 'github.com/rsanheim/plur/passthrough' \
  -e 'require_relative "../../../rspec/formatter"' \
  -e '// See rspec/formatter.rb for the formatter implementation.' \
  -e '### 3. **Parser** (rspec/ or minitest/)' \
  . --glob '!docs/plans/**' --glob '!tmp/**' --glob '!vendor/**'
```

Expected matches:
* `framework/framework.go`
* `complexity_test.go`
* `benchmark_test.go`
* `spec/integration/spec/json_rows_formatter_spec.rb`
* `rspec/parser.go`
* `docs/architecture/test-processing-flow.md`
* `docs/architecture/go-concurrency-and-data-structures-review.md`

Anything else needs to be reviewed before starting the move. The goal is to catch repo-local path references, not every mention of the Ruby frameworks by name.

---

### Task 2: Create refactor branch

**Files:** None (git only)

- [x] **Step 1: Create and switch to branch if needed**

Run: `git checkout -b refactor/framework-adapters-subpackages`
Expected: `Switched to a new branch 'refactor/framework-adapters-subpackages'` if starting from `main`.
If Task 1 already confirmed you are on `refactor/framework-adapters-subpackages`, skip this step.

- [x] **Step 2: Confirm branch**

Run: `git rev-parse --abbrev-ref HEAD`
Expected: `refactor/framework-adapters-subpackages`

---

### Task 3: Move the three adapter directories

**Files:**
- Move: `minitest/` → `framework/minitest/`
- Move: `rspec/` → `framework/rspec/`
- Move: `passthrough/` → `framework/passthrough/`

- [x] **Step 1: Move minitest**

Run: `git mv minitest framework/minitest`
Expected: silent success. Verify with `ls framework/minitest` — should contain `failures.go`, `failures_test.go`, `output_parser.go`, `output_parser_test.go`.

- [x] **Step 2: Move rspec**

Run: `git mv rspec framework/rspec`
Expected: silent success. Verify with `ls framework/rspec` — should contain `failure_list.go`, `formatter.go`, `formatter.rb`, `json_output.go`, `parser.go`, `parser_test.go`.

- [x] **Step 3: Move passthrough**

Run: `git mv passthrough framework/passthrough`
Expected: silent success. Verify with `ls framework/passthrough` — should contain `parser.go`.

- [x] **Step 4: Verify git sees three renames (not delete+add)**

Run: `git status`
Expected: renames shown as `renamed: minitest/... -> framework/minitest/...` etc. If any show as `deleted` + `new file`, stop and investigate — that means git lost rename detection.

- [x] **Step 5: Confirm build breaks (baseline for import fix)**

Run: `go build ./...`
Expected: FAIL with errors like `framework/framework.go:9:2: package github.com/rsanheim/plur/minitest is not in std`. This confirms the moves happened and imports need updating.

---

### Task 4: Update import paths in `framework/framework.go`

**Files:**
- Modify: `framework/framework.go:9-11`

- [x] **Step 1: Update the three imports**

Edit `framework/framework.go`, replace the three import lines:

```go
	"github.com/rsanheim/plur/minitest"
	"github.com/rsanheim/plur/passthrough"
	"github.com/rsanheim/plur/rspec"
```

with:

```go
	"github.com/rsanheim/plur/framework/minitest"
	"github.com/rsanheim/plur/framework/passthrough"
	"github.com/rsanheim/plur/framework/rspec"
```

Package-level references (`minitest.X`, `rspec.Y`, `passthrough.Z`) do NOT change — only the import path.

- [x] **Step 2: Verify framework package compiles**

Run: `go build ./framework/...`
Expected: silent success.

---

### Task 5: Update import paths and repo-local path references

**Files:**
- Modify: `complexity_test.go:7`
- Modify: `benchmark_test.go:9`
- Modify: `spec/integration/spec/json_rows_formatter_spec.rb:3`
- Modify: `framework/rspec/parser.go:90`
- Modify: `docs/architecture/test-processing-flow.md:118`
- Modify: `docs/architecture/go-concurrency-and-data-structures-review.md:88`

- [x] **Step 1: Update `complexity_test.go`**

Edit `complexity_test.go`, replace:

```go
	"github.com/rsanheim/plur/rspec"
```

with:

```go
	"github.com/rsanheim/plur/framework/rspec"
```

- [x] **Step 2: Update `benchmark_test.go`**

Edit `benchmark_test.go`, replace:

```go
	"github.com/rsanheim/plur/rspec"
```

with:

```go
	"github.com/rsanheim/plur/framework/rspec"
```

- [x] **Step 3: Update `json_rows_formatter_spec.rb`**

Edit `spec/integration/spec/json_rows_formatter_spec.rb`, replace:

```ruby
require_relative "../../../rspec/formatter"
```

with:

```ruby
require_relative "../../../framework/rspec/formatter"
```

- [x] **Step 4: Update the moved parser comment**

Edit `framework/rspec/parser.go`, replace:

```go
// See rspec/formatter.rb for the formatter implementation.
```

with:

```go
// See framework/rspec/formatter.rb for the formatter implementation.
```

- [x] **Step 5: Update docs that describe the old layout**

Edit `docs/architecture/test-processing-flow.md`, replace:

```md
### 3. **Parser** (rspec/ or minitest/)
```

with:

```md
### 3. **Parser** (framework/rspec/ or framework/minitest/)
```

Edit `docs/architecture/go-concurrency-and-data-structures-review.md` to describe the current RSpec file-tracking behavior:

```md
* RSpec now tracks `CurrentFile` directly inside `framework/rspec/parser.go` for trace output instead of emitting a separate group-started notification.
```

- [x] **Step 6: Confirm nothing else references the old repo-local paths**

Run:

```bash
rg -n -F \
  -e 'github.com/rsanheim/plur/minitest' \
  -e 'github.com/rsanheim/plur/rspec' \
  -e 'github.com/rsanheim/plur/passthrough' \
  -e 'require_relative "../../../rspec/formatter"' \
  -e '// See rspec/formatter.rb for the formatter implementation.' \
  -e '### 3. **Parser** (rspec/ or minitest/)' \
  . --glob '!docs/plans/**' --glob '!tmp/**' --glob '!vendor/**'
```

Expected: NO output. If anything matches, update that file too with the new `framework/...` or `framework/rspec/...` path.

---

### Task 6: Verify everything builds

**Files:** None (verification)

- [x] **Step 1: Full module compile**

Run: `go build ./...`
Expected: silent success, exit 0.

- [x] **Step 2: Full module vet**

Run: `go vet ./...`
Expected: silent success, exit 0.

---

### Task 7: Run Go test suite

**Files:** None (verification)

- [x] **Step 1: Run all Go tests**

Run: `bin/rake test:go`
Expected: PASS. Same test count and same pass rate as the baseline from Task 1. The moved tests (in `framework/minitest/`, `framework/rspec/`) execute under their new paths but produce identical results.

- [x] **Step 2: If any test fails, diagnose before proceeding**

Common causes if this step fails:
* Missed import update → re-run the grep from Task 5 Step 3.
* `go:embed` resolution broken → verify `framework/rspec/formatter.rb` exists beside `formatter.go` (git mv preserves this).
* Stale test binary cache → `go clean -testcache && bin/rake test:go`.

---

### Task 8: Run full build + Ruby integration suite

**Files:** None (verification)

- [x] **Step 1: Run the formatter Ruby spec directly**

Run: `bin/rspec spec/integration/spec/json_rows_formatter_spec.rb`
Expected: PASS. This directly verifies the moved `framework/rspec/formatter.rb` path still loads from Ruby.

- [x] **Step 2: Full `bin/rake`**

Run: `bin/rake`
Expected: PASS. This runs build → lint → install → tests (Go + Ruby integration). Ruby integration tests invoke the installed `plur` binary against fixtures under `fixtures/`; after the repo-local path updates above, they should pass unchanged.

- [x] **Step 3: Sanity-check installed binary works**

Run: `plur --version && plur -C fixtures/projects/minitest-success --dry-run`
Expected: version prints, dry-run shows detected tests. Confirms the reinstalled binary functions end-to-end.

---

### Task 9: Commit

**Files:** All staged changes from Tasks 3–5.

- [x] **Step 1: Review the diff**

Run: `git status && git diff --stat`
Expected files changed:
* `framework/framework.go` (3 import lines changed)
* `complexity_test.go` (1 import line changed)
* `benchmark_test.go` (1 import line changed)
* `spec/integration/spec/json_rows_formatter_spec.rb` (1 `require_relative` path changed)
* `framework/rspec/parser.go` (1 comment path changed)
* `docs/architecture/test-processing-flow.md` (1 heading path changed)
* `docs/architecture/go-concurrency-and-data-structures-review.md` (1 path reference changed)
* Renames: `minitest/*` → `framework/minitest/*`, `rspec/*` → `framework/rspec/*`, `passthrough/*` → `framework/passthrough/*`

- [x] **Step 2: Stage and commit**

Run:

```bash
git add -A && git commit -m "$(cat <<'EOF'
refactor: move minitest/rspec/passthrough under framework/

Adapter packages now live as subpackages of their dispatcher. Express
the architecture directly in the directory layout — framework/ owns its
adapters rather than having three sibling packages whose relationship
is only visible by reading imports.

No behavior change. Package names unchanged; repo-local paths shift
to github.com/rsanheim/plur/framework/{minitest,rspec,passthrough}
and framework/rspec/* where needed.
EOF
)"

Expected: commit created, `bin/rake` hook (if any) passes.

- [x] **Step 3: Confirm clean tree**

Run: `git status`
Expected: `nothing to commit, working tree clean`.

---

---

## Phase 2: Consistent adapter file names

This phase runs *after* Phase 1 (Tasks 1–9) is committed. It can stay on the same branch as a follow-up commit, or be a separate PR — both are fine.

**Inconsistencies identified — two file-naming issues in minitest:**

1. **Main streaming parser is named differently across adapters:**

   | Adapter | Current main parser file | Exports |
   |---------|--------------------------|---------|
   | `framework/minitest/` | `output_parser.go` | `NewOutputParser() types.TestOutputParser` |
   | `framework/rspec/` | `parser.go` | `NewOutputParser() types.TestOutputParser` |
   | `framework/passthrough/` | `parser.go` | `NewOutputParser() types.TestOutputParser` |

   Two of three already use `parser.go`; minitest is the outlier.

2. **Minitest's secondary parser is named for what it extracts, not what it is.** `failures.go` parses minitest's end-of-run summary block (lines like `1) Failure:` / `2) Error:`) into structured notifications. Its filename sounds like a domain type; its actual role is a post-run summary parser. Rename to `summary_parser.go` to match the `_parser.go` convention and make its responsibility explicit.

   After the rename, minitest has two parsers side-by-side with clear roles:

   | File | Role |
   |------|------|
   | `parser.go` | streaming, line-at-a-time; implements `types.TestOutputParser` |
   | `summary_parser.go` | bulk, post-run; exports `ExtractFailures()` |

   rspec and passthrough don't need a second parser — rspec's JSON stream carries structured failures inline, and passthrough doesn't parse at all.

**Scope of Phase 2:** four file renames in `framework/minitest/`. Other per-adapter file differences reflect real responsibilities and stay as-is:

* `framework/rspec/json_output.go` — streaming JSON row message structs (`StreamingMessage`, `StreamExample`, `StreamException`, `ExampleGroup`, `LoadSummary`).
* `framework/rspec/failure_list.go` — helper for building the `Failed examples:` rerun list from parsed failures.
* `framework/rspec/formatter.go` — wraps the embedded `formatter.rb` Ruby script via `//go:embed`; a setup-time concern, not a parsing concern.

### Task 10: Rename minitest parser files

**Files:**
- Rename: `framework/minitest/output_parser.go` → `framework/minitest/parser.go`
- Rename: `framework/minitest/output_parser_test.go` → `framework/minitest/parser_test.go`
- Rename: `framework/minitest/failures.go` → `framework/minitest/summary_parser.go`
- Rename: `framework/minitest/failures_test.go` → `framework/minitest/summary_parser_test.go`

- [x] **Step 1: Rename the streaming parser files**

Run:

```bash
git mv framework/minitest/output_parser.go framework/minitest/parser.go
git mv framework/minitest/output_parser_test.go framework/minitest/parser_test.go
```

Expected: silent success for both.

- [x] **Step 2: Rename the summary parser files**

Run:

```bash
git mv framework/minitest/failures.go framework/minitest/summary_parser.go
git mv framework/minitest/failures_test.go framework/minitest/summary_parser_test.go
```

Expected: silent success for both.

- [x] **Step 3: Confirm git sees all four as renames (not delete+add)**

Run: `git status`
Expected: four `renamed:` lines. If any show as `deleted:` + `new file:`, stop and investigate — that means git lost rename detection.

- [x] **Step 4: Confirm no code-level references to the old filenames**

Run:

```bash
rg --files framework/minitest | rg '(output_parser(_test)?|failures(_test)?)\.go$'
```

Expected before the renames: four matches (`output_parser.go`, `output_parser_test.go`, `failures.go`, `failures_test.go`).
Expected after the renames: NO output. Go symbol names inside the files (`outputParser` type, `ExtractFailures` function) are unchanged — only filenames move.

### Task 11: Verify build and tests

**Files:** None (verification)

- [x] **Step 1: Full module compile**

Run: `go build ./...`
Expected: silent success. Package name and all symbols unchanged; only the file name changed.

- [x] **Step 2: Run all Go tests**

Run: `bin/rake test:go`
Expected: PASS. Same tests run; identical results.

- [x] **Step 3: Full `bin/rake`**

Run: `bin/rake`
Expected: PASS.

### Task 12: Commit

**Files:** Four renames staged.

- [x] **Step 1: Review diff**

Run: `git status && git diff --stat`
Expected: four renames, zero content changes.

- [x] **Step 2: Commit**

Run:

```bash
git add -A && git commit -m "$(cat <<'EOF'
refactor: give minitest parser files role-expressive names

Two renames in framework/minitest/:

  output_parser.go      -> parser.go           (matches rspec/passthrough;
                                                main streaming parser)
  failures.go           -> summary_parser.go   (post-run summary-block
                                                parser; paired with parser.go)

minitest has two parsers because minitest emits failures in an end-of-run
summary block rather than inline. Keeping them as separate files reflects
that real distinction; renaming surfaces the parser/parser relationship
rather than hiding summary_parser behind a domain-type-sounding name.

Pure git mv; no code changes.
EOF
)"
```

Expected: commit created.

- [x] **Step 3: Confirm clean tree**

Run: `git status`
Expected: `nothing to commit, working tree clean`.

---

## Rollback Plan

If anything fails after Task 3 and you want to start over:

```bash
git reset --hard main
git checkout main
git branch -D refactor/framework-adapters-subpackages
```

No remote was touched; no destructive effect outside the branch.

## Post-Merge Follow-ups (out of scope for this plan)

These are mentioned for context but are **not** part of this plan:
* Apply the same direction elsewhere: move packages next to their owner when that reduces root-level clutter and makes responsibilities more obvious.
* Tier 2 internal-ization: move `types/`, `job/` under `internal/`.
* Tier 3 internal-ization: move `framework/`, `watch/`, `logger/`, `config/` under `internal/` (would cascade-move the adapter subpackages along for free — that's the design benefit of starting with this plan).
