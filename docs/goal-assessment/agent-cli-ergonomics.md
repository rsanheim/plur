# Agent CLI Ergonomics Assessment

Reviewer: Curie
Persona: CLI UX researcher focused on daily developer ergonomics
Model: `gpt-5.5`, reasoning `high`
Mode: read-only assessment

## Scope

Curie compared:

- baseline `v0.56.0`
- current `main`
- target `v0.60.0-rc.1`
- current `prep-goal` checkout

The review used `docs/goal/tx_score_card.md`, current CLI output, goal
scorecards, `docs/goal/new_design.md`, and the tracking log.

## Executive Assessment

The CLI-UX goal materially improved daily ergonomics. The biggest upgrade is
not one feature; it is the shift from hidden inference to explicit plans.

Observed improvements:

- Help leads with real workflows: `plur`, a target path, `--dry-run`, `watch`,
  and `watch find`.
- Dry-run explains selected job, framework, reason, target count, worker count,
  and command plan.
- Machine-readable previews now exist for one-shot runs and watch mapping
  previews.
- Watch preview and live watch now share session, admission, planning, and
  execution-plan code.
- Config moved from permissive parser-shaped TOML to a stricter persistent
  schema.

Remaining weakness: the visible CLI surface is still broad. Help grouping makes
it manageable, but Plur still exposes commandless run, `spec`, `watch run`,
`watch find`, `watch install`, `rails`, `rake`, `doctor`, `config init`,
`rails:init`, `version`, and several mode-specific flags.

## Scorecard

| Category | Baseline | Target | Assessment |
| --- | ---: | ---: | --- |
| Obviousness | 3 | 4.5 | Help now starts with `Usage: plur [patterns...] [flags]` and common workflows. |
| Brevity / surface area | 3 | 4 | Better grouping and flag pruning; still not a tiny command surface. |
| Default quality | 3 | 4.5 | RSpec/default dry-run and watch preview behavior are clearer and safer. |
| Conceptual coherence | 2 | 4.5 | Job, target, watch rule, plan, and execution plan are now visible and aligned. |
| Feedback quality | 2 | 4.5 | Warnings, no-op watch feedback, plain errors, and config errors are more actionable. |
| Composability | 3 | 4.5 | JSON dry-run and watch-find plans are useful for shells and agents. |
| Config/API cleanliness | 2 | 4.5 | Strict config keys and CLI-only preview controls are a clean break. |

## Category Notes

### Obviousness

Baseline issues:

- T3 scored obviousness at 3.
- Earlier help said `Usage: plur <command> [flags]` even though commandless
  `plur` was the happy path.
- `plur test` behaved as a target path but looked like a command.

Target evidence:

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

Remaining issue: users still need to distinguish command names from target
paths.

### Brevity / Surface Area

Baseline issues:

- T3 identified a flat surface of `spec`, `watch run`, `watch install`,
  `watch find`, `rails`, `rake`, `doctor`, `config init`, `rails:init`, and
  many global flags.
- `v0.56.0` exposed an unused global `--json` flag.

Target evidence:

- Workflow-first help and grouped commands.
- No-op inherited flags are now hidden or rejected for watch modes.
- Removed `--json` now gets direct guidance.

Remaining issue: the surface is better ordered, but still not small.

### Default Quality

Baseline issues:

- Explicit non-test files could become RSpec targets after autodetection.
- Excludes could match nothing without feedback.
- `plur --dry-run watch` started watcher setup.

Target evidence:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
[dry-run] Plan: 1 target across 1 worker; no commands will run
[dry-run] Commands:
```

Remaining issue: helper/support-file watch changes still require explicit
`[[watch]]` mappings for meaningful runs.

### Conceptual Coherence

Baseline issues:

- T3 scored conceptual coherence at 2.
- `job` and `framework` existed internally, but dry-run mostly showed
  `[rspec]`.
- `{{target}}` behavior differed by mode and was not well explained.

Target evidence:

```text
[watch] Checking lib/calculator.rb
[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)
[watch] Would run job rspec with spec/calculator_spec.rb
[watch] Command: bundle exec rspec spec/calculator_spec.rb
```

Remaining issue: one-shot JSON uses `--dry-run-format=json`; watch preview uses
`watch find --format=json`.

### Feedback Quality

Baseline issues:

- T3 scored feedback quality at 2.
- Dry-run showed worker commands but not why that plan was chosen.
- Watch no-ops were quiet.
- Expected user errors could appear as timestamped internal logs.

Target evidence:

- Plain `Error: ...` for command errors.
- Worker stderr no longer replays on stdout.
- Helper-file no-rule changes include a hint.
- Config and watch-glob errors are explicit.

Remaining issue: JSON-mode parser/config errors remain prose on stderr, not
structured JSON.

### Composability

Baseline issues:

- T3 scored composability at 3.
- Dry-run and watch-find were human-text oriented.

Target evidence:

```json
{
  "version": 1,
  "mode": "watch_find",
  "existing_targets": {
    "rspec": ["spec/calculator_spec.rb"]
  },
  "job_plans": [
    {
      "job": "rspec",
      "targets": ["spec/calculator_spec.rb"],
      "argv": ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
      "env": [],
      "shell": "bundle exec rspec spec/calculator_spec.rb"
    }
  ],
  "exit_code": 0
}
```

Remaining issue: scripts should use `argv` and `env`; `shell` is only human
convenience.

### Config/API Cleanliness

Baseline issues:

- T3 scored config/API cleanliness at 2.
- Unknown keys were not hard failures.
- Preview/session controls could be persisted.

Target evidence:

- Strict documented config keys.
- `dry-run` and `dry-run-format` rejected in TOML.
- TOML schema is independent from Kong command surfaces.

Remaining issue: this is a breaking cleanup, so release notes need clear
migration guidance.

## Goal Process Notes

What worked:

- The inventory harness created repeatable baseline evidence.
- T3 made weak areas explicit before implementation.
- Small DEV/REFLECT loops prevented one large rewrite.
- Tracking rows and scoped commits made the progression auditable.
- Persona reviews caught course corrections, especially watch parity and config
  cleanliness.
- The Diataxis gate helped separate workflow docs from reference docs.

What did not work:

- The process generated a lot of internal docs and scorecards.
- Some reflections were implementation-adjacent rather than independent product
  judgment.
- Tracking rows used point-in-time short OIDs; later docs needed explicit
  implementation refs to avoid ambiguity.
- T71 was superseded by T72/T73, showing that review follow-up can drift from
  the nominal phase machine.
- Watch work improved symptoms before T51 named the larger shared-boundary
  problem.

Recommended next-process guardrails:

- Start with a runnable inventory harness.
- Keep the scorecard short and stable.
- Record implementation commit refs in each reflection.
- Treat help hiding and behavior rejection as one unit.
- Put machine-output contracts in one reference doc.
- Preserve raw sub-agent outputs for audit.
