"Plur" is a go based testing tool born out of my frustration involved in running and configuring test/spec suites across a multidude of Ruby and Rails projects. I rely on tests constantly, and I jump across many projects in my day to day work and for side projects. There are some things I want available for any ruby project, without add new gems or doing any setup:

1. a single command to run the whole suite
2. a single command to run the whole suite in parallel, or at least _try_ to, given parallel running breaks many suites
3. a single command to run a persistent file watcher that runs appropriate specs/tests when files change
  - basically, like Ruby's gaurd, but without adding gems, without adding any configuration, at least for the common cases
4. smart, user-friendly output (stdout) and lower level logging (stderr) that helps developers understand what is happening, and offers more detailed information if things go wrong.  Some examples:
  - `plur -n 1` (single core) by default runs all spec files in `spec` in serial in the exact same format as rspec, basically piping through results from the underly rspec call
  - `plur -n 4` (4 cores) runs all spec files in `spec` in parallel, using the same approach as turbo_tests
  - `plur -n 4 --verbose` (4 cores) runs all spec files in `spec` in parallel, using the same approach as turbo_tests. It also shows more lower level info to help debugging how plur splits the suite and how a dev can troubleshoot things like global state issues.
  - `plur watch --verbose` should show what file events are seen, and also show things triggered or _not_ triggered and why.
  - `plur watch-config` reads the tree of files that plur _will use_ along with the patterns plur will configure to handle mappings of files to tests. This should the exact same logic as `plur watch` so it is reliable and helpful. The output should be a simple JSON output that devs can understand, and Plur can output in the future to disk to allow changing. 

### Additional changes needed

Logging needs an overhual.

Currently I use Go's standard `fmt` library for stdout/stderr feedback in most places. plur watch uses a combination of `fmt` and `slog`. A good first step would be to use `slog` all throughout plur watch, as the watching side will need more verbose output for debugging and observability. 

A followup step would be to use slog throughout the rest of the codebase, at least where it makes sense -- for some things, like direct printing of "progress dots" in parallel runs, it probably makes sense to not introduce any overhead.

Additionally, our current integration suite is quite good at capturing and confirming the output between plur and rspec where we want them to be equivalent. Care should be given to maintain that with any logging changes. 




