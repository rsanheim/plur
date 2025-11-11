# Autodetection System Design - Phase 6

## Executive Summary

This document presents a comprehensive analysis and design for plur's autodetection system as part of Phase 6 of the Task-to-Job migration. The goal is to create a **simple, transparent, and helpful** autodetection system that provides an excellent zero-configuration experience for test runners across multiple languages.

**Key Findings**:
* Current Ruby/Go autodetection is fundamentally sound but lacks visibility
* Detection logic is scattered between watch/defaults.go and main.go
* Users have no easy way to see what was detected or why
* Foundation exists for multi-language support but needs consolidation

**Phase 6 Focus**:
* Consolidate all autodetection logic into single source of truth
* Make detection decisions visible and debuggable
* Enhance `plur doctor` to show detection results
* Prepare architecture for future language support (JS/TS, Python, Rust, Zig)

**Priority**: Testing first, builds/linting secondary

## Current State Analysis

### What Works Well

**1. Clean Separation of Concerns**
* Autodetection logic isolated in `watch/defaults.go`
* Embedded `defaults.toml` provides declarative configuration at compile-time
* Profile-based system (ruby, go) is extensible

**2. Reliable Framework Detection**
* Simple filesystem-based detection (no complex heuristics)
* `go.mod` → Go profile
* `Gemfile` + `spec/` → Ruby/RSpec profile
* `Gemfile` + `test/` → Ruby/Minitest profile

**3. Good Test Coverage**
* `watch/defaults_test.go` covers core detection logic
* Integration tests validate framework selection
* Fixture projects test real-world scenarios

### Current Problems

**1. Scattered Detection Logic**

Framework priority is duplicated:
* `watch/defaults.go:AutodetectProfile()` returns generic "ruby" profile
* `main.go:39-47` has separate logic to choose rspec vs minitest
* Not DRY - changes require updates in multiple places

**2. Poor Visibility**

Users have no way to see:
* What was autodetected and why
* What alternatives are available
* How to override detection
* What the effective configuration is

Current output:
```bash
$ plur
# No indication of what framework was detected
# Just runs tests silently
```

**3. Limited Debugging**

`plur doctor` shows:
* Watch directories (good!)
* Active configuration files (good!)

But doesn't show:
* Detected language/profile
* Why that profile was chosen
* Available jobs from autodetection
* Which job would run by default

**4. Hard to Customize**

No easy path from "it works" to "I want to customize":
* Can't see what defaults were used
* No command to export autodetected config
* `plur config:init` templates still use old Task syntax

**5. Hardcoded Framework Names**

Parser selection and command building switch on job name strings:
```go
// job/job.go
func (j *Job) CreateParser() (types.TestOutputParser, error) {
    switch j.Name {
    case "rspec":      // Hardcoded
        return rspec.NewOutputParser(), nil
    case "minitest":   // Hardcoded
        return minitest.NewOutputParser(), nil
    default:
        return passthrough.NewOutputParser(), nil
    }
}
```

This is actually **fine for built-in frameworks** but limits pure config-based extensibility.

### Current Autodetection Call Sites

**1. `main.go` - SpecCmd.Run() (lines 39-62)**
```go
// Autodetects when no explicit --use flag
defaults := watch.GetAutodetectedDefaults()
// Then does separate priority logic for rspec vs minitest
```

**2. `watch.go` - loadWatchConfiguration() (lines 52-62)**
```go
// Uses autodetection for watch mode
defaults := watch.GetAutodetectedDefaults()
// Logs detected profile in verbose mode
```

**3. `doctor.go` - checkConfiguration() (lines 179-196)**
```go
// Shows watch directories from autodetection
defaults := watch.GetAutodetectedDefaults()
```

**4. `watch/mapping_rules.go` - GenerateSuggestions()**
```go
// Uses AutodetectProfile() for file suggestions
profile := watch.AutodetectProfile()
```

## Multi-Language Detection Strategy

### Language Priority Order

