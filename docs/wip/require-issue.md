I'm seeing specs fail when running with plur that pass through a normal `bundle exec rspec` command.

Here is one specific test case where I can reproduce the issue - note that I've added some debug output to the spec in question to help diagnose the issue.  The root cause is that `stringio` is not required when run via plur, but _is_ required successfully when run via `bundle exec rspec`.

### Environment info
```
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (1.044s) > pwd
/Users/rsanheim/src/oss/plur/references/tty-command
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (38ms) > which ruby
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin/ruby
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (39ms) > which bundle
bundle         bundle-audit   bundler        bundler-audit
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (39ms) > which bundler
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin/bundler
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (42ms) > ruby -v
ruby 3.4.7 (2025-10-08 revision 7a5688e2a2) +PRISM [arm64-darwin24]
[~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (47ms) > bundler -v
4.0.0.beta1
```

### Runs correctly with `bundle exec rspec`

```
~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (278ms) > bundle exec rspec spec/unit/printers/pretty_spec.rb
Bundler
4.0.0.beta1
  Platforms
ruby, arm64-darwin-24
Ruby
3.4.7p58 (2025-10-08 revision 7a5688e2a27668e48f8d6ff4af5b2208b98a2f5e) [arm64-darwin-24]
  Full Path
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin/ruby
  Config Dir
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/etc
RubyGems
3.7.2
  Gem Home
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0
  Gem Path
/Users/rsanheim/.gem/ruby/3.4.0:/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0
  User Home
/Users/rsanheim
  User Path
/Users/rsanheim/.gem/ruby/3.4.0
  Bin Dir
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin
Tools
  Git
2.51.2
  RVM
not installed
  rbenv
rbenv 1.3.2
  chruby
not installed
/Users/rsanheim/src/oss/plur/references/tty-command/spec
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/bundler-4.0.0.beta1/lib
/Users/rsanheim/src/oss/plur/references/tty-command/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-3.13.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-mocks-3.13.7/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-expectations-3.13.5/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-core-3.13.6/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-support-3.13.6/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/pastel-0.8.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/tty-color-0.6.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/memory_profiler-0.9.14/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/diff-lcs-1.6.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/coveralls_reborn-0.22.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/thor-1.4.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/term-ansicolor-1.11.3/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/tins-1.47.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/sync-0.5.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/mize-0.6.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov-0.21.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov_json_formatter-0.1.4/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov-html-0.13.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/docile-1.4.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/bigdecimal-3.3.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/extensions/arm64-darwin-24/3.4.0/bigdecimal-3.3.1
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rake-13.3.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby/3.4.0/arm64-darwin24
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby/3.4.0/arm64-darwin24
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/3.4.0/arm64-darwin24
/Users/rsanheim/src/oss/plur/references/tty-command

Randomized with seed 51829

TTY::Command::Printers::Pretty
  prints successful command exit in color
  prints command exit without exit status in color
/Users/rsanheim/src/oss/plur/references/tty-command/lib/tty/command/printers/pretty.rb:55: warning: literal string will be frozen in the future (run with --debug-frozen-string-literal for more information)
  prints output on error when only_output_on_error is true
/Users/rsanheim/src/oss/plur/references/tty-command/lib/tty/command/printers/pretty.rb:55: warning: literal string will be frozen in the future (run with --debug-frozen-string-literal for more information)
  doesn't print output on success when only_output_on_error is true
  prints command start in color
  prints command start without uuid
  prints command stderr data
  prints command stdout data
/Users/rsanheim/src/oss/plur/references/tty-command/lib/tty/command/printers/pretty.rb:55: warning: literal string will be frozen in the future (run with --debug-frozen-string-literal for more information)
  prints output on error & raises ExitError when only_output_on_error is true
  prints failure command exit in color
  prints command start without color

Top 2 slowest examples (0.50039 seconds, 66.0% of total time):
  TTY::Command::Printers::Pretty prints output on error when only_output_on_error is true
    0.25039 seconds ./spec/unit/printers/pretty_spec.rb:148
  TTY::Command::Printers::Pretty prints output on error & raises ExitError when only_output_on_error is true
    0.25 seconds ./spec/unit/printers/pretty_spec.rb:124

Finished in 0.75842 seconds (files took 0.08729 seconds to load)
11 examples, 0 failures

Randomized with seed 51829
```

### Fails with `plur`

