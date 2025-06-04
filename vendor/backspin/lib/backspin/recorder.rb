require "open3"
require "ostruct"
require "rspec/mocks"

module Backspin
  # Handles stubbing and recording of command executions
  class Recorder
    include RSpec::Mocks::ExampleMethods

    attr_reader :commands, :verification_data

    def initialize
      @commands = []
      @verification_data = {}
    end

    def record_calls(*command_types)
      command_types = [:capture3, :system] if command_types.empty?

      command_types.each do |command_type|
        record_call(command_type)
      end
    end

    def record_call(command_type)
      case command_type
      when :system
        setup_system_call_stub
      when :capture3
        setup_capture3_call_stub
      else
        raise ArgumentError, "Unknown command type: #{command_type}"
      end
    end

    # Setup stubs for playback mode - just return recorded values
    def setup_playback_stub(command)
      if command.method_class == Open3::Capture3
        actual_status = OpenStruct.new(exitstatus: command.status)
        allow(Open3).to receive(:capture3).and_return([command.stdout, command.stderr, actual_status])
      elsif command.method_class == ::Kernel::System
        # For system, return true if exit status was 0
        allow_any_instance_of(Object).to receive(:system).and_return(command.status == 0)
      end
    end

    # Setup stubs for verification - capture actual output
    def setup_verification_stub(command)
      @verification_data = {}

      if command.method_class == Open3::Capture3
        allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
          stdout, stderr, status = original_method.call(*args)
          @verification_data["stdout"] = stdout
          @verification_data["stderr"] = stderr
          @verification_data["status"] = status.exitstatus
          [stdout, stderr, status]
        end
      elsif command.method_class == ::Kernel::System
        allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
          # Execute the real system call
          result = original_method.call(receiver, *args)

          # For system calls, we only track the exit status
          @verification_data["stdout"] = ""
          @verification_data["stderr"] = ""
          # Derive exit status from result: true = 0, false = non-zero
          @verification_data["status"] = result ? 0 : 1

          result
        end
      end
    end

    private

    def setup_capture3_call_stub
      allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
        # Execute the real command
        stdout, stderr, status = original_method.call(*args)

        # Parse command args
        cmd_args = if args.length == 1 && args.first.is_a?(String)
          args.first.split(" ")
        else
          args
        end

        # Create command with interaction data
        command = Command.new(
          method_class: Open3::Capture3,
          args: cmd_args,
          stdout: stdout,
          stderr: stderr,
          status: status.exitstatus,
          recorded_at: Time.now.iso8601
        )
        @commands << command

        # Store output for later access (last one wins)
        Backspin.last_output = stdout

        # Return original result
        [stdout, stderr, status]
      end
    end

    def setup_system_call_stub
      allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
        # Handle different forms of system calls
        # When called as system("echo hello"), receiver is the command string and args is empty
        # When called as system("echo", "hello"), receiver is the test object and args contains the command
        if args.empty? && receiver.is_a?(String)
          # Single string form: system("command and args")
          command_string = receiver
          cmd_args = [command_string]
        else
          # Multi-arg form: system("command", "arg1", "arg2")
          cmd_args = args
        end

        # Execute the real system call
        result = original_method.call(receiver, *args)

        # For system calls, stdout and stderr are not captured
        # The caller of system() doesn't have access to them
        stdout = ""
        stderr = ""
        # Derive exit status from result: true = 0, false = non-zero, nil = command failed
        status = if result
          0
        else
          ((result == false) ? 1 : 127)
        end

        # Parse command args for recording
        parsed_args = if args.empty? && receiver.is_a?(String)
          # Single string form - split the command string
          receiver.split(" ")
        elsif cmd_args.length == 1 && cmd_args.first.is_a?(String)
          # Single string form
          cmd_args.first.split(" ")
        else
          # Multi-arg form - already an array
          cmd_args
        end

        # Create command with interaction data
        command = Command.new(
          method_class: ::Kernel::System,
          args: parsed_args,
          stdout: stdout,
          stderr: stderr,
          status: status,
          recorded_at: Time.now.iso8601
        )
        @commands << command

        # Return the original result (true/false/nil)
        result
      end
    end
  end
end

# Define the Kernel::System class for identification at top level
module ::Kernel
  class System; end
end