Based on user requirements:
1. **Ruby** (highest priority - production ready)
2. **Go** (high priority - production ready)
3. **JavaScript/TypeScript** (medium priority - planned)
4. **Python** (medium priority - planned)
5. **Rust** (future consideration)
6. **Zig** (future consideration)

### Detection Patterns by Language

#### 1. Ruby (Current - Excellent)

**Detection Signals**:
* `Gemfile` presence (definitive Ruby project marker)
* `spec/` directory → RSpec is likely
* `test/` directory → Minitest is likely
* Both exist → Prefer RSpec (most common in modern projects)

**Frameworks Supported**:
* **RSpec**: `bundle exec rspec {{target}}`
  * Pattern: `spec/**/*_spec.rb`
  * Watch: `lib/**/*.rb` → `spec/{{match}}_spec.rb`

* **Minitest**: `bundle exec ruby -Itest {{target}}`
  * Pattern: `test/**/*_test.rb`
  * Watch: `lib/**/*.rb` → `test/{{match}}_test.rb`

* **Cucumber** (potential addition):
  * Detect: `features/` directory
  * Pattern: `features/**/*.feature`
  * Command: `bundle exec cucumber {{target}}`

**Current State**: Production ready, just needs better visibility

#### 2. Go (Current - Good)

**Detection Signals**:
* `go.mod` file (definitive Go project marker)
* `go.sum` file (additional confirmation)
* `**/*_test.go` files

**Test Execution**:
* Go tests run by package directory, not individual files
* Use `{{dir_relative}}` token for package path
* Command: `go test -v {{target}}`

**Important Patterns**:
* Exclude `vendor/` and `testdata/` directories
* Watch mappings trigger package-level test runs
* `go test ./...` runs all packages recursively

**Potential Enhancements**:
* Detect **Ginkgo**: Check for imports in test files
  * `import "github.com/onsi/ginkgo"`
  * Command: `ginkgo -v {{target}}`
* Detect **GoConvey**: Check for imports
  * `import "github.com/smartystreets/goconvey/convey"`

**Current State**: Good foundation, could add specialized framework detection

#### 3. JavaScript/TypeScript (Planned)

**Detection Signals** (in priority order):
1. `package.json` presence (definitive)
2. Parse `devDependencies` for framework markers
3. Check `scripts.test` command as hint
4. Look for framework config files
5. Fallback to file pattern scanning

**Framework Detection Priority**:

| Framework | Popularity | Detection |
|-----------|-----------|-----------|
| **Vitest** | Rising fast | `"vitest"` in devDependencies |
| **Jest** | Most popular | `"jest"` in devDependencies |
| **Playwright** | E2E standard | `"@playwright/test"` in devDependencies |
| **Cypress** | E2E alternative | `"cypress"` in devDependencies |
| **Mocha** | Legacy | `"mocha"` in devDependencies |

**Recommended Detection Algorithm**:
```
1. Parse package.json
2. Check devDependencies:
   - "vitest" → Vitest (fast, modern)
   - "jest" → Jest (most common)
   - "@playwright/test" → Playwright (e2e)
   - "cypress" → Cypress (e2e)
   - "mocha" → Mocha (legacy)
3. Fallback to scripts.test analysis
4. Fallback to file patterns
```

**File Patterns**:
* `**/*.test.{js,jsx,ts,tsx}` (most common)
* `**/*.spec.{js,jsx,ts,tsx}` (also common)
* `__tests__/**/*.{js,jsx,ts,tsx}` (Jest convention)
* `cypress/e2e/**/*.cy.{js,ts}` (Cypress)
* `tests/**/*.spec.{js,ts}` (Playwright)

**Watch Mappings**:
* `src/**/*.{js,jsx,ts,tsx}` → `src/{{match}}.test.{js,jsx,ts,tsx}`
* `src/**/*.{js,jsx,ts,tsx}` → `__tests__/{{match}}.test.{js,jsx,ts,tsx}`

