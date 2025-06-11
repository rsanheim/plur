# Backspin IO Capture Design

## Overview

This design proposes a simple, robust IO capture mechanism for backspin that captures ALL stdout/stderr output at the IO level, regardless of the output method used (puts, print, warn, TTY::Command, system calls, etc.).

## Core Implementation

### 1. IO Capture Module

```ruby
module Backspin
  module IOCapture
    class CaptureResult
      attr_reader :stdout, :stderr, :result
      
      def initialize(stdout:, stderr:, result:)
        @stdout = stdout
        @stderr = stderr
        @result = result
      end
    end
    
    # Core capture method that redirects IO at the file descriptor level
    def self.capture(&block)
      # Save original file descriptors
      original_stdout = $stdout.dup
      original_stderr = $stderr.dup
      
      # Create temporary StringIO objects
      captured_stdout = StringIO.new
      captured_stderr = StringIO.new
      
      # Redirect at multiple levels for maximum compatibility
      begin
        # Level 1: Ruby global variables
        $stdout = captured_stdout
        $stderr = captured_stderr
        
        # Level 2: Constants (some libraries use these directly)
        silence_warnings do
          Object.const_set(:STDOUT, captured_stdout)
          Object.const_set(:STDERR, captured_stderr)
        end
        
        # Level 3: File descriptor level (catches system calls)
        stdout_fd = STDOUT.fileno
        stderr_fd = STDERR.fileno
        
        # Create temporary files for FD-level capture
        stdout_tempfile = Tempfile.new('backspin-stdout')
        stderr_tempfile = Tempfile.new('backspin-stderr')
        
        # Duplicate and redirect file descriptors
        stdout_backup = IO.for_fd(stdout_fd).dup
        stderr_backup = IO.for_fd(stderr_fd).dup
        
        # Redirect STDOUT/STDERR to tempfiles
        STDOUT.reopen(stdout_tempfile)
        STDERR.reopen(stderr_tempfile)
        
        # Execute the block
        result = yield
        
        # Flush to ensure all output is captured
        STDOUT.flush
        STDERR.flush
        
        # Read captured output from tempfiles
        stdout_tempfile.rewind
        stderr_tempfile.rewind
        fd_stdout = stdout_tempfile.read
        fd_stderr = stderr_tempfile.read
        
        # Combine StringIO and FD captures
        final_stdout = captured_stdout.string + fd_stdout
        final_stderr = captured_stderr.string + fd_stderr
        
        CaptureResult.new(
          stdout: final_stdout,
          stderr: final_stderr,
          result: result
        )
      ensure
        # Restore everything in reverse order
        
        # Restore file descriptors
        STDOUT.reopen(stdout_backup) if stdout_backup
        STDERR.reopen(stderr_backup) if stderr_backup
        
        # Restore constants
        silence_warnings do
          Object.const_set(:STDOUT, original_stdout)
          Object.const_set(:STDERR, original_stderr)
        end
        
        # Restore global variables
        $stdout = original_stdout
        $stderr = original_stderr
        
        # Clean up tempfiles
        stdout_tempfile&.close!
        stderr_tempfile&.close!
      end
    end
    
    private
    
    def self.silence_warnings
      old_verbose = $VERBOSE
      $VERBOSE = nil
      yield
    ensure
      $VERBOSE = old_verbose
    end
  end
end
```

### 2. Integration with Backspin Recorder

```ruby
module Backspin
  class Recorder
    # Enhanced record method that captures IO
    def record_with_io_capture(command_type, *args, **kwargs, &block)
      start_time = Time.now
      
      # Capture all IO during block execution
      capture_result = IOCapture.capture do
        if block_given?
          yield
        else
          # For method calls, execute them
          execute_command(command_type, *args, **kwargs)
        end
      end
      
      # Record the command with captured output
      command = {
        command_type: command_type,
        args: args,
        kwargs: kwargs,
        stdout: capture_result.stdout,
        stderr: capture_result.stderr,
        result: capture_result.result,
        recorded_at: Time.now.iso8601,
        duration: Time.now - start_time
      }
      
      @commands << command
      
      # Return the original result to maintain compatibility
      capture_result.result
    end
  end
end
```

### 3. Enhanced Playback Support

```ruby
module Backspin
  class Player
    def play_with_io(command)
      # During playback, we can optionally replay the output
      if @options[:replay_output]
        $stdout.write(command[:stdout]) if command[:stdout]
        $stderr.write(command[:stderr]) if command[:stderr]
      end
      
      # Return the recorded result
      command[:result]
    end
  end
end
```

## Usage Examples

### Basic Usage

