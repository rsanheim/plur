# Plan: Unify Job Resolution Across Commands

## Problem Summary

The job resolution logic is fragmented across commands with redundant code paths:

* **SpecCmd** has complex multi-path logic: DetectFramework → parent.Job → autodetectedJobs
* **WatchCmd** validates `--use` but never actually uses it when selecting a job
* **DoctorCmd** doesn't show which job would be used
* **defaults.go** has unnecessary deep copies (jobs/watches are immutable at runtime)

## Desired Flow (Same for All Commands)

```
1. If Use != "" → look up in user's config ONLY → fail if not found
2. Else → autodetect from defaults.toml
3. Else → fail with clear error
```

Kong handles CLI vs config file precedence for us. The "profile" concept is internal to autodetection - users never see or reference it.

All three commands (`spec`, `watch`, `doctor`) should use the exact same resolution logic.

## Implementation

### Step 1: Simplify defaults.go Data Access

**File:** `plur/autodetect/defaults.go`

Remove the "profile" abstraction entirely. Instead of `GetDefaultProfile("ruby")`, provide flat access to all jobs.

The embedded `defaults.toml` still has the `defaults.ruby.*` and `defaults.go.*` structure for organization, but our Go code just flattens it into a single map of jobs and list of watches.

```go
// getAllDefaultJobs returns all jobs from all profiles, flattened
func getAllDefaultJobs() map[string]job.Job {
    jobs := make(map[string]job.Job)
    for _, profile := range builtinDefaults.Defaults {
        for name, j := range profile.Jobs {
            j.Name = name
            jobs[name] = j
        }
    }
    return jobs
}

// getDefaultJob returns a specific job by name from any profile
func getDefaultJob(name string) (*job.Job, []watch.WatchMapping) {
    for _, profile := range builtinDefaults.Defaults {
        if j, exists := profile.Jobs[name]; exists {
            j.Name = name
            return &j, profile.Watches
        }
    }
    return nil, nil
}
```

No deep copies needed - Go's value semantics handle it when we assign `j.Name = name`.

### Step 2: Create Unified Job Resolution Function

**File:** `plur/autodetect/defaults.go`

Add a new function that encapsulates the entire resolution logic:

```go
// ResolveJobResult contains the resolved job and metadata
type ResolveJobResult struct {
    Job         job.Job
    Name        string
    WasInferred bool                 // true if inferred from file patterns (for hint message)
    Watches     []watch.WatchMapping // associated watch mappings from profile
}

// ResolveJob determines which job to use based on explicit selection or autodetection
// Parameters:
//   - explicitName: job name from --use flag/config (Kong handles precedence)
//   - userJobs: jobs defined in user's config file
//   - patterns: file patterns from CLI (for inference)
func ResolveJob(explicitName string, userJobs map[string]job.Job, patterns []string) (*ResolveJobResult, error) {
    // 1. If explicit name provided, look it up in user config ONLY
    if explicitName != "" {
        if j, exists := userJobs[explicitName]; exists {
            j.Name = explicitName
            return &ResolveJobResult{Job: j, Name: explicitName}, nil
        }
        // Job not found in user config - fail with helpful error
        return nil, buildJobNotFoundError(explicitName, userJobs)
    }

    // 2. If file patterns provided, infer from suffixes
    if len(patterns) > 0 {
        if jobName := inferJobFromPatterns(patterns); jobName != "" {
            // Get the job from defaults
            if j, watches := getDefaultJob(jobName); j != nil {
                return &ResolveJobResult{Job: *j, Name: jobName, WasInferred: true, Watches: watches}, nil
            }
        }
    }

    // 3. Autodetect: loop through default jobs, first one with matching files wins
    for name, j := range getAllDefaultJobs() {
        if j.TargetPattern == "" {
            continue // skip jobs without target_pattern (like rubocop, go-lint)
        }
        matches, _ := doublestar.FilepathGlob(j.TargetPattern)
        if len(matches) > 0 {
            watches := getDefaultWatchesForJob(name)
            j.Name = name
            return &ResolveJobResult{Job: j, Name: name, Watches: watches}, nil
        }
    }

    return nil, fmt.Errorf("no test files found. Create a .plur.toml with a job configuration")
}

// inferJobFromPatterns maps file suffixes to job names
func inferJobFromPatterns(patterns []string) string {
    for _, p := range patterns {
        if strings.HasSuffix(p, "_spec.rb") {
            return "rspec"
        }
        if strings.HasSuffix(p, "_test.rb") {
            return "minitest"
        }
        if strings.HasSuffix(p, "_test.go") {
            return "go-test"
        }
    }
    return ""
}

```

### Step 3: Update SpecCmd to Use Unified Resolution

**File:** `plur/main.go`

Simplify `SpecCmd.Run()`:

```go
func (r *SpecCmd) Run(parent *PlurCLI) error {
    cfg := parent.globalConfig

    // Resolve job - Kong already handles --use vs config precedence
    result, err := autodetect.ResolveJob(r.Use, parent.Job, r.Patterns)
    if err != nil {
        return err
    }

    currentJob := result.Job

    // Show hint if framework was auto-detected (not explicit)
    if r.Use == "" {
        showAutodetectHint(result)
    }

    // ... rest of the function unchanged ...
}
```

