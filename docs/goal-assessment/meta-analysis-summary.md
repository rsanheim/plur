# CLI-UX Goal Process Meta-Analysis

## Summary

The original CLI-UX goal worked as an evidence-driven improvement process, but
it was too loosely bounded. It produced a materially better CLI and a strong
audit trail. It also generated a lot of internal documentation, noisy commit
history, and a final closure that was narrower than the original whole-goal
scope.

Best single-sentence assessment:

> The process was effective at finding and fixing real UX problems, but future
> goals need stronger stop rules, less public-docs process churn, and a
> required broad final audit.

## What Worked

### The initial inventory was the right foundation

T1/T2/T3 established a concrete baseline before changing behavior. The process
used docs, code, dry-runs, integration specs, and command output to identify
the real weak spots:

- command-first help hid commandless daily use
- dry-run showed commands but not the selected job/reason
- watch dry-run semantics were confusing
- `watch find` was useful but not structured or fully live-like
- config validation was too permissive
- output contracts were not consolidated

### The scorecard kept user-facing quality visible

The stable categories from `docs/goal/tx_score_card.md` worked well:

- Obviousness
- Brevity / surface area
- Default quality
- Conceptual coherence
- Feedback quality
- Composability
- Config/API cleanliness

They prevented the work from becoming only a code cleanup. The categories kept
pressure on docs, output, config, and automation surfaces.

### Small DEV loops made progress auditable

Small phases helped isolate behavior changes:

- explain dry-run job selection
- warn on unmatched excludes
- reject watch dry-run with guidance
- lead help with common workflows
- humanize `watch find`
- add dry-run JSON
- clarify output contracts
- reject unknown config keys
- share watch session/planner/execution plans

This made regressions easier to detect and made review more targeted.

### Reflection phases caught course corrections

Important examples:

- T18 redirected from output features toward docs/config cleanup.
- T38 caught stdout/stderr composability risks.
- T47/T58 kept config/API cleanliness from being hand-waved.
- T51 identified that watch parity needed a shared session/planner boundary.
- T73/T74 caught and fixed the remaining execution-plan job-key and ignore-glob
  issues.

### Verification discipline improved over time

Later phases regularly recorded:

- red/green behavior checks
- focused integration specs
- `go test -mod=mod ./...`
- `script/check-links`
- `bin/rake`
- explicit implementation commit refs

## What Did Not Work

### Stop rules were too soft

The goal specified T50 or an all-4/5 reflection as an endpoint, but the work
continued through T75. T51-T75 produced good architecture, but it functioned as
a new watch-parity goal.

Future goals should force a human decision at major thresholds:

1. Stop and ship.
2. Continue the same goal for a named reason.
3. Open a new goal.

### Final closure narrowed the scope

T75 closed the watch-find/live-watch parity issue. It did not, by itself,
perform a broad final audit of the original CLI-UX objective. This new
assessment fills that gap.

Future final reflections should explicitly re-score the whole original goal
against the original baseline and should include release-readiness,
documentation, and performance evidence.

### Process docs became too large

`docs/goal/**` is valuable internal process evidence, but it is not durable
public product documentation. It also made release-note archaeology harder.

Future goal docs should default to `../plur-internal`, with only final product
docs copied into the public repo.

### Tracking semantics were ambiguous

`tracking.md` records point-in-time `git_oid` values, not always final
implementation commits. Later phase docs added explicit commit fields, which
helped.

Future tracking should distinguish:

- observation commit
- implementation commit
- docs/process commit

### Sub-agent evidence was not preserved early

Scorecards cite reviewer personas, but raw prompts and outputs were not stored
as durable artifacts during the original goal. This assessment stores agent
notes directly under `docs/goal-assessment/` to correct that pattern.

### Phase sequencing drifted

T71 was started, superseded by review follow-up, and later marked done after
T73. This was transparent but broke strict phase ordering.

Future process should allow explicit states like:

- superseded
- reopened
- replaced-by

instead of forcing everything through start/done.

## Blockers And Recurring Friction

| Area | What Happened | Lesson |
| --- | --- | --- |
| Sub-agent infrastructure | Restart state recorded thread-limit and permission failures. | Long goals need a fallback or a hard stop rule when sub-agents are required. |
| Kong help behavior | Several phases handled help hiding, flag rejection, and parser behavior. | Help visibility and parser behavior should be changed together. |
| Watch parity | Early work patched symptoms before T51 named the shared planning boundary. | Repeated drift between paths signals an architecture review, not another local patch. |
| Config schema | Initial validation was tied too closely to CLI/parser shape. | Persistent config schema should be owned explicitly. |
| Docs publication hygiene | Internal process docs lived under `docs/`. | Internal planning belongs in `../plur-internal` unless meant for public docs. |

## What To Keep Next Time

- Start with an executable inventory.
- Use the same scorecard through the whole goal.
- Keep small implementation loops.
- Require focused tests for CLI contracts and output behavior.
- Use full `bin/rake` gates before broad completion claims.
- Include sub-agent review at reflection checkpoints.
- Keep Diataxis as the docs gate.
- Record exact verification commands and outcomes.

## What To Change Next Time

1. Add hard stop rules before work starts.
2. Add a "release note / migration impact" field to every DEV phase.
3. Keep raw sub-agent prompts and outputs as durable artifacts.
4. Move process docs to `../plur-internal` by default.
5. Require a broad final audit before marking the whole goal complete.
6. Distinguish product commits from bookkeeping commits in tracking.
7. Treat new architecture arcs as new goals unless the latest scorecard names
   them as required to finish.
8. Require one "what should we stop doing?" answer in every reflection.

## Proposed Future Goal Template Additions

```markdown
## Stop Rules

At every reflection, answer:

- Are all categories at least 4?
- Is the remaining work still part of this goal?
- What user-visible outcome justifies another DEV loop?
- What will be worse if we stop now?

At the named endpoint, stop automatically unless the human explicitly approves a
new phase range.

## Phase Note Required Fields

- Status:
- Phase:
- User-visible delta:
- Release note / migration impact:
- Acceptance criteria:
- Verification:
- Implementation commit:
- Tracking rows:
- Follow-up risk:
```

## Final Verdict

Product outcome: strong success.
Process outcome: effective but overextended.
Next-process priority: preserve the evidence-driven loop while making stop
rules, artifact location, and final audits much stricter.