```ruby
require 'backspin'

Backspin.record do
  # This captures ALL output, regardless of method
  result = Backspin::IOCapture.capture do
    puts "Regular puts"
    print "Regular print"
    warn "Warning message"
    
    # System calls
    system("echo 'System echo'")
    `echo 'Backtick echo'`
    
    # TTY::Command
    cmd = TTY::Command.new
    cmd.run("ls -la")
    
    # Even output from required files
    require_relative 'noisy_library'
    
    # Return value
    42
  end
  
  result.stdout # => Contains all stdout from all sources
  result.stderr # => Contains all stderr from all sources
  result.result # => 42
end
```

### Integration with RSpec

```ruby
RSpec.describe "My Feature" do
  it "captures all output during tests" do
    Backspin.record do
      captured = Backspin::IOCapture.capture do
        # Your test code here
        system("rspec spec/some_spec.rb")
      end
      
      expect(captured.stdout).to include("examples, 0 failures")
    end
  end
end
```

### Nested Capture Support

```ruby
# The design handles nested captures by maintaining a stack
captured_outer = Backspin::IOCapture.capture do
  puts "Outer start"
  
  captured_inner = Backspin::IOCapture.capture do
    puts "Inner message"
  end
  
  puts "Outer end"
  puts "Inner captured: #{captured_inner.stdout}"
end

# captured_outer.stdout contains:
# "Outer start\nOuter end\nInner captured: Inner message\n"
```

## Edge Cases Handled

### 1. Direct FD Writes
```ruby
# Even direct file descriptor writes are captured
Backspin::IOCapture.capture do
  IO.for_fd(1).puts "Direct to FD 1"
  IO.for_fd(2).puts "Direct to FD 2"
end
```

### 2. Forked Processes
```ruby
# For forked processes, we need to handle carefully
Backspin::IOCapture.capture do
  fork do
    puts "From forked process"
  end
  Process.wait
end
```

### 3. Thread Safety
```ruby
# The implementation should be thread-safe
Backspin::IOCapture.capture do
  threads = 10.times.map do |i|
    Thread.new { puts "Thread #{i}" }
  end
  threads.each(&:join)
end
```

### 4. Binary Output
```ruby
# Handle binary data correctly
Backspin::IOCapture.capture do
  $stdout.binmode
  $stdout.write("\x00\x01\x02\x03")
end
```

## Implementation Notes

1. **Performance**: Use Tempfile for FD-level capture to handle large outputs efficiently
2. **Compatibility**: The three-level approach ensures maximum compatibility:
   - Ruby global variables ($stdout, $stderr)
   - Constants (STDOUT, STDERR)
   - File descriptors (for system calls)
3. **Cleanup**: Ensure proper cleanup in the ensure block
4. **Encoding**: Handle different encodings properly
5. **Signal Safety**: Consider signal handling for cleanup

## Alternative Simplified Version

For cases where FD-level capture isn't needed:

```ruby
module Backspin
  module SimpleIOCapture
    def self.capture(&block)
      original_stdout = $stdout
      original_stderr = $stderr
      
      stdout = StringIO.new
      stderr = StringIO.new
      
      $stdout = stdout
      $stderr = stderr
      
      result = yield
      
      {
        stdout: stdout.string,
        stderr: stderr.string,
        result: result
      }
    ensure
      $stdout = original_stdout
      $stderr = original_stderr
    end
  end
end
```

## Testing Strategy

```ruby
RSpec.describe Backspin::IOCapture do
  it "captures puts output" do
    result = Backspin::IOCapture.capture { puts "Hello" }
    expect(result.stdout).to eq("Hello\n")
  end
  
  it "captures system command output" do
    result = Backspin::IOCapture.capture { system("echo Hello") }
    expect(result.stdout).to eq("Hello\n")
  end
  
  it "captures TTY::Command output" do
    result = Backspin::IOCapture.capture do
      cmd = TTY::Command.new
      cmd.run("echo Hello")
    end
    expect(result.stdout).to include("Hello")
  end
  
  it "handles nested captures" do
    outer = Backspin::IOCapture.capture do
      puts "outer"
      inner = Backspin::IOCapture.capture { puts "inner" }
      puts "captured: #{inner.stdout.strip}"
    end
    expect(outer.stdout).to eq("outer\ncaptured: inner\n")
  end
end
```

## Benefits

1. **Universal**: Captures output from any source
2. **Simple**: No method stubbing or monkey patching required
3. **Reliable**: Works at the IO level, not dependent on specific methods
4. **Compatible**: Works with TTY::Command, Open3, system calls, etc.
5. **Testable**: Easy to test and verify behavior

## Integration Path

1. Add IOCapture module to backspin
2. Enhance Recorder to use IOCapture for command recording
3. Update Player to support output replay
4. Add configuration options for capture levels
5. Document usage patterns