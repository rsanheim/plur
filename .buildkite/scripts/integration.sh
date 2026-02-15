#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=mise-setup.sh
source .buildkite/scripts/mise-setup.sh

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
