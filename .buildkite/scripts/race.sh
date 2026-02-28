#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=mise-setup.sh
source .buildkite/scripts/mise-setup.sh

echo "--- :go: Go tests with race detection"
go test -mod=mod -race -short ./...
