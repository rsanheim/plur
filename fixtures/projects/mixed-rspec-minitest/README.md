# Mixed RSpec/Minitest Test Fixture

This fixture project contains both RSpec and Minitest test suites to test Plur's framework detection and selection behavior.

## Structure

* `spec/example_spec.rb` - RSpec tests (3 examples)
* `test/example_test.rb` - Minitest tests (3 tests)
* `Gemfile` - Contains both rspec and minitest gems

## Purpose

Tests framework selection scenarios:

* Auto-detection: When both `spec/` and `test/` exist, Plur defaults to RSpec
* Explicit selection: Use `--use=rspec` or `--use=minitest` to choose framework
* Config file: Set `use = "minitest"` in `.plur.toml` to change default
* Override: CLI flag overrides config file setting

## Usage

```bash
# Default: runs RSpec tests
plur -C fixtures/projects/mixed-rspec-minitest

# Explicit: run Minitest tests
plur -C fixtures/projects/mixed-rspec-minitest --use=minitest

# Explicit: run RSpec tests
plur -C fixtures/projects/mixed-rspec-minitest --use=rspec
```

## Expected Output

* RSpec: "3 examples, 0 failures"
* Minitest: "3 runs, 3 assertions, 0 failures, 0 errors"
