# Watch Mode Terminal Analysis And Options

## Status
Draft for review

## Summary

The recurring `plur watch` prompt/output regressions are not isolated formatting
bugs. They come from a deeper architectural mismatch:

- the prompt is rendered in one place
- logs are written in several other places
- file-triggered work runs off the main watch loop
- subprocess output streams directly to the terminal
- stdin is read in line-buffered mode, so plur does not actually own the input
  buffer or screen state

That combination makes small prompt fixes unstable. Each patch can improve one
write path while another path still interleaves on the same terminal.

The core decision is no longer "where should we put a newline." It is:

- do we keep watch mode as a simple line-oriented CLI and make that explicit, or
- do we invest in a terminal abstraction or library that truly owns prompt
  redraw, partial input, and async output

## What We Learned

### 1. There is no single owner of terminal output

Today, terminal writes come from multiple places with different timing models:

- prompt rendering in `cmd_watch.go`
- debug/info/warn logs through the shared logger
- watcher stderr passthrough in `watch/watcher.go`
- job command echo in `watch.ExecuteJob`
- subprocess stdout/stderr streamed directly to `os.Stdout` / `os.Stderr`

The prompt queue in `cmd_watch.go` helps only with the narrow case of prompt
display. It does not serialize all other writes around that prompt.

### 2. File-triggered runs are asynchronous, but we render them like typed input

The recent watch changes tried to make the echoed command appear as if it were
continuing the prompt line, which looks nice in the happy path:

```text
[plur] > rspec spec/foo_spec.rb
```

That model breaks down as soon as the run is not initiated by the user pressing
Enter at the prompt. File-triggered jobs start asynchronously. If the user has
already started typing `help`, `reload`, or `debug`, the file-triggered command
is not a continuation of their input. It is an interrupting background event.

Rendering asynchronous work as prompt input is the wrong mental model unless
plur fully owns the terminal and can clear/redraw the current line safely.

### 3. `bufio.Scanner(os.Stdin)` means plur does not know partial input

The current watch loop reads stdin one completed line at a time. Until Enter is
pressed, plur does not know whether the user has typed nothing, `h`, or
`reload`.

That means plur cannot reliably:

- preserve partially typed input
- clear and redraw the current prompt line
- restore the user's text after async output

With the current input model, any attempt to "protect" partially typed input is
best-effort only. It cannot be fully correct.

### 4. The debouncer moved execution off the main event loop

The original watch loop moved toward a single prompt/display owner. The later
debouncer simplification improved the event-processing code, but it also moved
file-triggered execution into the timer callback path instead of routing it back
through the main loop.

That made terminal behavior harder to reason about:

- the main loop prints the prompt
- the debouncer callback executes jobs
- the job path prints command lines and runs subprocesses
- logs can arrive during both

Functionally this works. Interactively it means the screen is shared by
concurrent actors.

### 5. Our tests mostly validate pipes, not an interactive TTY

The current Ruby watch harness is Open3-based and is good for many integration
cases, but the most painful regressions are specifically interactive:

- a prompt is visible
- the user has partially typed input
- a watcher lifecycle or file event arrives
- a job starts while the prompt is on screen

Those cases need PTY-oriented tests or a terminal abstraction with fakeable
screen behavior. Pipe-based tests do not fully cover them.

## Why This Keeps Flaking Back And Forth

These regressions are oscillating because the system boundary is unclear.

We keep applying local fixes to symptoms:

- add a leading newline before a command echo
- remove that newline so the command looks like prompt input
- add a newline before one class of debug log
- remove duplicate warnings from one failure path

Each of those can be individually reasonable. None answers the real question:
"who owns the screen while watch mode is running?"

Until that ownership is explicit, future prompt/output changes will tend to
re-break adjacent cases.

## Current Constraints

Any proposed solution needs to respect these realities:

- watch mode has asynchronous file events
- jobs may produce arbitrary stdout/stderr
- some jobs may change terminal mode and require recovery
- reload behavior already exists and must stay safe
- non-interactive test runs still matter
- experimental Windows support means platform-specific shell assumptions should
  be minimized

## Solution Options

### Option 0: Keep patching formatting locally

Description:

- continue adjusting where newlines are emitted
- special-case more event types before prompt display
- tweak command echo formatting case by case

Pros:

- smallest immediate code changes
- no architectural work

Cons:

- does not solve the ownership problem
- will keep regressing when new write paths are added
- cannot truly preserve partially typed input
- keeps terminal behavior implicit and fragile

Assessment:

Not recommended. This is the path that created the current loop.

### Option 1: Stay line-oriented, but serialize watch output through the main loop

Description:

- keep watch mode as a simple line-oriented CLI
- route debounced file batches back to the main watch loop over a channel
- ensure the main loop is the only place that starts watch-triggered jobs,
  prints watch-owned status lines, triggers reload, and redraws the prompt
- treat async file-triggered runs as their own output lines, not as typed prompt
  input
- document that background runs may interrupt a partially typed command

What this means in practice:

- manual `[Enter]` runs can still look shell-like
- asynchronous file-triggered runs should start on a fresh line with a stable
  prefix
- prompt redraw happens after the run completes
- logs that are owned by watch mode should also go through the same serialized
  path where practical

Pros:

- smallest holistic fix
- preserves the current overall architecture
- significantly reduces prompt/log interleaving
- easier to test because behavior is routed through one event loop
- no large third-party dependency required

Cons:

