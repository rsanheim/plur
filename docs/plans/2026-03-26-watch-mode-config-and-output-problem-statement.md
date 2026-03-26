# Watch Mode Config And Output Problem Statement

## Reset

This is not an implementation plan.

The previous draft over-specified architecture, test seams, and commit structure
before the bug had been narrowed down enough to justify that level of planning.
Reset here and keep this document focused on the problem only.

## Problem

There are two separate watch-mode issues on this branch:

- config resolution bug: user-defined `[[watch]]` mappings currently replace the
  built-in watch mappings instead of augmenting them
- watch terminal/output bugs: prompt rendering, debug log interleaving, and
  duplicate failure logging

These should be handled in that order. Fix the config bug first, then reassess
the output behavior with the smallest possible change.

## Expected Behavior

### Watch config semantics

User-defined `[[watch]]` mappings should be additive to the built-in watch
mappings for the resolved job.

Concrete example:

- resolved job: `rspec`
- user config adds:

```toml
[[watch]]
name = "custom-config-watch"
source = "config/**/*.yml"
targets = ["spec/config_spec.rb"]
jobs = ["rspec"]
```

Expected result:

- the custom watch is available
- built-in watches like `lib-to-spec`, `app-to-spec`, and `spec-files` still exist
- `plur watch find lib/calculator.rb` still resolves to `spec/calculator_spec.rb`

### Watch output semantics

Once config resolution is fixed, watch output should behave like this:

- the prompt and echoed command should read like a shell prompt, on one line
- debug/info logs should not be written on top of the prompt
- a failing test run should not emit duplicate watch-level warnings for the same non-zero exit
- if watch runs are serialized, prompt redraws should happen only after the run is complete

## Current Failing Regressions

These are intentionally failing right now and should pass once the config merge
behavior is implemented.

### Ruby integration regression

```bash
mise exec -- bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "keeps default watches when user config adds a custom watch mapping"
```

Current failure:

- `plur watch find lib/calculator.rb` exits with code `2`
- output shows `msg="found rules" count=0`
- expected built-in `lib-to-spec` mapping is missing

### Go regression

```bash
mise exec -- go test . -run TestLoadWatchConfigurationMergesUserAndDefaultWatches
```

Current failure:

- `loadWatchConfiguration` returns only the custom user mapping
- built-in mappings like `lib-to-spec` and `spec-files` are missing

## Output Acceptance Cases

The following transcripts describe the current watch output bugs. Keep them as
acceptance criteria; do not treat them as a prescribed implementation.

- S1: the `rspec ...` line should be on the same line as the `plur` prompt
- S2: in debug mode, DEBUG output interleaves on the prompt, and the echoed
  command is rendered as `[plur] ...` instead of as prompt input
- S3: on a failing spec run, watch mode emits two WARN messages after the RSpec
  output, which is redundant

### S1

```text
[plur] >
[plur] rspec spec/lib/dx/models/package_info_formatter_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 43258

Dx::Models::PackageInfoFormatter
  #format
    with a JavaScript package
      formats all fields with proper alignment
    with a gem package
      formats all fields with proper alignment
      formats each field on its own line
    with minimal package (missing optional fields)
      only shows fields that are present
      has 4 lines (name, version, type, location)

Finished in 0.01705 seconds (files took 0.12852 seconds to load)
5 examples, 0 failures

Randomized with seed 43258
```

### S2

```text
[plur] > 14:06:41 - DEBUG - watch path="spec/lib/dx/models/package_info_formatter_spec.rb" fullPath="/Users/rsanheim/work/gems/dox-dx-ruby/spec/lib/dx/models/package_info_formatter_spec.rb" event="modify" type="file"
14:06:41 - DEBUG - renderTargets result normalizedPath="spec/lib/dx/models/package_info_formatter_spec.rb" watch="spec/**/*.rb" targets=[spec/lib/dx/models/package_info_formatter_spec.rb]
14:06:41 - INFO  - Executing job job="rspec" targets="[spec/lib/dx/models/package_info_formatter_spec.rb]"

[plur] rspec spec/lib/dx/models/package_info_formatter_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 48623

Dx::Models::PackageInfoFormatter
  #format
    with a gem package
      formats all fields with proper alignment
      formats each field on its own line
    with minimal package (missing optional fields)
      has 4 lines (name, version, type, location)
      only shows fields that are present
    with a JavaScript package
      formats all fields with proper alignment

Finished in 0.01833 seconds (files took 0.13204 seconds to load)
5 examples, 0 failures

Randomized with seed 48623


[plur] >
```

### S3

```text
[plur] >
[plur] rspec spec/lib/dx/environment_spec.rb
Run options: exclude {fuzz: true}

Randomized with seed 24155

Dx::Environment
  empty and nil are invalid
  works with pre / pre-prod

Failures:

  1) Dx::Environment Environment::Null returns false for all env checks
     Failure/Error: fail
     RuntimeError:
     # ./spec/lib/dx/environment_spec.rb:19:in 'block (3 levels) in <top (required)>'
     # ./spec/spec_helper.rb:63:in 'block (2 levels) in <top (required)>'

Finished in 0.02127 seconds (files took 0.14032 seconds to load)
19 examples, 1 failure

Failed examples:

rspec ./spec/lib/dx/environment_spec.rb:17 # Dx::Environment Environment::Null returns false for all env checks

Randomized with seed 24155

14:10:58 - WARN  - Job execution failed job="rspec" error=exit status 1
14:10:58 - WARN  - Job execution error job="rspec" error=exit status 1
```

## Constraints For The Next Plan

- Keep the next plan proportional to the issue.
- Start with the smallest fix in existing files rather than inventing new abstractions.
- Do not prescribe new output infrastructure until the real write paths are understood.
- For the duplicate-warning bug, use tests that hit the real failing-job path rather than a mocked path that skips one of the warning sites.
