#!/usr/bin/env bash
set -euo pipefail

echo "--- :go: Installing GoReleaser"
export PATH="${PATH}:$(go env GOPATH)/bin"
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser --version

echo "--- :go: GoReleaser snapshot build"
cd plur
goreleaser build --snapshot --single-target --clean

echo "--- :go: Verify binary"
arch="$(go env GOARCH)"
if [ "$arch" = "arm64" ]; then
  bin="dist/plur_linux_arm64_v8.0/plur"
else
  bin="dist/plur_linux_${arch}_v1/plur"
fi

if [ ! -x "$bin" ]; then
  echo "Expected binary not found: $bin"
  echo "Available plur binaries:"
  find dist -maxdepth 3 -type f -name plur -print
  exit 1
fi

"$bin" --version
"$bin" doctor
