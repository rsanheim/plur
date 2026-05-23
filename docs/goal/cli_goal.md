# plur goal

 /goal Improve Plur’s CLI, configuration api, and overall UI/UX so every day usage for parallel specs and watch feels satisfying, consistent, obvious, and hard to misuse. 

The **final** goal should be a cleaner user-facing design that is easier to use, human-centered, powerful, and awesome for day to day usage. Prefer removing concepts, collapsing options, and making defaults smarter over adding new abstractions. 

### HOW TO WORK THRU THIS GOAL

Work thru the named phases below in order. Our convention for phase names is `TX-CODE`, where `X` is an integer, and code is short code for the phase. 

Read the instructions and follow them -- the phases each have a different focus. Some are writing docs and doing research and review, while others are implementation focused. Show your evidence, notes, and results in the referenced md documents -- always use [hyperlinks](https://example.com) for web references, and include file paths for logs, artifacts, etc. Prefer using the repository local ./tmp/ dir for artifacts where possible.

For each significant status (i.e start|done on a phase) you **must** annotate via `script/track-goal`. Run `script/track-goal --help` for info. 

```bash
script/track-goal --current TX-FOO -s done >> tracking.md
```

After a phase reaches its SUCCESS GATE and is tracked as `done`, make a scoped
git commit for that phase before starting the next phase. Keep unrelated local
changes out of that commit.

You must NOT move past a phase until the SUCCESS GATE is met. If you are truly blocked, make a note of in your current phase block, and try a different approach.

_Phase Sequencing_

1. T1-INV
2. T2-CURR-REVIEW
3. T3-SCORE-CARD
4. T4-DEV, T5-DEV, T6-DEV, etc..._(repeat TX-DEV phases 3 to 6 times)_
5. T7-REFLECT
6. repeat TX-DEV loop, aka T8-DEV, T9-DEV, T10-DEV, etc...
7. repeat TX-REFLECT
etc...

Continue alternating cycles of multiple TX-DEV phases, followed by a TX-REFLECT phase to assess progress, until the goal is achieved.


_Markdown docs_ these are all in current dir: `plur/docs/goal`

* `cli_goal.md` - this file - the overall goal outline and instructions
* `cli_inventory.md` - common plur commands to inventory/audit
* `current_design.md` - template doc for current design analysis and research
* `next_design.md` - template for design improvements, changes, specs
* `tx_score_card.md` - template for score card evaluation and review/reflection

### T1-INV

Start by inventorying current target discovery/selection behavior with evidence. Use docs, code, dry-runs, focused tests, or backspin where useful. Then compare that current behavior to the desired design.

Inventory at least the [commands in cli_inventory](./cli_inventory.md) cases in an automated, repeatable manner: we have many ruby integration specs that exercise much of this, but having one or two "current state" rspecs would be very helpful in seeing the *current* capabilities and gaps and confusing bits.  Most of them following can be exercised with `--dry-run` to safely demo what they do:

**T1 SUCCESS GATE:** all above surfaces are clearly, concisely exercised via an automated script that supports a `PLUR_DEMO` mode to demonstrate each specific call, the intention and the dry-run output"

### T2-CURR-REVIEW

Conduct a full UX, design, architecture review of the pros/cons/gaps/confusing parts of the current state, using the above tooling from T1, as well as interactive use. Do this with sub-agents of different personas. Spend time thinking, researching, and reflecting on the current state of Plur compared to some beloved tools for inspiration (these are just touchpoints, many of them do very different things than plur)

- watchexec: simple watch/run mental model
- vitest: distinction between watch mode and one-shot run
- ripgrep: excellent defaults with escape hatches
* fd: humane replacement for `find`
- gh: domain nouns as subcommands

**T2-CURR-REVIEW SUCCESS GATE**  

completed **current_design.md*** *doc compiling feedback, analysis, and research from the review. 

### T3-SCORE-CARD

Evaluate the current_design using the scorecard below on a 1-5 scale, where 1 is actively confusing and bad, and 5 is absolutely amazing and better than 99% of the tools out there. Let me level-set with you: I think plur is probably no higher than a **3** on most of these currently.

* Obviousness - can a new user guess and figure out the tool?
* Brevity / surface area - few commands, flags, and concepts
* Default quality - work well out of the box?
* Conceptual coherence - consistent names, concepts, patterns
* Feedback quality - does the tool explain whats going on? what to when errors happen?
* Composability - work well in shells, scripts, agent workflows?
* Config/API cleanliness - Is config declarative, minimal, predictable, and hard to misuse?

For each category, provide:
- Score: 1–5
- Evidence: specific command/config/output examples
- Main issue: one sentence
- Suggested improvement: smallest useful change
- Risk/tradeoff: what might get worse

Do not average the scores into a single number. Keep the individual category scores visible.

Then provide
* Top 3+ design problems
* Top 3+ recommended changes
* maybe 1 or 2 *big ticket items* -- i.e. pie in the sky, anythign goes suggestions
* Things that should **not** change

**T3-SCORE-CARD SUCCESS GATE**:  complated score card as `t3_score_card.md` - use template `tx_score_card.md`

---------------------------------------------------------------------------------------------------------------

**IMPORTANT** After T3 we now enter an repeating loop, so the next phase is T4, then T5, and so on ....

---------------------------------------------------------------------------------------------------------------

### T[X]-DEV

Plan one small concrete removal, consolidation, change, or fix, starting from the **user-facing experience** of using plur, using plain language. In doc `new_design.md` describe the change with less than 1000 words, including:

* a pain point or job to be done
* brief overview/spec of proposed change
* acceptance criteria 

Implement the change! Maintain high quality standards, simple design, fewest abstractions / componenents that coudl possibly work.

- Do not add another core abstraction or configuration point **without** first looking for things that overlap or do something simliar -- if there *is* something old and crufty that is modeling something similiar, prefer removing it first or replacing
- On the other hand, don't be boxed in by bad abstractions -- if a key abstraction feels wrong, its fine to take a big swing! Just be explicit and thoughtful about it Write down the problem, the proposed solution, and tradeoffs and success criteria.
- Do not worry about backward compatibility - just clearly call out breaking changes.
- Note that docs, internal refactoring, test coverage, research dives, spikes, performance work, and tooling are all also valid followup items to focus on here -- they should just be in service of the overall human-centered facing goals.
- random items to consider:
  - borrow/steal ideas from bacon, or watchexec, or ???
  - KongCLI has been a pain, maybe try a switch?
  - is TOML too much of a pain ?
  - consolidation of `plur spec` and `plur watch` ? would be more consistent, could allow parallel runs in watch mode
  - TUI interface for watch mode? maybe a requirement for proper parallel runs in watch
  - better terminal and interactive menu for watch
  - automatic tuning of workers based on machine, project size, files, etc
  - automatic bisecting
  - agent friendly output for `plur spec` - i.e. minimal tokens, failures immediately, etc
  - what would it take allow plur to be a drop-in for rust, JS, python, etc, at least for the simplest use cases?


When done, include before/after evidence of the improvement in the **new_design** doc, calling out tradeoffs and follow up items.

**TX-DEV SUCCESS GATE** Improved ux, CLI, config, or simplfiied interface . Tests and lints are passing. No significant perf or behavior regressions **unless** they are specificaly called out for a follow up phase.

---

**REPEAT TX-DEV** between **2 to 5 times** - picking a new or followup improvement, removal, refinement. Repeat the dev cycles, following all the steps from the above TX-DEV plan, before continuing to **TX-REFLECT**

----

### TX-REFLECT

Time to step back, assess where **new-design** is, and evaluate where we stand.

1) Refresh your memory on the original **current_design**  -- this is the original design, from back when we started
2) Refresh your memory on where we are now, in the actual executable and in the **new_design.md**
3) Evaluate and review  the **new_design** using the **same scorecard** as before on a 1-5 scale. Same criteria as before: Obviousness, Brevity, Default quality, Conceptual coherence, Feedback quality, Composability, and Config/API cleanliness. Each criteria gets a score, no average, and write it up in `tx_score_card.md`
4) Ask yourself and 1-3 sub-agents: are we moving in the right direction? Is the design and ux and experience of using plur trending in the right direction? Do we need to make a course-correction? Is there a change we made that is just not working? Is something holding us back?
5) Add your thoughts, reflections, and planned course corrections in `tx_score_card.md`

ARE YOU DONE DONE?
  * latest score card is all 4s and 5s across the board
  * the new design and interface is a clear, obvious, substantial improvement
  * all functionality works as expected, ideally better than before
  * full build passes, full QA cycle, performance same or better
  * IMPORTANT: assume this will take at least 15-20+ iterations to DONE DONE!

OTHERWISE, GOTO **TX-DEV** loop and repeat
