## TX-SCORE-CARD

_template doc for T3 and TX-REFLECT phases as part of `cli_goal.md`_

### Instructions

Evaluate the (current|new) design using the scorecard below on a 1-5 scale, where 1 is actively confusing and bad, and 5 is absolutely amazing and better than 99% of the tools out there. 

#### Scorecard

1. Obviousness - can a new user guess and figure out the tool?
2. Brevity / surface area - few commands, flags, and concepts
3. Default quality - work well out of the box?
4. Conceptual coherence - consistent names, concepts, patterns
5. Feedback quality - does the tool explain whats going on? what to when errors happen?
6. Composability - work well in shells, scripts, agent workflows?
7. Config/API cleanliness - Is config declarative, minimal, predictable, and hard to misuse?

_**For each category, provide:**_

- Score: 1–5
- Evidence: specific command/config/output examples
- Main issue: one sentence
- Suggested improvement: smallest useful change
- Risk/tradeoff: what might get worse

Do not average the scores into a single number. Keep the individual category scores visible.

Then provide:
- Top 3+ design problems
- Top 3+ recommended changes
* maybe 1 or 2*big ticket items* -- i.e. pie in the sky, anythign goes suggestions
- Things that should **not** change

#### Example

* Obviousness
  * Score: 3
  * Evidence: _foo_ is an obvious CLI
  * Main issue: I wish _foo_ did x, y, and z
  * Suggested improvement: _foo_ should wash my dishes
  * Risk/tradeoff: _foo_ might break my dishes

* todo 2
* todo 3
* etc...
