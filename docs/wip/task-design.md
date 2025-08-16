# Plur "Task" design

A `Task` is how we tell Plur how to run tests, linters, or other jobs in your project.
Tasks are used both for `plur test` and `plur watch`

Tasks will replace our mish-mash of different ways we configure rspec vs minitest for 
regular runs and watch. It will also give us a structure for other commands going forward. 

Currently we have a mix of `TestFramework` and `CommandBuilder` and other bits of 
logic spread around various places. `Task` will give us one place to store the common set of 
configuration needed to run things.

Plur will come out of the box with tasks for 'rspec' and 'minitest' (which we have support for now).
Plur should also support user defined tasks defined via our TOML config file(s).

### Task Structure

| Field           | Type | Description | Required | Default | Examples |
|-----------------|-----|-------------|----------|----------|----------|
| description     | string   | description of the recipe | No | "" | "Run RSpec specs" |
| run             | string | command to run | Yes | "" | "bundle exec rspec" |
| source_dirs     | string[] | directories used as input | No | `["."]` | `["spec", "lib"]` |
| mappings        | MappingRule[] | mappings for the task | No | `[]` | see [Mapping examples](#mapping-examples) |
| ignore_patterns | string[] | patterns to ignore (only applies to watch currently) | No | `[".git"]` | |

### Proposed responsibilities for Task

Some of the responsibilities Task will have.

```go
// NOTE: the taskOverride is an _optional_ task from CLI/config to override `rspec` or `minitest` options
// This allows someone to use a specific binstub for example, like `bin/rspec`
func (t *Task) BuildCommand(files []string, globalConfig *GlobalConfig, taskOverride Task) *exec.Cmd {
	// build the command to run or print
}

// Use the mappings to map source files to target file(s)
// This should return _all_ files that match the pattern
func (t *Task) MapFilesToTarget(sourceFile[] string) []string {
```

### Specification

* Plur will handle splitting the `run` command String into a command and arguments.
* Plur task code MUST live in `plur/internal/task/`, as this will be a developing internal API.
* Plur will default the `source_dirs` to the current working directory if not specified: this means simple
watches for things like linting, running go tests, etc, should just work.
* Our current 'auto-detect' logic can instead use the `source_dirs` to determine the test framework - most projects will either have a spec or test dir, and they always should have a `lib` dir as well. For cases where there is both a spec and test dir, we will default to rspec for `plur spec` and minitest for `plur test`.
* For mapping patterns, the target pattern MUST use {{name}} syntax for things Plur will replace from the changed path: we will support `{{path}}`, `{{name}}`, and `{{file}}` tokens.  Note that single brackets are valid glob patterns.
* Our ignore patterns will have to be applied _only_ for cases where the watcher is watching the root (for now)...
and we will have to carefully inject that into our current logic to discard paths that watcher returns. Our 
watcher libary does not have its own logic for ignore patterns.

### Examples and Details

The simplest possible task just requires a `run` command - for example, the following 
should be a valid, simple task to run `script/test`:

```toml
[task.script-test]
run: "script/test"
# `plur test` runs 'script/test`
# `plur test [file]` runs 'script/test [file]'
```

Our existing rspec task would translate to this task:

```toml
[task.rspec]
run: "bundle exec rspec"
source_dirs: ["spec", "lib"]
mappings:
  - pattern: "lib/**/*.rb"
    target: "spec/{path}/{name}_spec.rb"
  - pattern: "app/**/*.rb"
    target: "spec/{path}/{name}_spec.rb"
  - pattern: "spec/**/*_spec.rb"
    target: "{{file}}"
```

And our existing minitest task would be:

```toml
[task.minitest]
run: "" # NOTE: due to how difficult it is to run specific minitest tests, we will just handle the 'run' command here via our `minitest.BuildCommand`
source_dirs: ["test", "lib"]
mappings:
  - pattern: "lib/**/*.rb"
    target: "test/{path}/{name}_test.rb"
  - pattern: "app/**/*.rb"
    target: "test/{path}/{name}_test.rb"
  - pattern: "test/**/*_test.rb"
    target: "{{file}}"
```

#### Mapping TOML examples:

NOTE: The following two approaches are equivalent, but AFAIK they CANNOT be mixed for the same task mapping in TOML:

Consise approach (inline mapping rules):

```
[task.my-rspec]
run = "bin/rspec"
mappings = [
  { pattern = "lib/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "app/**/*.rb", target = "spec/{{path}}/{{name}}_spec.rb" },
  { pattern = "spec/**/*_spec.rb", target = "{{file}}" }, # runs the changed spec file
]
```

More explicit approach (separate mapping rules):

```
[task.my-rspec]
run = "bin/rspec"
source_dirs = ["spec", "lib"]

[[mappings]]
pattern = "lib/**/*.rb"
target = "spec/{{path}}/{{name}}_spec.rb"

[[mappings]]
pattern = "app/**/*.rb"
target = "spec/{{path}}/{{name}}_spec.rb"

[[mappings]]
pattern = "spec/**/*_spec.rb"
target = "{{file}}"
```


### Current places in PLUR that will be removed and/or updated with this new Task design:

* [ ] CommandBuilder should go away completely, as a task will know how to build the command to run.
* [ ] getTestFileSuffix and getDefaultPattern should be removed.
```
// getTestFileSuffix returns the test file suffix for the framework
func getTestFileSuffix(framework TestFramework) string {
```

```
// getDefaultPattern returns the default glob pattern for the framework
func getDefaultPattern(framework TestFramework) string {
```

* [ ] Lots of watch / mapping code scattered around, one in particular is in `watch_find.go`:

```
// DetectTestFramework attempts to detect the test framework based on directory structure
func DetectTestFramework() TestFramework {
```

* [ ] All of the `SpecCmd` code in main.go should be removed when we are done.
```
type SpecCmd struct {
```

Much of the special case code in glob.go will go away, as this will be 
driven by Task config and standard doublestar logic.