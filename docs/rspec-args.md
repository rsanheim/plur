**Context**
Issue #187 asks for CLI support to pass common RSpec flags (tags, seed, backtrace, fail-fast) without requiring `.plur.toml` job overrides. We are explicitly NOT solving line-number targeting right now, and we are NOT adding a `--rspec-args` flag.

We want a consistent path for all frameworks, so passthrough should work for RSpec, Minitest, and any custom/passthrough job. For phase 1, we will likely add an explicit `--tag` flag for RSpec plus a general `--` passthrough for everything.

**Relevant RSpec Flags (From `rspec --help`)**
These are the flags most likely to be requested and are safe to run before file arguments:
- `-t, --tag TAG[:VALUE]` (repeatable, supports `~slow` exclusion)
- `--seed SEED` and `--order TYPE[:SEED]`
- `-b, --backtrace`
- `--[no-]fail-fast[=COUNT]`
- `--only-failures` and `-n, --next-failure`
- `-p, --[no-]profile [COUNT]`
- `-P, --pattern PATTERN` and `--exclude-pattern PATTERN`
- `-r, --require PATH` and `-O, --options PATH`
- `-f, --format FORMATTER` and `-o, --out FILE`

**Constraints In Plur**
- Plur must inject its JSON formatter and color flags for RSpec parsing.
- RSpec flags must appear before the file list in the final command.
- Passthrough should be framework-agnostic and always provide an escape hatch.
- Duplicate flags should be preserved in the final arg list (no dedupe).

**Options**
1. **Explicit Flags Only**
   - Example: `plur --tag=integration --seed=1234 --backtrace`.
   - Implementation: add fields to `SpecCmd` (e.g., `Tags []string`, `Seed`, `Backtrace`, `FailFast`) and convert to args inserted after RSpec defaults and before files in `BuildRunArgs`.
   - Pros: best UX for common flags; flags are discoverable in `plur --help`.
   - Cons: incomplete coverage; we would need to keep adding flags as requests arrive.

2. **`--` Passthrough Only (Framework-Agnostic)**
   - Example: `plur spec/models -- --tag slow --seed 1234 --backtrace`.
   - Implementation: collect args after `--` and append them after framework defaults and before files.
   - Pros: fully general; minimal maintenance; works for all frameworks.
   - Cons: less discoverable; user must place file patterns before `--` because everything after is treated as framework args.

3. **Hybrid: Explicit `--tag` + `--` Passthrough**
   - Example: `plur --tag=slow --tag=integration spec/models -- --seed 1234`.
   - Implementation: build explicit tag args, then append passthrough args, then append files.
   - Pros: keeps the requested UX for tags while enabling general flags now and in the future.
   - Cons: precedence rules needed if users provide the same flag explicitly and via `--`.

4. **Status Quo (Config Only)**
   - Example: `[job.rspec] cmd = ["bin/rspec", "--tag", "integration"]` + `plur --use=rspec`.
   - Pros: no code changes.
   - Cons: not ergonomic for one-off runs; doesn’t address the issue request.

**Open Questions**
- Should `plur watch` accept and forward the same passthrough args, or keep them `spec`-only?
- Do we want any ordering guarantees between explicit flags and passthrough args beyond “both are inserted before files” (eg explicit first, passthrough second)?

**Decisions**
- Explicit `--tag` errors unless the resolved framework is `rspec`.
- Passthrough args always pass through (framework-agnostic; no validation).
- Formatter/output conflicts are tolerated: Plur injects its formatter first, and any user-specified formatters or output flags are appended. This may produce extra output but should still preserve JSON parsing.

**Recommendation**
Choose Option 3 (Hybrid) and start with explicit `--tag` plus a framework-agnostic `--` passthrough. This matches the desired UX for tags while giving a consistent escape hatch for RSpec, Minitest, and custom jobs. We should:
- Preserve duplicates (append both explicit and passthrough flags).
- Insert framework args after Plur’s injected defaults and before the file list.
- Document that file patterns must appear before `--`.
- Add integration coverage first (dry-run expectations) before wiring behavior.

**Plan**
Phase 1: Specs (Add First)
- [ ] Add integration spec: explicit `--tag` shows up before file args in dry-run output.
- [ ] Add integration spec: passthrough via `--` shows up before file args in dry-run output.
- [ ] Add integration spec: explicit `--tag` errors for non-`rspec` frameworks.

Phase 1: Implementation (Spec Command)
- [ ] Add explicit `--tag` flag to `SpecCmd` (repeatable).
- [ ] Preserve `--` token via Kong passthrough on positional args.
- [ ] Split patterns vs passthrough args at first `--`.
- [ ] Error on explicit `--tag` when framework != `rspec`.
- [ ] Thread passthrough args into `BuildRunArgs` for all frameworks (no dedupe).

Phase 1: Verification
- [ ] Run the new integration spec(s).
- [ ] Run full build (`bin/rake build`).
- [ ] Manual dry-run checks (explicit `--tag`, passthrough `--seed`).

Phase 2: Follow-ups
- [ ] Decide policy for `--format`/`--out` conflicts and add tests if needed.
- [ ] Evaluate passthrough support for `plur watch` and add tests if so.
