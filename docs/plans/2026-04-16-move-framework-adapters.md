# Move Framework Adapters Under `framework/` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `minitest/`, `rspec/`, and `passthrough/` to `framework/minitest/`, `framework/rspec/`, and `framework/passthrough/` so adapter packages live under their parent dispatcher.

**Architecture:** Pure directory move. Go package names (`minitest`, `rspec`, `passthrough`) stay identical — only import paths change. No behavior changes, no file edits beyond import statements. Validated by unchanged Go tests and full `bin/rake`.

**Tech Stack:** Go modules, git, bin/rake (Ruby-driven build+test orchestration)

---

## Facts Established During Planning

**Go importers of these three packages (all under plur root, no external consumers):**

| Package | Importers |
|---------|-----------|
| `plur/minitest` | `framework/framework.go` only |
| `plur/rspec` | `framework/framework.go`, `complexity_test.go`, `benchmark_test.go` |
| `plur/passthrough` | `framework/framework.go` only |

**Non-Go references:** None in `Rakefile`, `lib/`, `docs/`, `.circleci/config.yml`, `.github/workflows/`. Confirmed via grep.

**go:embed check:** `rspec/formatter.go:12` has `//go:embed formatter.rb`. Embed uses relative paths; moves cleanly with the directory.

**Package declarations:** All files in each adapter use `package minitest` / `package rspec` / `package passthrough`. These don't change — only the import path does.

---

### Task 1: Baseline — confirm green state before touching anything

**Files:** None (verification only)

- [ ] **Step 1: Confirm working tree clean on `main`**

Run: `git status && git rev-parse --abbrev-ref HEAD`
Expected: clean tree, on `main` (or note current branch — plan assumes starting from clean main).

- [ ] **Step 2: Run full build + test baseline**

Run: `bin/rake`
Expected: PASS. If this fails, STOP — resolve pre-existing failures before refactoring.

- [ ] **Step 3: Record baseline output for later comparison**

Run: `bin/rake test:go 2>&1 | tail -20`
Save the PASS/FAIL summary mentally (or scroll-back). You'll compare post-move.

---

### Task 2: Create refactor branch

**Files:** None (git only)

- [ ] **Step 1: Create and switch to branch**

Run: `git checkout -b refactor/framework-adapters-subpackages`
Expected: `Switched to a new branch 'refactor/framework-adapters-subpackages'`

- [ ] **Step 2: Confirm branch**

Run: `git rev-parse --abbrev-ref HEAD`
Expected: `refactor/framework-adapters-subpackages`

---

### Task 3: Move the three adapter directories

**Files:**
- Move: `minitest/` → `framework/minitest/`
- Move: `rspec/` → `framework/rspec/`
- Move: `passthrough/` → `framework/passthrough/`

- [ ] **Step 1: Move minitest**

Run: `git mv minitest framework/minitest`
Expected: silent success. Verify with `ls framework/minitest` — should contain `failures.go`, `failures_test.go`, `output_parser.go`, `output_parser_test.go`.

- [ ] **Step 2: Move rspec**

Run: `git mv rspec framework/rspec`
Expected: silent success. Verify with `ls framework/rspec` — should contain `formatter.go`, `formatter.rb`, `json_output.go`, `json_output_test.go`, `parser.go`, `parser_test.go`.

- [ ] **Step 3: Move passthrough**

Run: `git mv passthrough framework/passthrough`
Expected: silent success. Verify with `ls framework/passthrough` — should contain `parser.go`.

- [ ] **Step 4: Verify git sees three renames (not delete+add)**

Run: `git status`
Expected: renames shown as `renamed: minitest/... -> framework/minitest/...` etc. If any show as `deleted` + `new file`, stop and investigate — that means git lost rename detection.

- [ ] **Step 5: Confirm build breaks (baseline for import fix)**

Run: `go build ./...`
Expected: FAIL with errors like `framework/framework.go:9:2: package github.com/rsanheim/plur/minitest is not in std`. This confirms the moves happened and imports need updating.

---

### Task 4: Update import paths in `framework/framework.go`

**Files:**
- Modify: `framework/framework.go:9-11`

- [ ] **Step 1: Update the three imports**

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

- [ ] **Step 2: Verify framework package compiles**

