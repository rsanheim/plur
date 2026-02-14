#!/usr/bin/env bash
set -euo pipefail

echo "--- :gear: Installing system dependencies"
apt-get update -qq
apt-get install -y -qq curl git build-essential libssl-dev libreadline-dev zlib1g-dev libyaml-dev libffi-dev > /dev/null

echo "--- :gear: Installing mise"
curl -fsSL https://mise.run | sh
export PATH="$HOME/.local/bin:$PATH"
mise --version

echo "--- :gear: Installing toolchain from .mise.toml"
mise install --yes

eval "$(mise activate bash)"
mise current
