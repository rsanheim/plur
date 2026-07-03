# Release Process

The operator-facing release runbook lives in the release command help:

```bash
script/release --help
```

Keep command behavior, release steps, artifact expectations, and verification details in `script/release --help` so an operator can work from the terminal without jumping between docs.

This page exists as the stable documentation entry point for release process navigation.

Implementation references:

* `script/release` parses the CLI.
* `lib/plur/release.rb` implements `prepare` and `push`.
* `.github/workflows/release.yml` runs the tag-triggered release.
* `.goreleaser.yml` defines release artifacts and Homebrew publishing.
