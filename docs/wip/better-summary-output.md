# Plur TODO

## Show job source in summary output

**Priority:** Medium

Currently the startup summary only shows worker count:
```
Running 962 specs in parallel using 8 workers
```

It should also show the job name and command at INFO level (no `--verbose` required):
```
Running 962 specs in parallel using 8 workers with job.rspec ("bin/rspec")
```

Or for defaults:
```
Running 962 specs in parallel using 8 workers with defaults.job.rspec ("bundle exec rspec")
```

### Implementation

* Modify `printSummary()` in `plur/runner.go:133`
* Add `Source string` field to `job.Job` struct to track origin (e.g., "job.rspec" vs "defaults.job.rspec")
* Set source when job is resolved in config loading
* Format: `with %s (%q)` where first is source, second is `strings.Join(cmd, " ")`
