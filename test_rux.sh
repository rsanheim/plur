#!/bin/bash

echo "Testing rux CLI..."
echo

# Test dry-run with no args
echo "1. Testing dry-run with no args:"
rux --dry-run
echo

# Test dry-run with specific spec file
echo "2. Testing dry-run with specific spec file:"
rux -n spec/parallel_tests/pids_spec.rb
echo

# Test dry-run with --auto flag
echo "3. Testing dry-run with --auto flag:"
rux --dry-run --auto spec/parallel_tests/pids_spec.rb
echo

# Test dry-run with auto-discovery in rux-ruby
echo "4. Testing dry-run with auto-discovery in rux-ruby:"
cd rux-ruby
rux --dry-run
echo

# Test actual execution of all specs in parallel with --auto
echo "5. Running all specs in parallel with --auto:"
rux --auto
echo

echo "rux testing complete!"