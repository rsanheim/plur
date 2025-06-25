# Minitest Refactoring Summary

*Date: 2025-06-24*

## What We've Accomplished

1. **Successfully refactored the architecture** - We've created a clean event-based system with:
   - Parser factory that returns the right parser based on framework
   - Shared streaming logic that both RSpec and Minitest use
   - TestCollector that accumulates events into results
   - Consistent naming (OutputParser for both frameworks)

2. **The plumbing works** - Tests are running, dots are appearing, the framework detection works.

## Current Challenges

### Challenge 1: Output Format Mismatch
The integration tests expect minitest-style output ("12 runs, 5 assertions...") but are getting RSpec-style output ("12 examples, 0 failures"). This happens because:
- The `PrintResults` function always formats output RSpec-style
- It doesn't know which framework was used
- For minitest, we should preserve and display the raw output

### Challenge 2: Failures Not Being Detected
When running minitest-failures, we see:
- Progress indicators: 24 green dots (should be mix of dots and F's)
- Summary: "24 examples, 0 failures" (should show failures)

The root cause seems to be that the minitest parser isn't properly handling failures. When I look at actual minitest output:
```
...FFE
```
The progress indicators come on one line, and our parser correctly iterates through them. But when it sees 'F' or 'E', it just sets `p.inFailure = true` but doesn't create a TestFailed notification immediately.

### Challenge 3: Two-Phase Parsing Problem
Minitest output has two distinct phases:
1. Progress indicators (`.`, `F`, `E`, `S`)
2. Detailed failure information that comes later

Our parser tries to match the failure details to the earlier progress indicators, but this matching isn't working correctly.

## Reflections on the Abstraction

### What's Working Well
1. **The TestCollector abstraction is solid** - It successfully accumulates notifications and builds results
2. **The parser interface is clean** - Both parsers implement the same interface
3. **The shared streaming logic works** - No duplication between frameworks

### What's Challenging
1. **Output format assumptions** - The system assumes we want to reformat output into a consistent style, but for minitest we want to preserve the original
2. **Framework context loss** - By the time we get to `PrintResults`, we don't know which framework generated the results
3. **Different parsing models** - RSpec gives us structured JSON; minitest gives us unstructured text that requires stateful parsing

### The Real Issue
The TestEvent/TestCollector abstraction was designed around RSpec's model where:
- Each test event is self-contained
- We get structured data (JSON)
- We can reformat output consistently

But minitest is different:
- Progress indicators are separate from failure details  
- We need to preserve raw output for the authentic minitest experience
- The parsing is more stateful and context-dependent

## Potential Solutions

1. **Quick fix**: Pass framework type through to PrintResults and display raw output for minitest
2. **Better fix**: Have the minitest parser create a FormattedSummaryNotification with the preserved output
3. **Deeper fix**: Rethink how we handle output - maybe TestResult should include a framework field

The abstraction isn't wrong, but it's biased toward RSpec's structured approach. We need to make it more flexible for text-based test frameworks.

## Key Insights

- The event-based architecture works, but we need to respect each framework's native output format
- Minitest's text-based output requires more stateful parsing than RSpec's JSON
- The system should preserve original output for frameworks that don't benefit from reformatting
- Framework context needs to flow through the entire pipeline, not just the parsing phase