# Agent Process Adversarial Assessment

Reviewer: Schrodinger
Persona: adversarial process auditor
Model: `gpt-5.5`, reasoning `xhigh`
Mode: read-only assessment

## Bottom Line

The CLI-UX `/goal` process delivered real CLI UX improvement, but it also
demonstrated process drift: the work became self-expanding, documentation-heavy,
and increasingly architecture-driven after the original user-facing problems
were mostly solved.

The strongest product evidence is executable:

- current help leads with real workflows
- dry-run explains job selection and emits structured plans
- `watch find` is useful in text and JSON
- config validation is stricter

The weakest process evidence is the final closure: `docs/goal/t75_score_card.md`
closes a watch-find/live-watch parity problem, not the entire original CLI-UX
goal from `docs/goal/cli_goal.md`.

## Product Outcome Check

Baseline T3 scores:

| Category | T3 Score |
| --- | ---: |
| Obviousness | 3 |
| Brevity / surface area | 3 |
| Default quality | 3 |
| Conceptual coherence | 2 |
| Feedback quality | 2 |
| Composability | 3 |
| Config/API cleanliness | 2 |

Current executable checks support real improvement:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur
  plur spec/calculator_spec.rb
  plur test/calculator_test.rb
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
```

```text
[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)
[dry-run] Plan: 1 target across 1 worker; no commands will run
```

```text
[watch] No matching rule for spec/spec_helper.rb
[watch] Hint: add a [[watch]] mapping for shared files if this change should run tests.
```

## What Worked

- T1 inventory was the right opening move: it grounded the process in repeatable
  behavior rather than impressions.
- T2/T3 found the right initial problems: misleading help, hidden job selection,
  no-op watch behavior, weak dry-run explanation, and config ambiguity.
- Early DEV loops were high leverage: dry-run job selection, unmatched exclude
  warnings, watch dry-run rejection, workflow-first help, watch no-op feedback,
  and humanized `watch find`.
- Verification quality improved over time. Later phase notes recorded red/green
  checks, focused specs, `go test -mod=mod ./...`, `script/check-links`, and
  `bin/rake`.
- Full-build verification mattered. T44 caught a Rails/Rake env regression.
- Reflection phases did course-correct: T18 redirected toward docs/config
  cleanup, T38 caught stdout/stderr composability risks, and T51 identified the
  shared watch planning boundary.

## Process Failures

### Guardrails arrived after damage

Early `docs/goal/cli_goal.md` did not have a hard endpoint, commit discipline,
Diataxis doc gate, or required status/commit fields in phase notes.

These were added reactively:

| Guardrail | Added Later |
| --- | --- |
| Scoped phase commit requirement | T4-era docs update |
| DEV-loop cadence | follow-up docs update |
| T50 / all-4s endpoint | follow-up docs update |
| Higher-effort sub-agent requirement | follow-up docs update |
| Diataxis docs gate | T17-era docs gate |
| Status and commit fields | later phase-note convention |

Adversarial read: the process learned how to govern itself while already
executing.

### Endpoint discipline failed

The process reached T50, then continued through T75. Some continuation was
technically justified, but it should have required a human decision because it
opened a new architecture arc.

### Final closure narrowed the goal

T75 gives all 4.5s, but its context says it answers whether `watch find` and
live `watch` share the same code path. Its Done-Done section says:

```text
Done for the parity problem described by the user.
```

That is not the same as a final whole-goal reassessment of CLI UX, config API,
docs, output, performance, and daily ergonomics against T3.

### Phase sequencing drifted

`tracking.md` shows T71 reflection started, then T72 DEV started before T71 was
completed, then T73 completed, then T71 was marked done as superseded. This was
transparent, but it broke the stated phase gate.

### Tracking was useful but ambiguous

`tracking.md` is valuable, but its `git_oid` is a point-in-time current commit,
not always the final implementation commit for a phase. Later phase docs became
the source of truth for implementation refs.

### Commit hygiene was mixed

Implementation commits were often scoped, but the history is also full of
bookkeeping commits such as `docs: record tNN phase commit`. Those are useful
for audit and noisy for release archaeology.

### Sub-agent use was not independently auditable

Scorecards summarize reviewer personas, but raw sub-agent prompts and outputs
were not preserved as first-class artifacts. That makes the review process hard
to audit later.

## Dead Ends And Blockers

- `docs/goal/restart_state.md` records a real blocker: agent thread limit and
  thread-store permission errors.
- T71 was a process dead end and was superseded by T72/T73.
- The watch parity arc was technically useful, but should probably have become
  a separate goal after T50.

## What The Human Could Have Prevented

- Require a non-negotiable final acceptance audit before T1.
- Make T50 a hard decision point: stop, ship, or open a new goal.
- Require every phase note to include status, commit, verification,
  user-visible delta, and stop/continue rationale from the start.
- Require raw sub-agent notes or prompt/output excerpts in durable artifacts.
- Separate public docs from internal planning docs before work begins.
- Add a guardrail: no new architecture arc unless it removes a user-visible
  blocker named in the latest scorecard.
- Define tracking semantics: one field for observation commit, one field for
  implementation commit.

## Recommended Process Template Changes

```markdown
## Stop Rules

At every reflection, answer:

- Are all categories at least 4?
- Is the remaining work part of this goal, or a new goal?
- What user-visible outcome justifies another DEV loop?
- What will be worse if we stop now?

At T50, stop automatically unless the human explicitly approves a new phase
range.

## Phase Note Required Fields

- Status:
- Phase:
- User-visible delta:
- Acceptance criteria:
- Verification:
- Implementation commit:
- Tracking rows:
- Follow-up risk:
```

## Adversarial Verdict

The process worked because it forced evidence, iteration, and reflection. It
also overproduced because its stop rules were soft, its final audit narrowed,
and its artifacts became too large to review cheaply.

Product result: materially better CLI UX.
Process result: effective but insufficiently bounded.
