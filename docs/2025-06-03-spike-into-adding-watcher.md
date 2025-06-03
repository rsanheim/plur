# Spike: add e-dant/watcher to rux

Okay, the crown jewel of Rux: I would like to emulate what [guard](https://github.com/guard/guard) does for ruby/rails projects, but I want it be a single command to run in _any_ ruby project. Assuming someone installs rux and runs the following in any ruby project:

```
rux watch
```

Rux will:

* Start a simple TUI-like interface that shows the current state of the watcher, and the specs that are running - we can keep this super simple for now.
* Start a background watcher process that spwans e-dant/watcher  and sends those FSEvents to us in Rux land (over a pipe / channel?), and then Rux decides what to do with them.
* Prints immediate feedback from test process(es) to the terminal, using the same code we have built up in Rux (so it would be calling `rspec` under the hood, the same as if we ran `rux` directly).
* Use the default "ruby library/gem" file mapping for files to watch and what to run when they change
  * changes to `lib/**/*.rb` run the matching spec from `spec/**/*_spec.rb` 
  * changes to `spec/**/*_spec.rb` run the spec
  * changes to `spec/spec_helper.rb` run all specs
  * ignores .gitignore'd files, ignores things like node_modules, tmp, etc.
  * support dry-run mode, so we can see what files _would_ run

### Constraints
* Run specs in serial to start (KISS)
* Reuse the Rux code as much as possible - we have done good work in managing calling RSpec, handling formatting, etc, and we should build on that
* Use the `watcher` binary directly for our watcher - to start we will just hard lock to the aarch64-darwin binary, but we can make it more flexible later
* See example output of me using this binary directly in [2025-06-03-watcher-output-darwin.log](./2025-06-03-watcher-output-darwin.log) - so it appears the binary from the release artifact works 'out of the box' on Mac

### Future
* auto-discovery of files to watch, and generate a baseline config based on that
* rails support
* look for a Guardfile, and if its defined, we would use it as a template for creating a .rux-config file with the same mappings