**Important Considerations**:
* Multiple frameworks often coexist (Jest for unit, Playwright for e2e)
* TypeScript requires matching `.ts`/`.tsx` extensions
* Both `.test` and `.spec` patterns widely used
* Monorepo patterns need special handling

#### 4. Python (Planned)

**Detection Signals**:
1. `requirements.txt`, `setup.py`, `pyproject.toml`, or `Pipfile`
2. Check for pytest/unittest in dependencies
3. Look for `pytest.ini`, `setup.cfg`, `tox.ini` config files
4. Scan for file patterns: `test_*.py` or `*_test.py`

**Framework Detection Priority**:

| Framework | Popularity | Detection |
|-----------|-----------|-----------|
| **pytest** | Most popular | `pytest` in requirements, `pytest.ini` exists |
| **unittest** | Built-in | Default fallback (always available) |
| **nose2** | Legacy | `nose2` in requirements |

**Recommended Detection Algorithm**:
```
1. Check for pytest.ini or [tool.pytest] in pyproject.toml → pytest
2. Check for pytest in requirements.txt → pytest
3. Check for nose2 in requirements → nose2
4. Default to unittest (built-in, always works)
```

**File Patterns**:
* `test_*.py` (pytest/unittest convention)
* `*_test.py` (alternative pattern)
* `tests/**/*.py` (test directory)

**Watch Mappings**:
* `**/*.py` → `test_{{match}}.py`
* `**/*.py` → `{{match}}_test.py`
* `**/*.py` → `tests/test_{{match}}.py`

**Important Considerations**:
* Virtual environments: May need `python -m pytest` instead of `pytest`
* pytest autodiscovery: `pytest` with no args works
* unittest requires: `python -m unittest discover`
* Exclude patterns: `__pycache__/**`, `*.pyc`, test directories

#### 5. Rust (Future)

**Detection Signals**:
* `Cargo.toml` file (definitive)
* `src/` directory with `.rs` files
* `tests/` directory for integration tests

**Test Organization**:
* Unit tests: In same file as code, in `#[cfg(test)] mod tests` blocks
* Integration tests: Separate files in `tests/` directory
* Doc tests: In documentation comments

**Commands**:
* `cargo test` - Run all tests
* `cargo test --lib` - Unit tests only
* `cargo test --test name` - Specific integration test
* `cargo test test_name` - Filter by name

**Watch Patterns**:
* `src/**/*.rs` → Run tests in same file
* `tests/**/*.rs` → Run integration test

#### 6. Zig (Future)

**Detection Signals**:
* `build.zig` file (definitive)
* `zig.mod` file (package manager)
* `.zig` source files

**Test Execution**:
* Tests inline using `test` blocks
* `zig build test` - Run all tests (standard)
* `zig test file.zig` - Single file

**Watch Patterns**:
* `src/**/*.zig` → `zig build test` (build system handles all)

### Handling Multiple Languages/Frameworks

**Scenario 1: Monorepo with Multiple Languages**

Detection should:
1. Identify all present languages
2. Use precedence order for default (Go > Ruby > JS > Python)
3. Show message: "Multiple languages detected. Using Go by default. Override with --use"
4. Allow explicit selection: `plur --use=jest`

**Scenario 2: Multiple Frameworks in Same Language**

Example: Both `spec/` and `test/` exist (RSpec + Minitest)

Current behavior (good!):
```
>>> Detected both spec/ and test/ directories
>>> Using RSpec by default
>>> Tip: Use --use=minitest to run Minitest tests instead
```

**Scenario 3: Unit + E2E Tests**

Example: Jest (unit) + Playwright (e2e)

Strategy:
* Detect both frameworks
* Prioritize unit test framework for default
* Show available frameworks
* User configures in .plur.toml:
```toml
use = "jest"  # Default

[job.jest]
cmd = ["npm", "test"]

[job.playwright]
cmd = ["npx", "playwright", "test"]
```

## Design Recommendations for Phase 6

### Principle 1: Make Autodetection Obvious

