#!/bin/bash
# Interactive test for rux watch

echo "This script will test rux watch functionality"
echo "1. Start rux watch in another terminal: rux watch"
echo "2. Press Enter when ready"
read

echo "Modifying spec/single_failure_spec.rb..."
echo "" >> spec/single_failure_spec.rb

echo "You should see the spec run in the other terminal!"
echo "Press Ctrl+C in the watch terminal to stop"