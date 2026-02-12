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

echo "--- :go: Installing GoReleaser"
go install github.com/goreleaser/goreleaser/v2@latest

echo "--- :ruby: Installing gems"
bundle config set path vendor/bundle
bundle install

echo "--- :hammer: Installing plur"
bin/rake install
plur watch install
which plur
plur doctor

echo "--- :rubocop: Ruby linting"
bin/rake lint:ruby

echo "--- :rspec: Default Ruby fixture tests"
bin/rake test:default_ruby

echo "--- :rails: Default Rails fixture setup"
cd fixtures/projects/default-rails
bundle config set --local path vendor/bundle
bundle install
bundle exec rake db:create db:migrate

echo "--- :rails: Default Rails fixture tests"
plur

echo "--- :rspec: Full Ruby test suite"
cd /work
gem install turbo_tests
bin/rake test
