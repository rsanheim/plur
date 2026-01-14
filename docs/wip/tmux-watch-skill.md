# Claude Code Skill: tmux-driven plur watch verification

*Status:* Planned
*Created:* 2026-01-14

## Overview

Create a Claude Code skill (slash command) that enables interactive verification of `plur watch` by driving a tmux session. This allows Claude to observe and interact with the watch mode in a way that simulates real user interaction.

## Background

Claude Code cannot attach to interactive terminal sessions, but tmux can be driven entirely through commands:

* `tmux new-session -d -s <name>` - create detached session
* `tmux send-keys -t <name> "command" Enter` - send input
* `tmux capture-pane -t <name> -p` - read current output
* `tmux kill-session -t <name>` - cleanup

This pattern was successfully used to verify plur watch behavior including file detection, debounce, and clean shutdown.

## Skill Requirements

### Invocation

```
/plur-watch-test [project-path]
```

If no project path provided, use current working directory.

### Capabilities

The skill should support these verification scenarios:

1. **Start watch mode** - Launch `plur watch` in a tmux session
2. **Observe output** - Capture and report what plur watch displays
3. **Trigger file changes** - Modify files to trigger the watcher
4. **Verify detection** - Confirm the watcher detected changes and ran tests
5. **Test debounce** - Rapidly modify files and verify single execution
6. **Test shutdown** - Send Ctrl-C and verify clean exit
7. **Cleanup** - Kill session and verify no zombie processes

### Session Management

* Use a unique session name (e.g., `plur-watch-<timestamp>`)
* Always clean up sessions on exit, even on errors
* Check for existing sessions before creating new ones

### Output Formatting

The skill should provide clear status updates:

```
[plur-watch-test] Starting tmux session in /path/to/project...
[plur-watch-test] plur watch started, watching: lib, spec
[plur-watch-test] Modifying spec/example_spec.rb...
[plur-watch-test] ✓ Watcher detected change, ran: bundle exec rspec spec/example_spec.rb
[plur-watch-test] Testing debounce (3 rapid modifications)...
[plur-watch-test] ✓ Debounce working: 3 changes → 1 test run
[plur-watch-test] Sending Ctrl-C...
[plur-watch-test] ✓ Clean shutdown confirmed
[plur-watch-test] ✓ No zombie processes
[plur-watch-test] All verifications passed!
```

## Implementation Notes

### Key tmux Commands

```bash
# Create session in project directory
tmux new-session -d -s "$SESSION" -c "$PROJECT_PATH"

# Start plur watch
tmux send-keys -t "$SESSION" "plur watch" Enter

# Wait and capture output (last N lines)
sleep 2
tmux capture-pane -t "$SESSION" -p -S -30

# Trigger file change (append content, not just touch)
echo "# trigger $(date +%s)" >> "$PROJECT_PATH/spec/example_spec.rb"

# Send Ctrl-C for shutdown
tmux send-keys -t "$SESSION" C-c

# Kill session
tmux kill-session -t "$SESSION"

# Check for zombies
ps aux | grep -E "(plur|watcher)" | grep -v grep
```

### Parsing Watch Output

Look for these patterns in captured output:

* Startup: `plur .* ready and watching`
* Job run: `bundle exec rspec` or `bundle exec ruby -Itest`
* Shutdown: `shutting down gracefully`

### Error Handling

* Timeout if watch doesn't start within 10 seconds
* Timeout if file change not detected within 5 seconds
* Always attempt cleanup even on failure
* Report which step failed

## File Location

Place the skill implementation in the appropriate Claude Code skills directory. Reference existing plur skills (if any) for patterns.

## Testing the Skill

After implementation, verify by running:

```
/plur-watch-test fixtures/projects/default-ruby
```

Should complete all verification steps and report success.

## Future Enhancements

* Support for custom verification scenarios via arguments
* Integration with CI to run watch verification as part of test suite
* Support for testing multi-directory watching
* Verbose mode with full tmux output capture