**Current Experience**:
```bash
$ plur
# Tests run, but no indication what was detected
```

**Desired Experience**:
```bash
$ plur
>>> Using job 'rspec' (autodetected from spec/ directory)
>>> Found 47 test files
>>> Running with 4 workers...
```

**Implementation**:
* Add detection info to normal output (not just verbose)
* Show what was detected and why (concise)
* Clear indication this was autodetected (not explicit config)

### Principle 2: Single Source of Truth

**Problem**: Framework priority logic exists in two places
* `watch/defaults.go` - Returns generic "ruby" profile
* `main.go` - Separate logic chooses rspec vs minitest

**Solution**: Consolidate into `watch/defaults.go`

**New Functions**:
```go
// Returns the primary job name for autodetection
func GetAutodetectedJobName() (string, error)

// Returns detection details for debugging
func GetDetectionInfo() *DetectionInfo

type DetectionInfo struct {
    Profile      string   // "ruby", "go", ""
    JobName      string   // "rspec", "minitest", "go-test"
    Reason       string   // "Found Gemfile + spec/ directory"
    Alternatives []string // ["minitest"] if test/ also exists
    Available    []string // All jobs in profile
}
```

**Usage in main.go**:
```go
// Instead of duplicating priority logic
info, err := watch.GetDetectionInfo()
if err != nil {
    return err
}

logger.LogInfo("Using job", "name", info.JobName, "reason", info.Reason)
```

### Principle 3: Enhanced plur doctor

**Current Output**:
```
Plur Doctor
===========

Watch Directories:
  lib/
  app/
  spec/

Configuration Files:
  .plur.toml (not found)
  ~/.plur.toml (not found)
```

**Enhanced Output**:
```
Plur Doctor
===========

Framework Detection:
  ✓ Detected profile: ruby
  ✓ Detection reason: Found Gemfile + spec/ directory
  ✓ Primary job: rspec
  ✓ Alternative frameworks: minitest (use --use=minitest)
  ✓ Available jobs: rspec, minitest, rubocop

Configuration:
  Default job: rspec (autodetected)
  Workers: 4 (auto-detected from CPU cores)
  Override: Set 'use = "minitest"' in .plur.toml

Watch Directories:
  lib/
  app/
  spec/

Configuration Files:
  .plur.toml (not found)
  ~/.plur.toml (not found)
  Using embedded defaults: watch/defaults.toml (ruby profile)
```

### Principle 4: Easy Configuration Export

**New Command**: `plur config:show`

Shows effective configuration:
```bash
$ plur config:show
# Current effective configuration
# (merged from defaults + config files + CLI flags)

language = "ruby"
framework = "rspec"
use = "rspec"
workers = 4

[job.rspec]
cmd = ["bundle", "exec", "rspec", "{{target}}"]
target_pattern = "spec/**/*_spec.rb"

[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest", "{{target}}"]
target_pattern = "test/**/*_test.rb"

[[watch]]
name = "lib-to-spec"
source = "lib/**/*.rb"
targets = ["spec/{{match}}_spec.rb"]
jobs = "rspec"
```

**New Flag**: `plur config:init --from-autodetect`

Generates .plur.toml from current autodetected defaults:
```bash
$ plur config:init --from-autodetect
Created .plur.toml with autodetected Ruby/RSpec configuration
Customize as needed!
```

### Principle 5: Improved Debugging

**Verbose Mode** (`--verbose`):

Show full decision tree:
```
>>> Autodetection starting...
>>> Checking for go.mod... not found
>>> Checking for Cargo.toml... not found
>>> Checking for Gemfile... found!
>>> Checking for spec/ directory... found!
>>> Checking for test/ directory... found!
>>> Multiple frameworks available: rspec, minitest
>>> Using rspec (preferred for spec/ directory)
>>> Loaded ruby profile with 3 jobs: rspec, minitest, rubocop
```

**Dry Run** (`--dry-run`):

