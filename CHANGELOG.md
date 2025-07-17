# plur CHANGELOG

## v0.9.0 - 2025-07-17

* PLUR - the big rename [#51](https://github.com/rsanheim/rux-meta/pull/51)
  - Updated all references in code, documentation, and scripts
  - Binary is now `plur` instead of `rux`
  - Configuration files are now `.plur.toml` instead of `.rux.toml`
  - Home directory is now `~/.plur/` instead of `~/.rux/`
* Update backspin v060 [#68](https://github.com/rsanheim/rux-meta/pull/68)

## v0.8.2 - 2025-07-11

* Removing tracing [#67](https://github.com/rsanheim/plur-meta/pull/67)
* Clean up WIP docs [#66](https://github.com/rsanheim/plur-meta/pull/66)

## v0.8.1 - 2025-07-11

* release process needs work 

## v0.8.0 - 2025-07-11

* Minitest clean up [#65](https://github.com/rsanheim/plur-meta/pull/65)
* Minitest support - simplify reporting; support minitest-reporters [#64](https://github.com/rsanheim/plur-meta/pull/64)
* new line after results [#63](https://github.com/rsanheim/plur-meta/pull/63)
* Refactoring test handling 2 [#59](https://github.com/rsanheim/plur-meta/pull/59)
* Minitest 3 [#62](https://github.com/rsanheim/plur-meta/pull/62)
* Bundler overhead findings [#61](https://github.com/rsanheim/plur-meta/pull/61)
* Backport mkdocs improvements [#60](https://github.com/rsanheim/plur-meta/pull/60)

## v0.7.5 - 2025-06-22

* Use installed plur for integration specs; update specs [#58](https://github.com/rsanheim/plur-meta/pull/58)
* Simple config file [#56](https://github.com/rsanheim/plur-meta/pull/56)
* clean up benchmark more [#55](https://github.com/rsanheim/plur-meta/pull/55)
* Moving more stuff to plur [#54](https://github.com/rsanheim/plur-meta/pull/54)

## v0.7.2 - 2025-06-21

* Benchmark cleanup [#53](https://github.com/rsanheim/plur-meta/pull/53)
* Clean up [#52](https://github.com/rsanheim/plur-meta/pull/52)
* Docker [#48](https://github.com/rsanheim/plur-meta/pull/48)
* Cleaning up rakefile and build [#49](https://github.com/rsanheim/plur-meta/pull/49)

## v0.7.1 - 2025-06-17

* Fix file load time; clean up StreamingMessage [#47](https://github.com/rsanheim/plur-meta/pull/47)

## v0.7.0 - 2025-06-17

* Replace urfave/cli with Kong [#44](https://github.com/rsanheim/plur-meta/pull/44)
* Convert all tests to use testify assertions [#45](https://github.com/rsanheim/plur-meta/pull/45)

## v0.6.10 - 2025-06-15

* Consolidating init [#43](https://github.com/rsanheim/plur-meta/pull/43)

## v0.6.9 - 2025-06-12
Lotsa internal cleanup
* Extract execution logic and add Kong CLI experiment [#31](https://github.com/rsanheim/plur-meta/pull/31)
* Add the integration test to CI (oops) [#41](https://github.com/rsanheim/plur-meta/pull/41)
* get caching working for integration in CI [#40](https://github.com/rsanheim/plur-meta/pull/40)
* Planning for auto config [#35](https://github.com/rsanheim/plur-meta/pull/35)
* sort and clean up. [#39](https://github.com/rsanheim/plur-meta/pull/39)
* Fix release thinger [#38](https://github.com/rsanheim/plur-meta/pull/38)

## v0.6.8 - 2025-06-12

* Fix: update ci [#37](https://github.com/rsanheim/plur-meta/pull/37)
* Migrate documentation from Jekyll to MkDocs [#33](https://github.com/rsanheim/plur-meta/pull/33)

## v0.6.7 - 2025-06-12

* Trying out Kong CLI [#29](https://github.com/rsanheim/plur-meta/pull/29)
* GitHub mcp setup [#32](https://github.com/rsanheim/plur-meta/pull/32)
* Reorg all test projects to fixtures/projects: [#25](https://github.com/rsanheim/plur-meta/pull/25)
* Update and reorg docs [#24](https://github.com/rsanheim/plur-meta/pull/24)
* Update backspint to latest 0.4.1; update logging [#23](https://github.com/rsanheim/plur-meta/pull/23)

### v0.6.6 (2025-06-08)

- Extract watcher downloader into separate module
- Various test suite fixes and cleanup
- CI configuration improvements

### v0.6.5 (2025-06-08)
- Fast parallel test runner for Ruby/RSpec via runtime-based test distribution  ([#12](https://github.com/rsanheim/plur-meta/pull/12), [#15](https://github.com/rsanheim/plur-meta/pull/15))
- Watcher mode support ([#19](https://github.com/rsanheim/plur-meta/pull/19), [#20](https://github.com/rsanheim/plur-meta/pull/20))
- Backspin (formerly dubplate) for CLI testing ([#18](https://github.com/rsanheim/plur-meta/pull/18))
- Test cleanup ([#13](https://github.com/rsanheim/plur-meta/pull/13), [#14](https://github.com/rsanheim/plur-meta/pull/14), [#17](https://github.com/rsanheim/plur-meta/pull/17))
- Test project cleanup ([#21](https://github.com/rsanheim/plur-meta/pull/21))

### v0.0.1 (2025-05-22)
- Initial project creation!
- Basic parallel test execution
- CI setup ([#6](https://github.com/rsanheim/plur-meta/pull/6))
- Colorized output support ([#9](https://github.com/rsanheim/plur-meta/pull/9), [#10](https://github.com/rsanheim/plur-meta/pull/10))
- Single formatter support ([#11](https://github.com/rsanheim/plur-meta/pull/11))