Note: We pass `r.Use` directly - Kong has already resolved CLI vs config precedence.

### Step 4: Update WatchCmd to Use Unified Resolution and Honor --use

**File:** `plur/watch.go`

Simplify `loadWatchConfiguration()` - it now just resolves job and gets watches:

```go
func loadWatchConfiguration(cli *PlurCLI, explicitJobName string) (map[string]job.Job, []watch.WatchMapping, string, error) {
    // Resolve job using unified logic
    result, err := autodetect.ResolveJob(explicitJobName, cli.Job, nil)
    if err != nil {
        return nil, nil, "", err
    }

    // Build jobs map: resolved job + any additional user-defined jobs
    jobs := map[string]job.Job{result.Name: result.Job}
    for name, j := range cli.Job {
        if name != result.Name {
            j.Name = name
            jobs[name] = j
        }
    }

    // Use user's watches if provided, else from profile
    watches := cli.WatchMappings
    if len(watches) == 0 {
        watches = result.Watches
    }

    return jobs, watches, result.Name, nil
}
```

Update `runWatchWithConfig()` to pass the selected job name through and use it:

```go
func runWatchWithConfig(globalConfig *config.GlobalConfig, watchCmd *WatchRunCmd, cli *PlurCLI) error {
    // Resolve job - Kong handles --use precedence
    jobs, watches, selectedJobName, err := loadWatchConfiguration(cli, watchCmd.Use)
    if err != nil {
        return err
    }

    // ... later in the select loop, use selectedJobName instead of hard-coded rspec/minitest:
    case "":
        j := jobs[selectedJobName]
        cmd := job.BuildJobAllCmd(j)
        runCommandArgs(cmd)
```

### Step 5: Update DoctorCmd to Show Job Info and Honor --use

**File:** `plur/main.go`

Add `--use` flag to DoctorCmd:

```go
type DoctorCmd struct {
    Use string `short:"u" help:"Show which job would be used" default:""`
}

func (d *DoctorCmd) Run(parent *PlurCLI) error {
    return runDoctorWithConfig(parent.globalConfig, parent.Job, d.Use)
}
```

**File:** `plur/doctor.go`

Update `runDoctorWithConfig()` to show job info:

```go
func runDoctorWithConfig(globalConfig *config.GlobalConfig, userJobs map[string]job.Job, explicitJobName string) error {
    // ... existing output ...

    // Job Resolution
    fmt.Println("Job Resolution:")
    result, err := autodetect.ResolveJob(explicitJobName, userJobs, nil)
    if err != nil {
        fmt.Printf("  Error: %v\n", err)
    } else {
        fmt.Printf("  Active Job:     %s\n", result.Name)
        fmt.Printf("  Command:        %v\n", result.Job.Cmd)
        fmt.Printf("  Target Pattern: %s\n", result.Job.GetTargetPattern())
    }
    // ... rest of function ...
}
```

### Step 6: Remove Dead Code and Simplify Defaults

**File:** `plur/autodetect/defaults.go`

* Delete `DetectFramework()` function (replaced by `ResolveJob`)
* Delete `GetAutodetectedDefaults()` function (no longer needed)
* Delete `AutodetectProfile()` function (no directory-based detection)
* Delete `selectJobFromProfile()` function (no directory-based detection)
* Delete `GetDefaultProfile()` function (no "profile" concept)
* Keep `inferFrameworkFromPatterns()` renamed to `inferJobFromPatterns()`
* Keep `containsGlobChars()`
* Add `getAllDefaultJobs()` - returns flat map of all jobs with target_pattern
* Add `getDefaultJob(name)` - get specific job by name
* Add `getDefaultWatchesForJob(name)` - get watches associated with a job

**File:** `plur/autodetect/defaults.toml`

* Remove `rubocop` job (no target_pattern, not useful for autodetection)
* Remove `go-lint` job (no target_pattern, not useful for autodetection)
* Keep: rspec, minitest, go-test (all have target_pattern)

## Files to Modify

| File | Changes |
|------|---------|
| `plur/autodetect/defaults.go` | Remove deep copies, remove profile/directory detection, add `ResolveJob()` with pattern-based autodetection |
| `plur/autodetect/defaults.toml` | Remove rubocop and go-lint jobs |
| `plur/main.go` | Simplify `SpecCmd.Run()`, add `--use` to `DoctorCmd` |
| `plur/watch.go` | Update `loadWatchConfiguration()`, use selected job name |
| `plur/doctor.go` | Add job resolution display |

## Testing Strategy

* Run existing integration tests: `bin/rake test`
* Manual test `plur spec` with and without `--use` flag
* Manual test `plur watch` with `--use` flag
* Manual test `plur doctor` with `--use` flag
* Verify autodetection still works in fixture projects

## Success Criteria

* [ ] All three commands use the same `ResolveJob()` function
* [ ] `plur watch --use=minitest` actually runs minitest (not rspec)
* [ ] `plur doctor --use=minitest` shows minitest as active job
* [ ] No deep copies in defaults.go
* [ ] All existing tests pass
