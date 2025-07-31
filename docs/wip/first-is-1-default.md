For cases where plur auto sets TEST_ENV_NUMBER, we want to auto set the _first_ worker to '1' -- this is to ensure that the default DB remains unused during parallel runs.  This should be controlled via a flag '--first-is-1' which will default to true.

For example, this means in the default case, if someone runs `plur spec -n4`, workers will run:

worker 0 -> TEST_ENV_NUMBER=1 rspec [files]
worker 1 -> TEST_ENV_NUMBER=2 rspec [files]
worker 2 -> TEST_ENV_NUMBER=3 rspec [files]
worker 3 -> TEST_ENV_NUMBER=4 rspec [files]

If someone runs `plur spec -n4 --first-is-1=false`, it should work as it does right now, where the first worker gets '', and the rest get incremented by 1.

worker 0 -> TEST_ENV_NUMBER='' rspec [files]
worker 1 -> TEST_ENV_NUMBER=2 rspec [files]
worker 2 -> TEST_ENV_NUMBER=3 rspec [files]
worker 3 -> TEST_ENV_NUMBER=4 rspec [files]

This means plur will default to what is the sane, predictable behavior of using explicit numbered DBs for parallel runs.

Note that if someone is running in 'serial' mode, i.e. --n=1, then we should not set the TEST_ENV_NUMBER at all.

### Implementation Plan

#### Summary
Add a `--first-is-1` flag that controls how TEST_ENV_NUMBER is assigned to workers. When enabled (default), the first worker gets "1" instead of an empty string, ensuring predictable numbered database usage in parallel test runs.

#### Key Changes Required:

1. **Add FirstIs1 flag to PlurCLI struct** (plur/main.go:~179)
   - Add `FirstIs1 bool` field with default true
   - Add to help text explaining the behavior

2. **Update GlobalConfig struct** (plur/config.go:~2)
   - Add `FirstIs1 bool` field to pass configuration through

3. **Update globalConfig initialization** (plur/main.go:~211)
   - Add `FirstIs1: r.FirstIs1` to the GlobalConfig initialization

4. **Add IsSerial method to GlobalConfig** (plur/config.go)
   - Add method `func (c *GlobalConfig) IsSerial() bool { return c.WorkerCount == 1 }`
   - This encapsulates the serial mode check

5. **Modify GetTestEnvNumber function** (plur/runner.go:~75)
   - Change signature to accept globalConfig instead of individual parameters
   - Use globalConfig.IsSerial() to check for serial mode
   - Use globalConfig.FirstIs1 to determine numbering behavior
   - Cleaner implementation without multiple parameters

6. **Update RunDatabaseTask signature** (plur/database.go:~13)
   - Change from `func RunDatabaseTask(task string, workerCount int, dryRun bool) error`
   - To: `func RunDatabaseTask(task string, config *GlobalConfig) error`
   - Use config.WorkerCount and config.DryRun internally
   - Update all callers in db:setup, db:create, db:migrate, db:test:prepare commands

7. **Update all GetTestEnvNumber calls**:
   - RunRSpecFiles (plur/runner.go:~202)
   - RunMinitestFiles (plur/runner.go:~434)
   - RunDatabaseTask (plur/database.go:~17 and ~41)
   - Simply pass globalConfig instead of multiple parameters

8. **Update tests**:
   - Update spec/parallel_execution_spec.rb to test both modes
   - Update spec/database_tasks_spec.rb to expect new numbering
   - Add new test cases to verify:
     - Default behavior (first-is-1=true)
     - Legacy behavior (first-is-1=false)
     - Serial mode behavior (n=1, no TEST_ENV_NUMBER)

9. **Update documentation**:
   - Add flag documentation to README
   - Update any examples showing TEST_ENV_NUMBER values

#### Implementation Details:

The key logic changes will be:

1. Add IsSerial method to GlobalConfig:
```go
// IsSerial returns true if running in serial mode (single worker)
func (c *GlobalConfig) IsSerial() bool {
    return c.WorkerCount == 1
}
```

2. Update GetTestEnvNumber to use GlobalConfig:
```go
func GetTestEnvNumber(workerIndex int, config *GlobalConfig) string {
    // Serial mode: don't set TEST_ENV_NUMBER
    if config.IsSerial() {
        return ""
    }
    
    // New default behavior: all workers get explicit numbers
    if config.FirstIs1 {
        return fmt.Sprintf("%d", workerIndex+1)
    }
    
    // Legacy behavior: first worker gets "", others get 2,3,4...
    if workerIndex == 0 {
        return ""
    }
    return fmt.Sprintf("%d", workerIndex+1)
}
```

This ensures backward compatibility while defaulting to the more predictable behavior of explicit database numbering.

### Checklist
