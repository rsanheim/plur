# File Watcher CLI Ecosystem Comparison

This document provides a comparison of popular user-level file watcher command-line tools across different ecosystems (JavaScript, Ruby, Rust, and Go). It also analyzes how these tools differ from `plur` and the proposed new watcher system.

## JavaScript

### `nodemon`

*   **Link:** [https://nodemon.io/](https://nodemon.io/)
*   **Description:** `nodemon` is a very popular tool in the Node.js ecosystem. It's a simple command-line utility that wraps your application and automatically restarts it when files change.
*   **Configuration:** `nodemon` can be configured through a `nodemon.json` file, command-line flags, or in the `package.json` file.
*   **Process Management:** Yes, `nodemon` automatically restarts the application.
*   **Extensibility:** `nodemon` is not designed to be extensible. It's a simple tool for a specific purpose.
*   **Comparison with `plur`:** `nodemon` is similar to the proposed `restart` feature for `plur`. However, `plur` is a more general-purpose tool that can be used to run any command, not just restart a Node.js application.

### `chokidar-cli`

*   **Link:** [https://github.com/open-cli-tools/chokidar-cli](https://github.com/open-cli-tools/chokidar-cli)
*   **Description:** `chokidar-cli` is a command-line interface for the `chokidar` library. It allows you to run commands in response to file changes.
*   **Configuration:** `chokidar-cli` is configured through command-line flags.
*   **Process Management:** No, `chokidar-cli` simply runs a command when a file changes.
*   **Extensibility:** `chokidar-cli` is not designed to be extensible.
*   **Comparison with `plur`:** `chokidar-cli` is very similar to the core functionality of the proposed new watcher system for `plur`. The main difference is that `plur` will be configured through a `.plur.toml` file, which will make it easier to manage complex configurations.

## Ruby

### `guard`

*   **Link:** [https://github.com/guard/guard](https://github.com/guard/guard)
*   **Description:** `guard` is a command-line tool that runs commands in response to file changes. It's very popular in the Ruby community.
*   **Configuration:** `guard` is configured through a `Guardfile` in the root of the project. The `Guardfile` is a Ruby script, which makes it very flexible.
*   **Process Management:** Yes, `guard` can restart long-running processes.
*   **Extensibility:** `guard` has a powerful plugin system that allows it to be extended with new features.
*   **Comparison with `plur`:** `guard` is very similar to the proposed new watcher system for `plur`. The main difference is that `guard` is a Ruby-specific tool, while `plur` is a general-purpose tool. The `Guardfile` is also more powerful than the proposed `.plur.toml` file, but it's also more complex.

### `rerun`

*   **Link:** [https://github.com/alexch/rerun](https://github.com/alexch/rerun)
*   **Description:** `rerun` is a simple command-line tool that restarts a command when files change.
*   **Configuration:** `rerun` is configured through command-line flags.
*   **Process Management:** Yes, `rerun` automatically restarts the command.
*   **Extensibility:** `rerun` is not designed to be extensible.
*   **Comparison with `plur`:** `rerun` is a simpler alternative to `guard`. It's similar to the proposed `restart` feature for `plur`.

## Rust

### `watchexec`

*   **Link:** [https://github.com/watchexec/watchexec](https://github.com/watchexec/watchexec)
*   **Description:** `watchexec` is a popular, general-purpose command-line tool for watching files and running commands.
*   **Configuration:** `watchexec` is configured through command-line flags.
*   **Process Management:** Yes, `watchexec` can restart long-running processes.
*   **Extensibility:** `watchexec` is not designed to be extensible.
*   **Comparison with `plur`:** `watchexec` is very similar to the proposed new watcher system for `plur`. The main difference is that `watchexec` is configured through command-line flags, while `plur` will be configured through a `.plur.toml` file. This will make `plur`'s configuration more explicit and easier to manage for complex projects.

### `cargo-watch`

*   **Link:** [https://github.com/watchexec/cargo-watch](https://github.com/watchexec/cargo-watch)
*   **Description:** `cargo-watch` is a specialized tool for Rust projects that uses `watchexec` to watch for file changes and run `cargo` commands.
*   **Configuration:** `cargo-watch` is configured through command-line flags.
*   **Process Management:** Yes, `cargo-watch` can restart long-running processes.
*   **Extensibility:** `cargo-watch` is not designed to be extensible.
*   **Comparison with `plur`:** `cargo-watch` is a good example of a specialized tool for a specific ecosystem. `plur` is a more general-purpose tool, but it could be used to create a similar experience for other ecosystems through its configuration file.

## Go

### `go-watcher`

*   **Link:** [https://github.com/canthefason/go-watcher](https://github.com/canthefason/go-watcher)
*   **Description:** `go-watcher` is a command-line tool for watching `.go` files and restarting applications.
*   **Configuration:** `go-watcher` is configured through command-line flags.
*   **Process Management:** Yes, `go-watcher` can restart long-running processes.
*   **Extensibility:** `go-watcher` is not designed to be extensible.
*   **Comparison with `plur`:** `go-watcher` is another example of a specialized tool for a specific ecosystem. `plur`'s proposed system could provide a similar out-of-the-box experience for Go projects.

### `air`

*   **Link:** [https://github.com/cosmtrek/air](https://github.com/cosmtrek/air)
*   **Description:** `air` is a command-line tool for watching Go files and live-reloading a Go application.
*   **Configuration:** `air` is configured through a `.air.toml` file.
*   **Process Management:** Yes, `air` automatically restarts the application.
*   **Extensibility:** `air` is not designed to be extensible.
*   **Comparison with `plur`:** `air` is very similar to `nodemon` and `go-watcher`. It's a good example of a tool that is focused on the live-reloading use case. `plur`'s proposed system could be used to create a similar experience, but it would be more flexible.

## Summary

| Tool | Language | Configuration | Process Management | Extensibility |
| --- | --- | --- | --- | --- |
| `nodemon` | JavaScript | `nodemon.json`, CLI flags, `package.json` | Yes | No |
| `chokidar-cli` | JavaScript | CLI flags | No | No |
| `guard` | Ruby | `Guardfile` (Ruby script) | Yes | Yes (plugins) |
| `rerun` | Ruby | CLI flags | Yes | No |
| `watchexec` | Rust | CLI flags | Yes | No |
| `cargo-watch` | Rust | CLI flags | Yes | No |
| `go-watcher` | Go | CLI flags | Yes | No |
| `air` | Go | `.air.toml` file | Yes | No |

## Conclusion

The proposed new watcher system for `plur` is well-positioned in the existing ecosystem of file watcher tools. It provides a good balance of simplicity, flexibility, and power. The use of a configuration file for defining mapping rules is a key differentiator from tools like `watchexec`, and will make `plur` a more attractive option for complex projects. The ability to define default rules for different ecosystems will provide a great out-of-the-box experience for new users.