Run: `go build ./framework/...`
Expected: silent success.

---

### Task 5: Update import paths in root-level test files

**Files:**
- Modify: `complexity_test.go:7`
- Modify: `benchmark_test.go:9`

- [ ] **Step 1: Update `complexity_test.go`**

Edit `complexity_test.go`, replace:

```go
	"github.com/rsanheim/plur/rspec"
```

with:

```go
	"github.com/rsanheim/plur/framework/rspec"
```

- [ ] **Step 2: Update `benchmark_test.go`**

Edit `benchmark_test.go`, replace:

```go
	"github.com/rsanheim/plur/rspec"
```

with:

```go
	"github.com/rsanheim/plur/framework/rspec"
```

- [ ] **Step 3: Confirm nothing else references the old paths**

Run: `grep -rn 'plur/minitest"\|plur/rspec"\|plur/passthrough"' --include='*.go' . | grep -v '/tmp/' | grep -v '/vendor/'`
Expected: NO output. If anything matches, update that file too with the new `framework/...` path.

---

### Task 6: Verify everything builds

**Files:** None (verification)

- [ ] **Step 1: Full module compile**

Run: `go build ./...`
Expected: silent success, exit 0.

- [ ] **Step 2: Full module vet**

Run: `go vet ./...`
Expected: silent success, exit 0.

---

### Task 7: Run Go test suite

**Files:** None (verification)

- [ ] **Step 1: Run all Go tests**

Run: `bin/rake test:go`
Expected: PASS. Same test count and same pass rate as the baseline from Task 1. The moved tests (in `framework/minitest/`, `framework/rspec/`) execute under their new paths but produce identical results.

- [ ] **Step 2: If any test fails, diagnose before proceeding**

Common causes if this step fails:
* Missed import update → re-run the grep from Task 5 Step 3.
* `go:embed` resolution broken → verify `framework/rspec/formatter.rb` exists beside `formatter.go` (git mv preserves this).
* Stale test binary cache → `go clean -testcache && bin/rake test:go`.

---

### Task 8: Run full build + Ruby integration suite

**Files:** None (verification)

- [ ] **Step 1: Full `bin/rake`**

Run: `bin/rake`
Expected: PASS. This runs build → lint → install → tests (Go + Ruby integration). Ruby integration tests invoke the installed `plur` binary against fixtures under `fixtures/`; they don't care about internal Go package structure, so they should pass unchanged.

- [ ] **Step 2: Sanity-check installed binary works**

Run: `plur --version && plur -C fixtures/minitest-success --dry-run`
Expected: version prints, dry-run shows detected tests. Confirms the reinstalled binary functions end-to-end.

---

### Task 9: Commit

**Files:** All staged changes from Tasks 3–5.

- [ ] **Step 1: Review the diff**

Run: `git status && git diff --stat`
Expected files changed:
* `framework/framework.go` (3 import lines changed)
* `complexity_test.go` (1 import line changed)
* `benchmark_test.go` (1 import line changed)
* Renames: `minitest/*` → `framework/minitest/*`, `rspec/*` → `framework/rspec/*`, `passthrough/*` → `framework/passthrough/*`

- [ ] **Step 2: Stage and commit**

Run:

```bash
git add -A && git commit -m "$(cat <<'EOF'
refactor: move minitest/rspec/passthrough under framework/

Adapter packages now live as subpackages of their dispatcher. Express
the architecture directly in the directory layout — framework/ owns its
adapters rather than having three sibling packages whose relationship
is only visible by reading imports.

No behavior change. Package names unchanged; only import paths shift
to github.com/rsanheim/plur/framework/{minitest,rspec,passthrough}.
EOF
)"

Expected: commit created, `bin/rake` hook (if any) passes.

- [ ] **Step 3: Confirm clean tree**

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
* Tier 2 internal-ization: move `types/`, `job/` under `internal/`.
* Tier 3 internal-ization: move `framework/`, `watch/`, `logger/`, `config/` under `internal/` (would cascade-move the adapter subpackages along for free — that's the design benefit of starting with this plan).
* Investigate `tmp/review-smaller-binary/` — appears to be a stale snapshot of the codebase polluting grep results; unrelated to this refactor.