Already shows commands, enhance to show detection:
```bash
$ plur --dry-run
>>> Autodetected: rspec (from spec/ directory)
>>> Would run: bundle exec rspec spec/models/user_spec.rb
>>> Would run: bundle exec rspec spec/controllers/home_spec.rb
>>> ...
>>> Total: 47 test files with 4 workers
```

### Principle 6: Clear Error Messages

**When Nothing Detected**:
```
>>> No test framework detected

Plur supports:
  • Ruby (RSpec, Minitest) - requires Gemfile + spec/ or test/
  • Go (go test) - requires go.mod

Create a configuration file to specify manually:
  plur config:init

Or use a specific job:
  plur --use=<job-name>
```

**When Multiple Languages Detected** (future):
```
>>> Multiple languages detected:
  • Ruby (Gemfile found)
  • Go (go.mod found)

Using Ruby by default (first in precedence order)

To use Go instead:
  plur --use=go-test

Or set default in .plur.toml:
  use = "go-test"
```

## Implementation Roadmap

### Phase 6.1: Consolidate Detection Logic

**Files to Modify**:
* `watch/defaults.go` - Add consolidated detection functions
* `main.go` - Use new detection functions, remove duplicate logic
* `watch.go` - Use new detection functions
* `doctor.go` - Use new detection info struct

**New Functions in watch/defaults.go**:
```go
// DetectionInfo contains results of framework autodetection
type DetectionInfo struct {
    Profile      string   // "ruby", "go", ""
    PrimaryJob   string   // "rspec", "go-test", etc.
    Reason       string   // Human-readable explanation
    Available    []string // All job names in profile
    Alternatives []string // Other jobs user could choose
}

// GetDetectionInfo returns detailed detection results
func GetDetectionInfo() (*DetectionInfo, error)

// GetAutodetectedJobName returns primary job name
func GetAutodetectedJobName() (string, error)
```

**Tests to Update**:
* `watch/defaults_test.go` - Add tests for new functions
* `spec/integration/plur_spec/framework_selection_spec.rb` - Verify behavior

### Phase 6.2: Enhance Visibility

**Main Command Output**:
* Show detection info in normal output (not just verbose)
* Format: `>>> Using job 'rspec' (autodetected)`
* Add to logger output in main.go SpecCmd.Run()

**Verbose Mode**:
* Show full decision tree
* Why each framework was/wasn't chosen
* What alternatives exist

**Dry Run Mode**:
* Show detection summary before command preview
* Make clear what was autodetected vs configured

### Phase 6.3: Improve plur doctor

**Add to doctor.go**:
* Call GetDetectionInfo()
* Show detection results in structured format
* Show available jobs and alternatives
* Show how to override
* Indicate if using autodetection or explicit config

**Example Output Structure**:
```
Framework Detection:
  ✓ Profile: ruby
  ✓ Primary job: rspec
  ✓ Alternatives: minitest

Configuration:
  Use: rspec (autodetected)
  Workers: 4
  Override: Set 'use = "minitest"' in .plur.toml
```

### Phase 6.4: Add Configuration Commands

**Option A**: Enhance existing `config:init`
* Update templates to use Job syntax (not Task)
* Add `--from-autodetect` flag
* Add `--language` flag for templates

**Option B**: Add new `config:show`
* Display effective configuration
* Show merged result of all config layers
* Support `--format=toml` to output in TOML format

**Recommendation**: Do both
* `config:init` for creating new configs
* `config:show` for understanding current state

### Phase 6.5: Update Documentation

**Files to Update**:
* `docs/configuration.md` - Document autodetection behavior
* `docs/usage.md` - Show examples with detection messages
* `README.md` - Update examples to show detection
* `CLAUDE.md` - Update framework detection section

**New Sections Needed**:
* "How Autodetection Works"
* "Debugging Detection Issues"
* "Customizing Defaults"

### Phase 6.6: Prepare for Multi-Language