```
~/src/oss/plur/references/tty-command (perf-spike)↑⚡] (45ms) > plur -v spec/unit/printers/pretty_spec.rb
00:50:58 - INFO  - found 1 test files testFiles=[spec/unit/printers/pretty_spec.rb]
plur version version=0.13.0-dev-4c3f7fa0
Running 1 spec in parallel using 1 workers
00:50:58 - INFO  - running cmd="bundle exec rspec --require spec_helper -r /Users/rsanheim/.plur/formatter/json_rows_formatter.rb --format Plur::JsonRowsFormatter --force-color --tty spec/unit/printers/pretty_spec.rb" worker=0
00:50:58 - INFO  - actual cmd cmd="/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin/bundle exec rspec --require spec_helper -r /Users/rsanheim/.plur/formatter/json_rows_formatter.rb --format Plur::JsonRowsFormatter --force-color --tty spec/unit/printers/pretty_spec.rb"
Bundler
4.0.0.beta1
  Platforms
ruby, arm64-darwin-24
Ruby
3.4.7p58 (2025-10-08 revision 7a5688e2a27668e48f8d6ff4af5b2208b98a2f5e) [arm64-darwin-24]
  Full Path
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin/ruby
  Config Dir
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/etc
RubyGems
3.7.2
  Gem Home
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0
  Gem Path
/Users/rsanheim/.gem/ruby/3.4.0:/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0
  User Home
/Users/rsanheim
  User Path
/Users/rsanheim/.gem/ruby/3.4.0
  Bin Dir
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/bin
Tools
  Git
2.51.2
  RVM
not installed
  rbenv
rbenv 1.3.2
  chruby
not installed
/Users/rsanheim/src/oss/plur/references/tty-command/spec
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/bundler-4.0.0.beta1/lib
/Users/rsanheim/src/oss/plur/references/tty-command/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-3.13.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-mocks-3.13.7/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-expectations-3.13.5/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-core-3.13.6/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rspec-support-3.13.6/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/pastel-0.8.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/tty-color-0.6.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/memory_profiler-0.9.14/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/diff-lcs-1.6.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/coveralls_reborn-0.22.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/thor-1.4.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/term-ansicolor-1.11.3/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/tins-1.47.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/sync-0.5.0/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/mize-0.6.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov-0.21.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov_json_formatter-0.1.4/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/simplecov-html-0.13.2/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/docile-1.4.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/bigdecimal-3.3.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/extensions/arm64-darwin-24/3.4.0/bigdecimal-3.3.1
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/gems/3.4.0/gems/rake-13.3.1/lib
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby/3.4.0/arm64-darwin24
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/site_ruby
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby/3.4.0/arm64-darwin24
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/vendor_ruby
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/3.4.0
/Users/rsanheim/.local/share/mise/installs/ruby/3.4.7/lib/ruby/3.4.0/arm64-darwin24
/Users/rsanheim/src/oss/plur/references/tty-command
FFFFFFFFFFF
Runtime data saved to: /Users/rsanheim/.plur/runtime/6aa6fc34.json

Failures:

  1) TTY::Command::Printers::Pretty prints command stdout data
     Failure/Error: let(:output) { StringIO.new }

     NameError:
       uninitialized constant StringIO
     # ./spec/unit/printers/pretty_spec.rb:8:in 'block (2 levels) in <top (required)>'
     # ./spec/unit/printers/pretty_spec.rb:47:in 'block (2 levels) in <top (required)>'

  2) TTY::Command::Printers::Pretty prints failure command exit in color
     Failure/Error: let(:output) { StringIO.new }

     NameError:
       uninitialized constant StringIO
     # ./spec/unit/printers/pretty_spec.rb:8:in 'block (2 levels) in <top (required)>'
     # ./spec/unit/printers/pretty_spec.rb:81:in 'block (2 levels) in <top (required)>'

[snipped repeated failures]

Finished in 0.01292 seconds (files took 0.08811 seconds to load)
11 examples, 11 failures

Failed examples:

rspec ./spec/unit/printers/pretty_spec.rb:45 # TTY::Command::Printers::Pretty prints command stdout data
rspec ./spec/unit/printers/pretty_spec.rb:79 # TTY::Command::Printers::Pretty prints failure command exit in color
rspec ./spec/unit/printers/pretty_spec.rb:34 # TTY::Command::Printers::Pretty prints command start without uuid
rspec ./spec/unit/printers/pretty_spec.rb:101 # TTY::Command::Printers::Pretty doesn't print output on success when only_output_on_error is true
rspec ./spec/unit/printers/pretty_spec.rb:124 # TTY::Command::Printers::Pretty prints output on error & raises ExitError when only_output_on_error is true
rspec ./spec/unit/printers/pretty_spec.rb:23 # TTY::Command::Printers::Pretty prints command start without color
rspec ./spec/unit/printers/pretty_spec.rb:90 # TTY::Command::Printers::Pretty prints command exit without exit status in color
rspec ./spec/unit/printers/pretty_spec.rb:11 # TTY::Command::Printers::Pretty prints command start in color
rspec ./spec/unit/printers/pretty_spec.rb:56 # TTY::Command::Printers::Pretty prints command stderr data
rspec ./spec/unit/printers/pretty_spec.rb:68 # TTY::Command::Printers::Pretty prints successful command exit in color
rspec ./spec/unit/printers/pretty_spec.rb:148 # TTY::Command::Printers::Pretty prints output on error when only_output_on_error is true
```

### Help me find the bug

The bug here is NOT in the tty-command suite itself, as clearly it works via normal bundle exec rspec.
This is a __plur__ bug or some combination of plur my environment.

I suspect this is:

1) an environment issue related to how bundler and my mise ruby are interacting when run via Go's `exec.Command` vs a normal shell command  -> so this says to me that _maybe_ plur is losing the mise setup, or the bundler env setup....though I have looked thru the debug output and compared between plur and regular rspec, and cannot see it.

2) Some weird quirk involving how we construct the `bundle exec rspec` command when plur runs it, as we do add our own formatter via `-r`, and then configure it...but I'm not sure why that would cause this issue. 

other things I've ruled out:
* spec_helper gets loaded by plur (see my deubgging I added to the tty-command spec_Helper)
* missing require that somehow works in tty-command? I do see one require for 'stringio' in tty-command, havent traced thru how it happens.

### Add findings below!

_add findings here, along with steps or investigation notes that can help prevent this sort of thing in the future_