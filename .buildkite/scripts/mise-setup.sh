#!/usr/bin/env bash
set -euo pipefail

echo "--- :gear: Installing system dependencies"
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl git build-essential libssl-dev libreadline-dev zlib1g-dev libyaml-dev libffi-dev tzdata locales > /dev/null

# Set UTF-8 locale (ubuntu:24.04 defaults to POSIX/US-ASCII)
sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen
locale-gen en_US.UTF-8 > /dev/null
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

echo "--- :gear: Installing mise"
curl -fsSL https://mise.run | sh
export PATH="$HOME/.local/bin:$PATH"
mise --version

echo "--- :gear: Installing toolchain from .mise.toml"
mise trust
mise install --yes

eval "$(mise activate bash)"
mise current
