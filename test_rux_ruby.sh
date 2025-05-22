#!/bin/bash

echo "Testing rux with rux-ruby project..."
echo

cd rux-ruby

# Test dry-run with auto-discovery
echo "1. Testing dry-run with auto-discovery:"
rux --dry-run
echo

# Test dry-run with specific file
echo "2. Testing dry-run with specific spec file:"
rux --dry-run spec/calculator_spec.rb
echo

# Test dry-run with multiple specific files
echo "3. Testing dry-run with multiple spec files:"
rux --dry-run spec/calculator_spec.rb spec/rux_ruby_spec.rb
echo

# Test actual execution with auto-discovery
echo "4. Running all specs in parallel (auto-discovery):"
rux
echo

# Test with --auto flag (bundle install + run)
echo "5. Running with --auto flag:"
rux --auto
echo

echo "rux-ruby testing complete!"
echo "✅ Both spec files should have run in parallel with interleaved output"