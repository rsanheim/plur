# rux CHANGELOG


## v0.7.0 - 2025-06-17

* Replace urfave/cli with Kong [#44](https://github.com/rsanheim/rux-meta/pull/44)
* Convert all tests to use testify assertions [#45](https://github.com/rsanheim/rux-meta/pull/45)

## v0.6.10 - 2025-06-15

* Consolidating init [#43](https://github.com/rsanheim/rux-meta/pull/43)

## v0.6.9 - 2025-06-12
Lotsa internal cleanup
* Extract execution logic and add Kong CLI experiment [#31](https://github.com/rsanheim/rux-meta/pull/31)
* Add the integration test to CI (oops) [#41](https://github.com/rsanheim/rux-meta/pull/41)
* get caching working for integration in CI [#40](https://github.com/rsanheim/rux-meta/pull/40)
* Planning for auto config [#35](https://github.com/rsanheim/rux-meta/pull/35)
* sort and clean up. [#39](https://github.com/rsanheim/rux-meta/pull/39)
* Fix release thinger [#38](https://github.com/rsanheim/rux-meta/pull/38)

## v0.6.8 - 2025-06-12

* Fix: update ci [#37](https://github.com/rsanheim/rux-meta/pull/37)
* Migrate documentation from Jekyll to MkDocs [#33](https://github.com/rsanheim/rux-meta/pull/33)

## v0.6.7 - 2025-06-12

* Trying out Kong CLI [#29](https://github.com/rsanheim/rux-meta/pull/29)
* GitHub mcp setup [#32](https://github.com/rsanheim/rux-meta/pull/32)
* Reorg all test projects to fixtures/projects: [#25](https://github.com/rsanheim/rux-meta/pull/25)
* Update and reorg docs [#24](https://github.com/rsanheim/rux-meta/pull/24)
* Update backspint to latest 0.4.1; update logging [#23](https://github.com/rsanheim/rux-meta/pull/23)

### v0.6.6 (2025-06-08)

- Extract watcher downloader into separate module
- Various test suite fixes and cleanup
- CI configuration improvements

### v0.6.5 (2025-06-08)
- Fast parallel test runner for Ruby/RSpec via runtime-based test distribution  ([#12](https://github.com/rsanheim/rux-meta/pull/12), [#15](https://github.com/rsanheim/rux-meta/pull/15))
- Watcher mode support ([#19](https://github.com/rsanheim/rux-meta/pull/19), [#20](https://github.com/rsanheim/rux-meta/pull/20))
- Backspin (formerly dubplate) for CLI testing ([#18](https://github.com/rsanheim/rux-meta/pull/18))
- Test cleanup ([#13](https://github.com/rsanheim/rux-meta/pull/13), [#14](https://github.com/rsanheim/rux-meta/pull/14), [#17](https://github.com/rsanheim/rux-meta/pull/17))
- Test project cleanup ([#21](https://github.com/rsanheim/rux-meta/pull/21))

### v0.0.1 (2025-05-22)
- Initial project creation!
- Basic parallel test execution
- CI setup ([#6](https://github.com/rsanheim/rux-meta/pull/6))
- Colorized output support ([#9](https://github.com/rsanheim/rux-meta/pull/9), [#10](https://github.com/rsanheim/rux-meta/pull/10))
- Single formatter support ([#11](https://github.com/rsanheim/rux-meta/pull/11))
