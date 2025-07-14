# RSpec Success Simple Fixture

This is a minimal RSpec test fixture used for performance testing in the rux test suite.

## Purpose

This fixture contains a single, simple passing test that executes quickly. It's used to measure the overhead that rux adds compared to running RSpec directly.

## Structure

- `spec/simple_spec.rb` - A single test that always passes
- `spec/spec_helper.rb` - Basic RSpec configuration
- `.rspec` - RSpec options
- `Gemfile` - Minimal dependencies (just RSpec)

## Usage

This fixture is used by `spec/performance_spec.rb` in the "overhead measurement" test.