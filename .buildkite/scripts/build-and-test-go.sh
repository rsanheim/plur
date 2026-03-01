#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=mise-setup.sh
source .buildkite/scripts/mise-setup.sh

echo "--- :ruby: Installing gems"
bundle config set path vendor/bundle
bundle install

echo "--- :go: Build + Lint"
bin/rake build lint

echo "--- :go: Verify binary"
./plur --version
./plur --help
./plur doctor

echo "--- :go: Go tests"
bin/rake test:go
