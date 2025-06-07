#!/bin/bash

# Script to test rux watch verbose output with specific file change scenarios

cd test_fixtures/rux-ruby || exit 1

# Clean up any previous log
rm -f rux_watch_verbose2.log

echo "Starting rux watch in verbose mode..."
echo "This will run for about 10 seconds and capture two file change events."
echo "Output will be saved to: test_fixtures/rux-ruby/rux_watch_verbose2.log"
echo

# Start rux watch in background with verbose flag
rux watch --verbose --timeout 10 > rux_watch_verbose2.log 2>&1 &
RUX_PID=$!

# Wait for watch to start
sleep 2

echo "1. Triggering a file change that will run tests (lib/string_utils.rb)..."
echo "# trigger test run" >> lib/string_utils.rb
sleep 3

echo "2. Triggering a file change that won't run tests (lib/test.txt - not a Ruby file)..."
echo "test content" > lib/test.txt
sleep 3

echo "3. Waiting for timeout..."
wait $RUX_PID

# Clean up test file
rm -f lib/test.txt

echo
echo "Done! Check test_fixtures/rux-ruby/rux_watch_verbose2.log for the output."
echo
echo "Here's the full captured output:"
echo "========================================="
cat rux_watch_verbose2.log