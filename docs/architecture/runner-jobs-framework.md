# Jobs and Frameworks

A job defines a command, environment, and target patterns. A framework supplies
output parsing, default arguments, and target handling. For configuration fields
and examples, see [Configuration](../configuration.md#job-configuration).

## Configuration and Selection

`internal/runtime` merges user jobs with the built-in jobs in `defaults.toml`.
Nonempty user fields override the built-in values. Custom jobs default to the
`passthrough` framework unless a framework is specified.

Configuration loading resolves each job's framework once. Job selection, file
discovery, and the runner then use the same `framework.Job`. Validation rejects
unknown frameworks, jobs without commands, and invalid watch mappings before
execution.

Plur selects a job by name if one was requested. Otherwise, it infers the framework
from the supplied paths, then falls back to autodetection in the order RSpec,
Minitest, Go. Custom jobs must be selected by name.

`framework.Job.TargetPatterns` uses the job's target pattern when set and otherwise
the resolved framework's detection patterns.

## Test Runs

`internal/fileset` expands files, directories, and globs, preserves RSpec selectors,
and applies exclusions before returning sorted, deduplicated targets.

`framework.Job.BuildRunArgs` starts with `job.cmd`, adds framework defaults, and
then adds passthrough arguments and targets:

* RSpec loads Plur's JSON formatter and applies the resolved color setting.
* Minitest loads target files with `require File.expand_path(f)` in a Ruby `-e`
  script. On Minitest 6, the script explicitly loads the Plur plugin unless
  `MT_NO_PLUGINS` is set. Passthrough arguments follow the script and `--`.
* Other frameworks append passthrough arguments and target paths to the command.

`internal/runner` groups targets across workers and applies the job environment.
See [Test Processing Flow](test-processing-flow.md) for output handling.

## Watch Runs

Watch mappings expand target templates before building commands.
`watch.JobRun.Command` appends those targets directly to `job.cmd` and applies
`job.env`. It does not add the test runner's framework arguments or output parser.
A `no_targets = true` mapping runs the command without target arguments.

See [Watch Architecture](plur-watch-architecture.md) for scheduling and lifecycle.
