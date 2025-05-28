# Plan: Implement Streaming JSON Output for Rux

## Problem Statement

Rux currently uses dual RSpec formatters (`--format progress --format json`), which causes:
- 2x CPU usage compared to turbo_tests (15s vs 7.4s user time)
- Performance degradation as RSpec formats output twice
- Visible pausing/locking during parallel execution

Turbo_tests avoids this by using a single custom JSON formatter that streams results line-by-line.

## Goal

Implement a streaming JSON approach similar to turbo_tests to:
1. Eliminate double formatting overhead
2. Maintain real-time progress feedback
3. Achieve performance parity with turbo_tests

## Current Architecture

### Rux (Current)
```
RSpec process → progress formatter → stdout (dots)
              → JSON formatter → file → parse after completion
```

### Turbo_tests
```
RSpec process → JsonRowsFormatter → stdout (line-by-line JSON) → parse in real-time
```

## Implementation Tasks

### Phase 1: Research & Preparation

- [x] Study turbo_tests' JsonRowsFormatter implementation
  - Location: `references/turbo_tests/lib/turbo_tests/json_rows_formatter.rb`
  - Understand the JSON message format
  - Understand how it hooks into RSpec events

* [x] analyze `git status` - lets get to a clean state before starting
* [x] Add the meta-rspec gem (git@github.com:rspec/rspec.git) as another 'reference' repo, so we can research its context as necessary. Add it as a git subtree underneath references/rspec
- [x] Create a JsonRowsFormatter in ruby (basically emulate TurboTests formatter for now)
  - Created at `rux/lib/rux/json_rows_formatter.rb`
* [x] Add some specs for the formatter in isolation
  - Created at `rux/spec/json_rows_formatter_spec.rb`
* [x] Pause and analyze how to best refactor to a "single stream" per rspec rux - we probably want to update this plan at that point
  - See detailed refactoring plan in `docs/single-stream-refactor-plan.md`

#### JSON Message Format (from turbo_tests)

Each line contains: `ENV["RSPEC_FORMATTER_OUTPUT_ID"] + JSON_OBJECT`

Message types and their structures:

```json
// Type: load_summary
{"type": "load_summary", "summary": {"count": 10, "load_time": 0.123}}

// Type: group_started/finished
{"type": "group_started", "group": {"group": {"description": "MyClass"}}}

// Type: example_passed
{"type": "example_passed", "example": {
  "execution_result": {
    "status": "passed",
    "example_skipped?": false,
    "pending_message": null,
    "pending_fixed?": false,
    "exception": null
  },
  "location": "./spec/foo_spec.rb:10",
  "description": "does something",
  "full_description": "MyClass does something",
  "metadata": {...},
  "location_rerun_argument": "./spec/foo_spec.rb[1:1]"
}}

// Type: example_failed
{"type": "example_failed", "example": {
  "execution_result": {
    "status": "failed", 
    "exception": {
      "class_name": "RSpec::Expectations::ExpectationNotMetError",
      "message": "expected: 3\n     got: 2",
      "backtrace": [...],
      "cause": null
    }
  },
  // ... other fields same as example_passed
}}

// Type: seed
{"type": "seed", "seed": 12345}

// Type: close
{"type": "close"}
```

Key insights:
- Each JSON object is on a single line (no pretty printing)
- Uses `output.flush` after each line for real-time streaming
- Prefixed with separator for easy parsing in multiplexed output
- Minimal data sent (only what's needed for reporting)

- [x] Research RSpec custom formatter requirements
  - How to package and distribute a custom formatter
  - How to load it from rux

#### RSpec Formatter Loading Options

RSpec can load custom formatters in several ways:

1. **Direct class reference**: `--format MyModule::MyFormatter`
   - Requires the formatter class to be in Ruby's load path
   - Used by turbo_tests as a gem

2. **Require and format**: `-r ./path/to/formatter.rb --format MyFormatter`
   - Loads a specific file before running tests
   - Good for local formatter files

3. **Full path to formatter**: `--format /absolute/path/to/formatter.rb`
   - RSpec will require the file and use the formatter

For rux, the options are:

**Option A: Embedded Formatter (Recommended)**
- Embed formatter Ruby code as a Go string constant
- Write to a temp file before running tests
- Use `-r ./tmp/rux_formatter.rb --format RuxFormatter`
- Pros: Self-contained, no external dependencies
- Cons: Need to manage temp file lifecycle

**Option B: Generate on First Run**
- Generate formatter in `~/.rux/formatter.rb` on first use
- Reuse for subsequent runs
- Pros: Only write once, can be user-customized
- Cons: Version management complexity

**Option C: Separate Gem**
- Create `rux-formatter` gem
- Users must `gem install rux-formatter`
- Pros: Standard Ruby distribution
- Cons: Extra installation step, version synchronization

### Phase 2: Create Custom RSpec Formatter

- [ ] Create a new Ruby gem or embedded formatter for rux
  - Option A: Embed Ruby code in rux binary
  - Option B: Create separate `rux-formatter` gem
  - Option C: Generate formatter file on-the-fly

- [ ] Implement streaming JSON formatter
  - Output one JSON object per line
  - Include all necessary test result data
  - Ensure proper stdout flushing for real-time streaming

- [ ] Test formatter independently with RSpec

### Phase 3: Update Rux Runner

- [ ] Remove current dual formatter approach
  - Update RunSpecFile to use single custom formatter
  - Remove JSON file creation/parsing logic

- [ ] Implement JSON stream parser
  - Read stdout line-by-line
  - Parse each JSON message as it arrives
  - Update progress display in real-time

- [ ] Handle progress output
  - Generate dots/F's based on JSON events
  - Maintain current colorization logic
  - Ensure thread-safe output

### Phase 4: Optimize & Test

- [ ] Remove output mutex bottlenecks
  - Only lock when necessary
  - Consider lock-free alternatives

- [ ] Benchmark against turbo_tests
  - Target: Match turbo_tests performance
  - Measure CPU usage reduction

- [ ] Update integration tests
  - Ensure all existing tests pass
  - Add tests for streaming behavior

## Technical Considerations

1. **Formatter Distribution**
   - How to ensure the custom formatter is available to RSpec?
   - Consider embedding Ruby code as Go string constant
   - Or dynamically generate formatter file in tmp directory

2. **Backward Compatibility**
   - Keep current JSON file output as fallback option?
   - Add flag to choose between streaming and file-based output?

3. **Error Handling**
   - Handle malformed JSON lines gracefully
   - Ensure partial results are captured if process crashes

4. **Platform Compatibility**
   - Ensure solution works on macOS, Linux, Windows
   - Consider differences in process/pipe handling

## Next Steps

1. Review and refine this plan
2. Decide on formatter distribution approach
3. Start with Phase 1 research tasks
4. Create proof-of-concept formatter

## Success Metrics

- [ ] Rux matches turbo_tests performance (within 10%)
- [ ] CPU usage reduced by ~50%
- [ ] Real-time progress output maintained
- [ ] No visible pausing between test runs
- [ ] All existing tests pass