- still does not preserve partially typed input
- user experience remains "interruptible line mode," not a fully managed shell
- some writes still come from subprocesses directly unless we further wrap them

Assessment:

This is the best near-term stabilization path if the goal is to make watch mode
predictable without turning it into a larger UI project.

### Option 2: Build a small internal terminal abstraction for watch mode

Description:

- introduce a focused watch-mode terminal owner, likely a small internal package
- give it responsibility for:
  - prompt state
  - suspending the prompt before async output
  - printing watch-owned messages consistently
  - redrawing the prompt afterward
- keep the rest of watch mode event-driven, but stop allowing arbitrary watch
  code to print directly to the terminal
- optionally keep line-buffered stdin at first, then add raw-mode input later if
  needed

A small abstraction could expose a limited surface such as:

- `ShowPrompt()`
- `SuspendPrompt()`
- `PrintEvent(...)`
- `PrintJobStart(...)`
- `PrintJobEnd(...)`
- `PrintReload(...)`

Pros:

- gives terminal behavior a clear owner without committing to a full TUI
- creates a seam for better tests
- supports incremental migration
- keeps the dependency surface small
- can later swap input handling without rewriting watch event flow

Cons:

- we become responsible for terminal edge cases ourselves
- if we later need full input editing/redraw, the abstraction may grow into a
  homegrown mini-TUI
- still cannot preserve partially typed input until we replace the current
  stdin model

Assessment:

This is the best medium-term architecture if we believe watch mode should remain
simple but deserve a real terminal boundary.

### Option 3: Adopt a proper line-editor or TUI runtime

Description:

- move watch mode onto a library that already manages terminal state, redraw,
  input, and async updates
- two sub-variants are relevant:
  - a line-editor style library if the goal is mainly reliable prompt editing
  - a full TUI runtime if we want watch mode to become a richer interactive
    interface

This path is justified if we want plur to truly own the terminal experience
instead of approximating it.

Pros:

- strongest foundation for prompt redraw and partial-input preservation
- clearer mental model: one runtime owns screen state
- better fit if watch mode will grow into a richer experience later
- often comes with testing tools or established patterns for async UI updates

Cons:

- largest architectural shift
- more dependency surface and more framework conventions
- subprocess streaming, reload semantics, and non-interactive behavior all need
  careful adaptation
- easy to overbuild if all we really need is a robust one-line prompt

Assessment:

This is the right path only if we want first-class interactive UX. If the real
goal is just "stop stomping on the prompt," a full TUI may be too much.

## Tradeoff Summary

### If we optimize for fastest stabilization

Choose Option 1.

That gives us a clear rule:

- the main watch loop owns watch-triggered execution and prompt redraw
- async runs are rendered as async output, not prompt input

This fixes the biggest source of regressions without pretending we support
terminal behavior that we do not actually control.

### If we optimize for a clean internal architecture

Choose Option 2.

That gives watch mode an explicit terminal boundary and leaves room to improve
input handling later. It is the strongest "build our own small abstraction"
option.

### If we optimize for polished interactive UX

Choose Option 3.

That is the only path that honestly supports preserving partially typed input
while async events happen. If we want a watch experience that behaves like a
real terminal application, we should use a runtime designed for that.

## Recommended Direction

Recommendation:

1. Stop trying to render asynchronous file-triggered jobs as if they were typed
   into the prompt.
2. Make the main watch loop the owner of watch-triggered execution again.
3. Add PTY-focused acceptance coverage for the cases that keep regressing.
4. After that stabilization step, decide between:
   - Option 2 if we want a small internal watch terminal layer
   - Option 3 if we want true prompt redraw and partial-input preservation

In other words:

- short term: Option 1
- likely medium term: Option 2
- long term only if justified by product direction: Option 3

This recommendation avoids both extremes:

- it does not keep papering over the problem with local newline fixes
- it does not force a TUI migration before we know we need one

## Suggested Acceptance Criteria For The Next Iteration

Before changing code again, agree on these behavioral rules:

### Rule 1: One watch-loop owner

All watch-owned output that affects prompt rendering must be serialized through
one coordinator.

### Rule 2: Async work is not prompt input

File-triggered runs must not be rendered as if the user typed them at the
prompt unless plur fully owns and redraws the input line.

### Rule 3: Partial input support is explicit

Either:

- we explicitly do not preserve partially typed input in line mode

or

- we adopt tooling that makes preservation a real guarantee

### Rule 4: TTY cases must have TTY tests

Add acceptance coverage for:

- watcher lifecycle logs while prompt is visible
- user types partial input, then a file-triggered run starts
- failing watch-triggered job
- reload path while prompt is visible

## Decision Guide

Use these questions to choose the next step:

### Choose Option 1 if:

- we want the smallest credible fix
- we can accept that partial input may be interrupted
- we mainly need predictability, not polish

### Choose Option 2 if:

- we want a durable watch-mode boundary
- we expect more watch UI work in this codebase
- we want to improve testability before adopting a bigger library

### Choose Option 3 if:

- we want to preserve partially typed input correctly
- we want richer interactive watch UX in the future
- we are willing to treat watch mode as a real terminal application

## Final Take

The missing piece is not another newline. It is explicit terminal ownership.

If we stay with `bufio.Scanner` and ad hoc prints, watch mode should behave like
a simple interruptible CLI and say so in code and tests. If we want shell-like
prompt behavior during async events, we need either a real terminal abstraction
or a real terminal runtime.