**Design defaults.toml structure**:
```toml
# Current: Flat structure
[profile.ruby.job.rspec]
# ...

# Future: Grouped by language for clarity
[defaults.ruby]
# Ruby defaults

[defaults.go]
# Go defaults

[defaults.javascript]
# JS defaults (future)
```

**Precedence for Multiple Languages** (future):
1. Explicit `--use` flag
2. `use = "job"` in config
3. Language precedence order: Go > Ruby > JavaScript > Python
4. Within language: Framework precedence (rspec > minitest)

## Success Criteria for Phase 6

**Must Have**:
* ✓ All autodetection logic consolidated in watch/defaults.go
* ✓ No duplicate framework priority logic
* ✓ `plur doctor` shows detection information
* ✓ Normal output shows what was autodetected
* ✓ Verbose mode shows detection decision tree
* ✓ `config:init` templates use Job syntax

**Should Have**:
* ✓ `plur config:show` command to see effective config
* ✓ `plur config:init --from-autodetect` flag
* ✓ Updated documentation for autodetection
* ✓ Clear error messages when detection fails

**Nice to Have**:
* Enhanced dry-run output with detection info
* More detailed detection reasoning
* Examples for all supported frameworks

**Out of Scope for Phase 6**:
* Adding new language support (JS, Python, Rust, Zig)
* Extensible parser system
* Plugin architecture
* Complex heuristics or ML-based detection

## Future Considerations

### Phase 7: JavaScript/TypeScript Support

* Add `DetectJavaScript()` function
* Parse package.json for framework detection
* Handle monorepo patterns (workspace, lerna)
* Support multiple frameworks (jest + playwright)

### Phase 8: Python Support

* Add `DetectPython()` function
* Parse requirements.txt, pyproject.toml
* Handle virtual environments
* Support pytest, unittest, nose2

### Phase 9: Rust/Zig Support

* Add language detection for Cargo.toml, build.zig
* Simple test execution patterns
* Handle inline test blocks

### Beyond: Extensibility

**Potential Features**:
* Plugin system for custom languages
* Config-based parser definitions
* Community defaults repository
* Language-specific hooks

**Not Planned**:
* AI/ML-based detection
* Network-based framework detection
* Auto-downloading test runners
* Mock modes

## Appendix: Research References

### Current Plur Implementation Files
* `plur/watch/defaults.go` - Core autodetection logic
* `plur/watch/defaults.toml` - Embedded default configurations
* `plur/main.go` - SpecCmd framework selection
* `plur/watch.go` - Watch mode configuration loading
* `plur/doctor.go` - Configuration validation
* `plur/job/job.go` - Job struct and methods

### Test Coverage
* `plur/watch/defaults_test.go` - Autodetection unit tests
* `spec/integration/plur_spec/framework_selection_spec.rb` - Framework selection integration tests
* `fixtures/projects/` - Test fixture projects

### Design Patterns Analyzed
* **parallel_tests**: Simple ENV var coordination
* **turbo_tests**: RSpec-specific, minimal config
* **plur**: Profile-based, embedded defaults, extensible

### Language Ecosystem Patterns
* **Ruby**: Gemfile + directory structure (spec/ vs test/)
* **Go**: go.mod + package-based testing
* **JavaScript**: package.json + varied frameworks
* **Python**: requirements.txt + pytest dominance
* **Rust**: Cargo.toml + inline tests
* **Zig**: build.zig + inline tests

## Conclusion

Plur's autodetection system has a solid foundation with Ruby and Go support. Phase 6 should focus on making this system **transparent, debuggable, and user-friendly** rather than adding new language support.

The key improvements are:
1. **Consolidate** detection logic into single source of truth
2. **Communicate** what was detected and why
3. **Enable** easy customization from defaults
4. **Prepare** architecture for multi-language future

By completing Phase 6, we'll have a robust autodetection system that:
* Works perfectly for Ruby and Go (primary use cases)
* Provides clear feedback to users
* Makes customization trivial
* Sets patterns for future language additions

The path from "it just works" to "I can customize it" should be obvious and friction-free.