# File Watcher CLI Ecosystem Comparison

This document provides a comparison of popular user-level file watcher command-line tools across different ecosystems (JavaScript, Ruby, Rust, and Go). It also analyzes how these tools differ from `plur` and the proposed new watcher system.

## JavaScript

### `nodemon`

*   Link: [https://nodemon.io/](https://nodemon.io/)
*   Description: `nodemon` is a very popular tool in the Node.js ecosystem. It's a simple command-line utility that wraps your application and automatically restarts it when files change.
*   Configuration: `nodemon` can be configured through a `nodemon.json` file, command-line flags, or in the `package.json` file.
*   Stars: 26666
*   Last Commit: [2025-09-08](https://github.com/remy/nodemon/commit/89e92da6416577eca19bb842a494eed3be638c30)
*   Last Release: [v3.1.10 (2025-04-23)](https://github.com/remy/nodemon/releases/tag/v3.1.10)
*   Example Usage: `nodemon server.js`
*   Comparison with `plur`: `nodemon` is similar to the proposed `restart` feature for `plur`. However, `plur` is a more general-purpose tool that can be used to run any command, not just restart a Node.js application.

### `chokidar-cli`

*   Link: [https://github.com/open-cli-tools/chokidar-cli](https://github.com/open-cli-tools/chokidar-cli)
*   Description: `chokidar-cli` is a command-line interface for the `chokidar` library. It allows you to run commands in response to file changes.
*   Configuration: `chokidar-cli` is configured through command-line flags.
*   Stars: 860
*   Last Commit: [2025-09-22](https://github.com/open-cli-tools/chokidar-cli/commit/8dd8a1e8631d377de600f628d819a0cda46c102f)
*   Last Release: [v3.0.0 (2021-07-28)](https://github.com/open-cli-tools/chokidar-cli/releases/tag/v3.0.0)
*   Example Usage: `chokidar "**/*.js" -c "npm run build-js"`
*   Comparison with `plur`: `chokidar-cli` is very similar to the core functionality of the proposed new watcher system for `plur`. The main difference is that `plur` will be configured through a `.plur.toml` file, which will make it easier to manage complex configurations.

## Ruby

### `guard`

*   Link: [https://github.com/guard/guard](https://github.com/guard/guard)
*   Description: `guard` is a command-line tool that runs commands in response to file changes. It's very popular in the Ruby community.
*   Configuration: `guard` is configured through a `Guardfile` in the root of the project. The `Guardfile` is a Ruby script, which makes it very flexible.
*   Stars: 6598
*   Last Commit: [2025-09-01](https://github.com/guard/guard/commit/b20b7910245aee5cd4ee79fe8dc7984568afdbc4)
*   Last Release: [v2.19.1 (2025-01-02)](https://github.com/guard/guard/releases/tag/v2.19.1)
*   Example Usage: `bundle exec guard`
*   Comparison with `plur`: `guard` is very similar to the proposed new watcher system for `plur`. The main difference is that `guard` is a Ruby-specific tool, while `plur` is a general-purpose tool. The `Guardfile` is also more powerful than the proposed `.plur.toml` file, but it's also more complex.

### `rerun`

*   Link: [https://github.com/alexch/rerun](https://github.com/alexch/rerun)
*   Description: `rerun` is a simple command-line tool that restarts a command when files change.
*   Configuration: `rerun` is configured through command-line flags.
*   Stars: 990
*   Last Commit: [2024-05-22](https://github.com/alexch/rerun/commit/36f2d237985b670752abbe4a7f6814893cdde96f)
*   Last Release: N/A
*   Example Usage: `rerun 'rackup --port 8080'`
*   Comparison with `plur`: `rerun` is a simpler alternative to `guard`. It's similar to the proposed `restart` feature for `plur`.

## Rust

### `watchexec`

*   Link: [https://github.com/watchexec/watchexec](https://github.com/watchexec/watchexec)
*   Description: `watchexec` is a popular, general-purpose command-line tool for watching files and running commands.
*   Configuration: `watchexec` is configured through command-line flags.
*   Stars: 6463
*   Last Commit: [2025-10-27](https://github.com/watchexec/watchexec/commit/49afec17c1a834aa4b5fbbda8f230b2dc16092fb)
*   Last Release: [v2.3.2 (2025-05-18)](https://github.com/watchexec/watchexec/releases/tag/v2.3.2)
*   Example Usage: `watchexec -e go -- rake test:go`
*   Comparison with `plur`: `watchexec` is very similar to the proposed new watcher system for `plur`. The main difference is that `watchexec` is configured through command-line flags, while `plur` will be configured through a `.plur.toml` file. This will make `plur`'s configuration more explicit and easier to manage for complex projects.

### `cargo-watch`

*   Link: [https://github.com/watchexec/cargo-watch](https://github.com/watchexec/cargo-watch)
*   Description: `cargo-watch` is a specialized tool for Rust projects that uses `watchexec` to watch for file changes and run `cargo` commands.
*   Configuration: `cargo-watch` is configured through command-line flags.
*   Stars: 2845
*   Last Commit: [2025-01-14](https://github.com/watchexec/cargo-watch/commit/58e792b99fade3b0c0ad198fd9627dcd4817eb37)
*   Last Release: [v8.5.3 (2024-10-02)](https://github.com/watchexec/cargo-watch/releases/tag/v8.5.3)
*   Example Usage: `cargo watch -x test`
*   Comparison with `plur`: `cargo-watch` is a good example of a specialized tool for a specific ecosystem. `plur` is a more general-purpose tool, but it could be used to create a similar experience for other ecosystems through its configuration file.

### `bacon`

*   Link: [https://github.com/Canop/bacon](https://github.com/Canop/bacon)
*   Description: `bacon` is a background code checker for Rust, providing fast and continuous feedback.
*   Configuration: `bacon` is configured through a `.bacon.toml` file.
*   Stars: 2856
*   Last Commit: [2025-11-04](https://github.com/Canop/bacon/commit/a2efe05b1a876de0fc2b9e899ba42af84800b2e8)
*   Last Release: [v3.19.0 (2025-10-13)](https://github.com/Canop/bacon/releases/tag/v3.19.0)
*   Example Usage: `bacon test`
*   Comparison with `plur`: `bacon` is highly specialized for the Rust development workflow, offering more than just file watching. `plur` is a general-purpose test runner, not a language-specific development server.

## Go

### `go-watcher`

*   Link: [https://github.com/canthefason/go-watcher](https://github.com/canthefason/go-watcher)
*   Description: `go-watcher` is a command-line tool for watching `.go` files and restarting applications.
*   Configuration: `go-watcher` is configured through command-line flags.
*   Stars: 422
*   Last Commit: [2021-04-01](https://github.com/canthefason/go-watcher/commit/59980168ee35c24b8dc3c3bb3dec2ed749fa5c44)
*   Last Release: [v0.2.4 (2017-02-03)](https://github.com/canthefason/go-watcher/releases/tag/v0.2.4)
*   Example Usage: `go-watcher`
*   Comparison with `plur`: `go-watcher` is another example of a specialized tool for a specific ecosystem. `plur`'s proposed system could provide a similar out-of-the-box experience for Go projects.

### `air`

*   Link: [https://github.com/air-verse/air](https://github.com/air-verse/air)
*   Description: `air` is a command-line tool for watching Go files and live-reloading a Go application.
*   Configuration: `air` is configured through a `.air.toml` file.
*   Stars: 22117
*   Last Commit: [2025-09-07](https://github.com/air-verse/air/commit/53e3fb858de84a75d227eb784479b88d11d00754)
*   Last Release: [v1.63.0 (2025-09-07)](https://github.com/air-verse/air/releases/tag/v1.63.0)
*   Example Usage: `air`
*   Comparison with `plur`: `air` is very similar to `nodemon` and `go-watcher`. It's a good example of a tool that is focused on the live-reloading use case. `plur`'s proposed system could be used to create a similar experience, but it would be more flexible.

## General Purpose

### `watchman`

*   Link: [https://github.com/facebook/watchman](https://github.com/facebook/watchman)
*   Description: `watchman` is a file watching service by Facebook that monitors files and directories for changes, and can trigger actions.
*   Configuration: `watchman` can be configured via command-line arguments or a JSON configuration file.
*   Stars: 13333
*   Last Commit: [2025-11-04](https://github.com/facebook/watchman/commit/9a4db76a2a0cdc23ab58b2e7efab03c01b9946c7)
*   Last Release: [v2025.11.03.00 (2025-11-03)](https://github.com/facebook/watchman/releases/tag/v2025.11.03.00)
*   Example Usage: `watchman-make -p '**/*.js' 'build-js'`
*   Comparison with `plur`: `watchman` is a low-level file watching utility that other tools can build upon. `plur` aims to provide a higher-level, opinionated test runner experience.

### `entr`

*   Link: [https://github.com/eradman/entr](https://github.com/eradman/entr)
*   Description: `entr` is a command-line utility that executes arbitrary commands when files change. It's known for its simplicity and Unix philosophy.
*   Configuration: `entr` is configured through command-line flags and standard input.
*   Stars: 5335
*   Last Commit: [2025-10-31](https://github.com/eradman/entr/commit/9273c1bb52500084525874ff4a66ca9fdf9fb587)
*   Last Release: N/A
*   Example Usage: `find . -name '*.rb' | entr -r plur`
*   Comparison with `plur`: `entr` is a very minimalist tool, focusing solely on executing commands on file changes. `plur` aims for a more integrated experience, especially for test running, with configuration and database management.

### `fswatch`

*   Link: [https://github.com/emcrisostomo/fswatch](https://github.com/emcrisostomo/fswatch)
*   Description: `fswatch` is a cross-platform file change monitor with a rich feature set.
*   Configuration: `fswatch` is configured through command-line flags.
*   Stars: 5367
*   Last Commit: [2025-02-15](https://github.com/emcrisostomo/fswatch/commit/5c443d22c53df1eef661d780d816700935a51e1b)
*   Last Release: [1.18.3 (2025-02-04)](https://github.com/emcrisostomo/fswatch/releases/tag/1.18.3)
*   Example Usage: `fswatch . | xargs -n1 -I{} make`
*   Comparison with `plur`: `fswatch` is a very flexible and powerful file watcher, but like many others, it is a general purpose tool. `plur` is focused on the specific use case of running tests in parallel.

## Python

### `watchdog`

*   Link: [https://github.com/gorakhargosh/watchdog](https://github.com/gorakhargosh/watchdog)
*   Description: `watchdog` is a Python library and a command-line tool (`watchmedo`) to monitor file system events.
*   Configuration: `watchmedo` is configured through command-line flags.
*   Stars: 7142
*   Last Commit: [2025-10-27](https://github.com/gorakhargosh/watchdog/commit/e87935af702005ecb233df5ec0d5d3ef3198c673)
*   Last Release: [6.0.0 (2024-11-01)](https://github.com/gorakhargosh/watchdog/releases/tag/v6.0.0)
*   Example Usage: `watchmedo shell-command --patterns="*.py;*.js" --recursive --command='echo "File changed: ${watch_src_path}"'`
*   Comparison with `plur`: `watchdog` provides both a library and a CLI tool. The `watchmedo` tool is similar to other CLI watchers, while `plur` is a more specialized tool for running tests in parallel.

## Summary

| Tool | Language | Configuration | Stars | Last Commit | Last Release |
| --- | --- | --- | --- | --- | --- |
| [`nodemon`](https://github.com/remy/nodemon) | JavaScript | `nodemon.json`, CLI flags, `package.json` | 26666 | [2025-09-08](https://github.com/remy/nodemon/commit/89e92da6416577eca19bb842a494eed3be638c30) | [v3.1.10 (2025-04-23)](https://github.com/remy/nodemon/releases/tag/v3.1.10) |
| [`chokidar-cli`](https://github.com/open-cli-tools/chokidar-cli) | JavaScript | CLI flags | 860 | [2025-09-22](https://github.com/open-cli-tools/chokidar-cli/commit/8dd8a1e8631d377de600f628d819a0cda46c102f) | [v3.0.0 (2021-07-28)](https://github.com/open-cli-tools/chokidar-cli/releases/tag/v3.0.0) |
| [`guard`](https://github.com/guard/guard) | Ruby | `Guardfile` (Ruby script) | 6598 | [2025-09-01](https://github.com/guard/guard/commit/b20b7910245aee5cd4ee79fe8dc7984568afdbc4) | [v2.19.1 (2025-01-02)](https://github.com/guard/guard/releases/tag/v2.19.1) |
| [`rerun`](https://github.com/alexch/rerun) | Ruby | CLI flags | 990 | [2024-05-22](https://github.com/alexch/rerun/commit/36f2d237985b670752abbe4a7f6814893cdde96f) | N/A |
| [`watchexec`](https://github.com/watchexec/watchexec) | Rust | CLI flags | 6463 | [2025-10-27](https://github.com/watchexec/watchexec/commit/49afec17c1a834aa4b5fbbda8f230b2dc16092fb) | [v2.3.2 (2025-05-18)](https://github.com/watchexec/watchexec/releases/tag/v2.3.2) |
| [`cargo-watch`](https://github.com/watchexec/cargo-watch) | Rust | CLI flags | 2845 | [2025-01-14](https://github.com/watchexec/cargo-watch/commit/58e792b99fade3b0c0ad198fd9627dcd4817eb37) | [v8.5.3 (2024-10-02)](https://github.com/watchexec/cargo-watch/releases/tag/v8.5.3) |
| [`go-watcher`](https://github.com/canthefason/go-watcher) | Go | CLI flags | 422 | [2021-04-01](https://github.com/canthefason/go-watcher/commit/59980168ee35c24b8dc3c3bb3dec2ed749fa5c44) | [v0.2.4 (2017-02-03)](https://github.com/canthefason/go-watcher/releases/tag/v0.2.4) |
| [`air`](https://github.com/air-verse/air) | Go | `.air.toml` file | 22117 | [2025-09-07](https://github.com/air-verse/air/commit/53e3fb858de84a75d227eb784479b88d11d00754) | [v1.63.0 (2025-09-07)](https://github.com/air-verse/air/releases/tag/v1.63.0) |
| [`watchman`](https://github.com/facebook/watchman) | C | CLI flags, JSON | 13333 | [2025-11-04](https://github.com/facebook/watchman/commit/9a4db76a2a0cdc23ab58b2e7efab03c01b9946c7) | [v2025.11.03.00 (2025-11-03)](https://github.com/facebook/watchman/releases/tag/v2025.11.03.00) |
| [`entr`](https://github.com/eradman/entr) | C | CLI flags, stdin | 5335 | [2025-10-31](https://github.com/eradman/entr/commit/9273c1bb52500084525874ff4a66ca9fdf9fb587) | N/A |
| [`fswatch`](https://github.com/emcrisostomo/fswatch) | C++ | CLI flags | 5367 | [2025-02-15](https://github.com/emcrisostomo/fswatch/commit/5c443d22c53df1eef661d780d816700935a51e1b) | [1.18.3 (2025-02-04)](https://github.com/emcrisostomo/fswatch/releases/tag/1.18.3) |
| [`watchdog`](https://github.com/gorakhargosh/watchdog) | Python | CLI flags | 7142 | [2025-10-27](https://github.com/gorakhargosh/watchdog/commit/e87935af702005ecb233df5ec0d5d3ef3198c673) | [6.0.0 (2024-11-01)](https://github.com/gorakhargosh/watchdog/releases/tag/v6.0.0) |
| [`bacon`](https://github.com/Canop/bacon) | Rust | `.bacon.toml` | 2856 | [2025-11-04](https://github.com/Canop/bacon/commit/a2efe05b1a876de0fc2b9e899ba42af84800b2e8) | [v3.19.0 (2025-10-13)](https://github.com/Canop/bacon/releases/tag/v3.19.0) |

## Conclusion

The proposed new watcher system for `plur` is well-positioned in the existing ecosystem of file watcher tools. It provides a good balance of simplicity, flexibility, and power. The use of a configuration file for defining mapping rules is a key differentiator from tools like `watchexec`, and will make `plur` a more attractive option for complex projects. The ability to define default rules for different ecosystems will provide a great out-of-the-box experience for new users.
