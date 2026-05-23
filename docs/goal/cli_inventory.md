# CLI inventory

_for T1-INV_

### Commands to inventory for current plur goal

* `plur`
* `plur spec`
* `plur spec foo_spec.rb`
* `plur test`
* `plur test foo_test.rb`
* `plur spec/**/*.rb`
* `plur --use custom-job`
* `plur foo/baz/other-file.go`
* `plur foo/baz/other_test.rs`
* `plur foo_spec.rb bar_spec.rb`
* `plur -C ~/src/oss/rubocoop spec`
* `plur spec --exclude-pattern '*user*/_spec.rb'`
* `plur foo/**/*_spec.rb other/**/*_spec.rb`
* `plur --help`
* `plur help spec`
* `plur "foo/(1|2|3|)_spec.rb"` or closest supported glob syntax
* `plur watch` (persistent process that connets to TTY - so examine config/cli shape, and then run interactively via tmux to use it)