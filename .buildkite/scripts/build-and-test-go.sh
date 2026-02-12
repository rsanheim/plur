#!/usr/bin/env bash
set -euo pipefail

# Install Go into the Ruby container
arch="$(uname -m)"
if [ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ]; then
  go_arch="arm64"
else
  go_arch="amd64"
fi

echo "--- :go: Installing Go 1.25.5 (${go_arch})"
curl -fsSL "https://go.dev/dl/go1.25.5.linux-${go_arch}.tar.gz" | tar -C /usr/local -xz
export PATH="/usr/local/go/bin:${PATH}"
export PATH="${PATH}:$(go env GOPATH)/bin"
go version

echo "--- :ruby: Installing gems"
bundle install --path vendor/bundle

echo "--- :go: Build + Lint"
bin/rake build lint

echo "--- :go: Verify binary"
./plur/plur --version
./plur/plur --help
./plur/plur doctor

echo "--- :go: Go tests"
bin/rake test:go
