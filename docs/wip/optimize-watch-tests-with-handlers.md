# Optimizing Watch Tests with TTY::Command Handlers

## Problem
Watch tests are the slowest part of our test suite (3-5 seconds each) because they wait for the full timeout even when the expected output is received early.

## Solution
Create a flexible helper that uses TTY::Command's output streaming to:
1. React to output patterns to trigger actions (like modifying files)
2. Exit early when a success pattern is matched
3. Have a failsafe timeout

## Implementation

### Helper Method

```ruby
def run_plur_watch_with_handlers(
  timeout: 10,
  on_stdout: nil,
  on_stderr: nil,
  **options
)
  cmd = TTY::Command.new(timeout: timeout, uuid: false, printer: :null)
  captured_out = []
  captured_err = []
  
  begin
    result = cmd.run("plur --debug watch run --timeout #{timeout}", **options) do |out, err|
      # Capture output for later use
      captured_out << out if out
      captured_err << err if err
      
      # Call handlers if provided
      on_stdout.call(out) if on_stdout && out
      on_stderr.call(err) if on_stderr && err
    end
    
    # Return result with captured output
    result
  rescue TTY::Command::ExitError => e
    # Create a result-like object with captured output
    OpenStruct.new(
      out: captured_out.join,
      err: captured_err.join,
      success?: false,
      exit_status: 1
    )
  end
end
```

### Simple Test Example

Convert a basic test that just checks startup:

```ruby
it "starts successfully when spec directory exists" do
  started = false
  
  result = run_plur_watch_with_handlers(
    timeout: 2,
    on_stderr: lambda do |err|
      if !started && err.include?("plur watch starting!")
        started = true
        # Exit early by raising exception (like wait does)
        raise TTY::Command::ExitError, "Watch started successfully"
      end
    end
  )
  
  expect(result.err).to include("plur watch starting!")
  expect(result.err).to include("directories=[spec lib]")
end
```

### Complex Test Example

For tests that need to modify files after the watcher starts:

```ruby
it "detects and runs spec when file is modified" do
  watcher_ready = false
  file_modified = false
  
  result = run_plur_watch_with_handlers(
    timeout: 5,
    on_stderr: lambda do |err|
      # Wait for watcher to be ready
      if !watcher_ready && err.include?("s/self/live@")
        watcher_ready = true
        # Modify file to trigger watch
        spec_file = default_ruby_dir.join("spec/calculator_spec.rb")
        spec_file.write(spec_file.read + "\n# trigger")
        file_modified = true
      end
    end,
    on_stdout: lambda do |out|
      # Exit when we see the spec ran
      if file_modified && out.include?("calculator_spec.rb")
        raise TTY::Command::ExitError, "Test completed successfully"
      end
    end
  )
  
  expect(result.out).to include("calculator_spec.rb")
  expect(result.out).to include("bundle exec rspec")
end
```

## Key Benefits

1. **Faster**: Tests exit immediately when success condition is met
2. **Flexible**: Can trigger actions based on output patterns
3. **Reliable**: No thread coordination or sleep statements needed
4. **Simple**: Uses TTY::Command's existing patterns (exception-based early exit)

## How It Works

- TTY::Command's `run` method yields stdout and stderr as they stream
- Handlers can inspect output and perform actions
- Raising `TTY::Command::ExitError` exits the command early (same as `wait` method)
- Captured output is preserved even on early exit

## Next Steps

1. Implement the helper in `spec/support/plur_watch_helper.rb`
2. Convert one simple test to validate the approach
3. If successful, convert remaining watch tests
4. Expected time savings: 30-40 seconds off test suite