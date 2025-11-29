# plur CHANGELOG




## v0.15.2 - 2025-11-29

* Fix release notes not appearing in GitHub releases

## v0.15.1 - 2025-11-29

* Add `script/bench-git` [#140](https://github.com/rsanheim/plur/pull/140)

## v0.15.0 - 2025-11-28

* Redesign Runner [#139](https://github.com/rsanheim/plur/pull/139)

## v0.14.0 - 2025-11-28
* more docs clean up [#138](https://github.com/rsanheim/plur/pull/138)
* Clean ups [#137](https://github.com/rsanheim/plur/pull/137)
* Remove framework hints [#136](https://github.com/rsanheim/plur/pull/136)
* Migrate release process to GitHub Actions [#135](https://github.com/rsanheim/plur/pull/135)

## v0.13.0 - 2025-11-25
* Major internal refactor (task→job) [#131](https://github.com/rsanheim/plur/pull/131)
* Remove dead code: test-only getters and unused Ruby helper [#133](https://github.com/rsanheim/plur/pull/133)

## v0.12.0 - 2025-10-31

* Add plur watch reload command [#126](https://github.com/rsanheim/plur/pull/126)
* rename to targets in watch [#125](https://github.com/rsanheim/plur/pull/125)
* Remove default formatter from watch command [#124](https://github.com/rsanheim/plur/pull/124)
* Cleanup old docs [#118](https://github.com/rsanheim/plur/pull/118)

## v0.11.0 - 2025-10-17

* Default to Rspec; fail fast if custom task doesn't exist [#116](https://github.com/rsanheim/plur/pull/116)
* Modernize python mkdocs [#115](https://github.com/rsanheim/plur/pull/115)
* Release testing and refinement [#114](https://github.com/rsanheim/plur/pull/114)
* add gh cli for circle [#113](https://github.com/rsanheim/plur/pull/113)
* Update rails example project [#112](https://github.com/rsanheim/plur/pull/112)

## v0.10.0 - 2025-08-21

* Consolidate to `Task` to for defining how to run things [#100](https://github.com/rsanheim/plur/pull/100)
* fix: align plur doctor output formatting [#104](https://github.com/rsanheim/plur/pull/104)
* Change trigger phrase for automated review [#102](https://github.com/rsanheim/plur/pull/102)
* Performance optimizations and benchmarking infrastructure [#99](https://github.com/rsanheim/plur/pull/99)

## v0.9.11 - 2025-08-13

* Fix "-C" flag for change dir [#98](https://github.com/rsanheim/plur/pull/98)

## v0.9.10 - 2025-08-13
* Implement RSpec-style adaptive duration formatting [#94](https://github.com/rsanheim/plur/pull/94)
* Use doublestar for globs [#90](https://github.com/rsanheim/plur/pull/90)

## v0.9.9 - 2025-08-12

* Improve database tasks spec: [#83](https://github.com/rsanheim/plur/pull/83)
* Clean up docs [#82](https://github.com/rsanheim/plur/pull/82)
* Fix watcher binary install; fix DB task error handling [#81](https://github.com/rsanheim/plur/pull/81)
* gitignore plur-race binary [#80](https://github.com/rsanheim/plur/pull/80)
* Wire up CircleCI MCP [#77](https://github.com/rsanheim/plur/pull/77)
* Re org specs [#76](https://github.com/rsanheim/plur/pull/76)

## v0.9.4 - 2025-07-31

* Add script/install-docker [#74](https://github.com/rsanheim/plur/pull/74)

## v0.9.3 - 2025-07-31

* Default first worker DB number to 1 for parallel specs / DB setup [#75](https://github.com/rsanheim/plur/pull/75)
* Clean up old WIP docs [#73](https://github.com/rsanheim/plur/pull/73)

## v0.9.2 - 2025-07-30

* Config file improvements [#72](https://github.com/rsanheim/plur/pull/72)

## v0.9.1 - 2025-07-22

* Add multi-platform build support [#70](https://github.com/rsanheim/plur/pull/70)
* PLUR: final renames and clean up  [#69](https://github.com/rsanheim/plur/pull/69)

## v0.9.0 - 2025-07-17

* PLUR - the big rename [#51](https://github.com/rsanheim/plur/pull/51)
  - Updated all references in code, documentation, and scripts
  - Binary is now `plur` instead of `rux`
  - Configuration files are now `.plur.toml` instead of `.rux.toml`
  - Home directory is now `~/.plur/` instead of `~/.rux/`
* Update backspin v060 [#68](https://github.com/rsanheim/plur/pull/68)

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
