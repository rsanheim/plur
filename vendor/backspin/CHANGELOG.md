# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-06-02

### Added
- Initial release of Backspin
- `record` method to capture CLI command outputs
- `verify` and `verify!` methods for output verification
- `use_cassette` method for VCR-style record/replay
- Support for multiple verification modes (strict, playback, custom matcher)
- Auto-naming of cassettes from RSpec context
- Multi-command recording support
- RSpec integration using RSpec's mocking framework