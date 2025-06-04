require "yaml"
require "fileutils"
require "open3"
require "pathname"
require "ostruct"
require "rspec/mocks"
require "backspin/version"
require "backspin/rspec_metadata"
require "backspin/record"

module Backspin
  class RecordNotFoundError < StandardError; end

  # Include RSpec mocks methods
  extend RSpec::Mocks::ExampleMethods

  # Configuration for Backspin
  class Configuration
    attr_accessor :scrub_credentials
    # The directory where backspin will store its files - defaults to spec/backspin
    attr_accessor :backspin_dir
    # Regex patterns to scrub from saved output
    attr_reader :credential_patterns

    def initialize
      @scrub_credentials = true
      @credential_patterns = default_credential_patterns
      @backspin_dir = Pathname(Dir.pwd).join("spec", "backspin_data")
    end

    def add_credential_pattern(pattern)
      @credential_patterns << pattern
    end

    def clear_credential_patterns
      @credential_patterns = []
    end

    def reset_credential_patterns
      @credential_patterns = default_credential_patterns
    end

    private

    def default_credential_patterns
      [
        # AWS credentials
        /AKIA[0-9A-Z]{16}/,                                    # AWS Access Key ID
        /aws_secret_access_key\s*[:=]\s*["']?([A-Za-z0-9\/+=]{40})["']?/i,  # AWS Secret Key
        /aws_session_token\s*[:=]\s*["']?([A-Za-z0-9\/+=]+)["']?/i,         # AWS Session Token

        # Google Cloud credentials
        /AIza[0-9A-Za-z\-_]{35}/,                              # Google API Key
        /[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com/, # Google OAuth2 client ID
        /-----BEGIN (RSA )?PRIVATE KEY-----/,                  # Private keys

        # Generic patterns
        /api[_-]?key\s*[:=]\s*["']?([A-Za-z0-9\-_]{20,})["']?/i,  # Generic API keys
        /auth[_-]?token\s*[:=]\s*["']?([A-Za-z0-9\-_]{20,})["']?/i, # Auth tokens
        /password\s*[:=]\s*["']?([^"'\s]{8,})["']?/i,             # Passwords
        /secret\s*[:=]\s*["']?([A-Za-z0-9\-_]{20,})["']?/i       # Generic secrets
      ]
    end
  end

  class << self
    def configuration
      @configuration ||= Configuration.new
    end

    def configure
      yield(configuration)
    end

    def reset_configuration!
      @configuration = Configuration.new
    end
  end

  class Result
    attr_reader :commands, :record_path

    def initialize(commands:, record_path:)
      @commands = commands
      @record_path = record_path
    end
  end

  class VerifyResult
    attr_reader :record_path, :expected_output, :actual_output, :diff, :stderr_diff

    def initialize(verified:, record_path:, expected_output: nil, actual_output: nil,
      expected_stderr: nil, actual_stderr: nil, expected_status: nil, actual_status: nil,
      command_executed: true)
      @verified = verified
      @record_path = record_path
      @expected_output = expected_output
      @actual_output = actual_output
      @expected_stderr = expected_stderr
      @actual_stderr = actual_stderr
      @expected_status = expected_status
      @actual_status = actual_status
      @command_executed = command_executed

      if !verified && expected_output && actual_output
        @diff = generate_diff(expected_output, actual_output)
      end

      if !verified && expected_stderr && actual_stderr && expected_stderr != actual_stderr
        @stderr_diff = generate_diff(expected_stderr, actual_stderr)
      end
    end

    def verified?
      @verified
    end

    def output
      @actual_output
    end

    def error_message
      return nil if verified?

      "Output verification failed\nExpected: #{@expected_output.chomp}\nActual: #{@actual_output.chomp}"
    end

    def command_executed?
      @command_executed
    end

    private

    def generate_diff(expected, actual)
      expected_lines = expected.lines
      actual_lines = actual.lines
      diff = []

      # Simple diff for now
      expected_lines.each do |line|
        unless actual_lines.include?(line)
          diff << "-#{line.chomp}"
        end
      end

      actual_lines.each do |line|
        unless expected_lines.include?(line)
          diff << "+#{line.chomp}"
        end
      end

      diff.join("\n")
    end
  end

  class Command
    attr_reader :args, :stdout, :stderr, :status, :recorded_at

    def initialize(method_class:, args:, stdout: nil, stderr: nil, status: nil, recorded_at: nil)
      @method_class = method_class
      @args = args
      @stdout = stdout
      @stderr = stderr
      @status = status
      @recorded_at = recorded_at
    end

    def class
      @method_class
    end

    # Convert to hash for YAML serialization
    def to_h(filter: nil)
      data = {
        "command_type" => @method_class.name,
        "args" => @args,
        "stdout" => Backspin.scrub_text(@stdout),
        "stderr" => Backspin.scrub_text(@stderr),
        "status" => @status,
        "recorded_at" => @recorded_at
      }

      # Apply filter if provided
      if filter
        data = filter.call(data)
      end

      data
    end

    # Create from hash (for loading from YAML)
    def self.from_h(data)
      # Determine method class from command_type
      method_class = case data["command_type"]
      when "Open3::Capture3"
        Open3::Capture3
      when "Kernel::System"
        Kernel::System
      else
        # Default to capture3 for backwards compatibility
        Open3::Capture3
      end

      new(
        method_class: method_class,
        args: data["args"],
        stdout: data["stdout"],
        stderr: data["stderr"],
        status: data["status"],
        recorded_at: data["recorded_at"]
      )
    end
  end

  class << self
    attr_accessor :last_output

    def scrub_text(text)
      return text unless configuration.scrub_credentials && text

      scrubbed = text.dup
      configuration.credential_patterns.each do |pattern|
        scrubbed.gsub!(pattern) do |match|
          # Replace with asterisks of the same length
          "*" * match.length
        end
      end
      scrubbed
    end

    def call(record_name, filter: nil)
      raise ArgumentError, "record_name is required" if record_name.nil? || record_name.empty?

      commands = []
      record_path = build_record_path(record_name)
      capture3_original = Open3.method(:capture3)

      # Use RSpec's and_wrap_original to intercept capture3 calls
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
        commands << command

        # Store output for later access (last one wins)
        Backspin.last_output = stdout

        # Return original result
        [stdout, stderr, status]
      end

      # Also intercept system calls
      system_intercepted = false
      allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
        system_intercepted = true

        # Handle different forms of system calls
        # When called as system("echo hello"), receiver is the command string and args is empty
        # When called as system("echo", "hello"), receiver is the test object and args contains the command
        if args.empty? && receiver.is_a?(String)
          # Single string form: system("command and args")
          command_string = receiver
          capture_args = [command_string]
        else
          # Multi-arg form: system("command", "arg1", "arg2")
          capture_args = args
        end

        # For system calls, we capture output by using the original capture3
        # directly to avoid triggering our stub
        stdout, stderr, status = capture3_original.call(*capture_args)

        # Execute the real system call to get the correct return value
        result = original_method.call(receiver, *args)

        # Parse command args based on how system was called
        cmd_args = if args.empty? && receiver.is_a?(String)
          # Single string form - split the command string
          receiver.split(" ")
        elsif capture_args.length == 1 && capture_args.first.is_a?(String)
          # Single string passed to capture3
          capture_args.first.split(" ")
        else
          # Multi-arg form
          capture_args
        end

        # Create command with interaction data
        command = Command.new(
          method_class: Kernel::System,
          args: cmd_args,
          stdout: stdout,
          stderr: stderr,
          status: status.exitstatus,
          recorded_at: Time.now.iso8601
        )
        commands << command

        # Return original result (true/false)
        result
      end

      yield

      # Save commands using new format
      FileUtils.mkdir_p(File.dirname(record_path))
      # Don't load existing data when creating new record
      record = Record.new(record_path)
      record.clear  # Clear any loaded data
      commands.each { |cmd| record.add_command(cmd) }
      record.save(filter: filter)

      Result.new(commands: commands, record_path: Pathname.new(record_path))
    end

    def output
      last_output
    end

    def use_record(record_name, options = {}, &block)
      raise ArgumentError, "record_name is required" if record_name.nil? || record_name.empty?

      record_path = build_record_path(record_name)
      record_mode = options[:record] || :once
      filter = options[:filter]

      case record_mode
      when :none
        # Never record, only replay
        unless File.exist?(record_path)
          raise RecordNotFoundError, "Record not found: #{record_path}"
        end
        replay_record(record_path, &block)
      when :all
        # Always record
        record_and_save_record(record_path, filter: filter, &block)
      when :once
        # Record if doesn't exist, replay if exists
        if File.exist?(record_path)
          replay_record(record_path, &block)
        else
          record_and_save_record(record_path, filter: filter, &block)
        end
      when :new_episodes
        # Record new commands not in record
        # For now, simplified: just append new recordings
        record_new_episode(record_path, filter: filter, &block)
      else
        raise ArgumentError, "Unknown record mode: #{record_mode}"
      end
    end

    def verify(record_name, mode: :strict, matcher: nil, &block)
      record_path = build_record_path(record_name)

      record = Record.load_or_create(record_path)
      unless record.exists?
        raise RecordNotFoundError, "Record not found: #{record_path}"
      end

      if record.empty?
        raise RecordNotFoundError, "No commands found in record"
      end

      # For verify, we only handle single command verification for now
      # Use the first command
      command = record.commands.first

      if mode == :playback
        # Playback mode: return recorded output without running command
        actual_stdout = command.stdout
        actual_stderr = command.stderr
        actual_status = OpenStruct.new(exitstatus: command.status)

        # Stub based on command type
        if command.class == Open3::Capture3
          allow(Open3).to receive(:capture3).and_return([actual_stdout, actual_stderr, actual_status])
        elsif command.class == Kernel::System
          # For system, return true if exit status was 0
          allow_any_instance_of(Object).to receive(:system).and_return(command.status == 0)
        end

        yield

        # In playback mode, always verified
        VerifyResult.new(
          verified: true,
          record_path: Pathname.new(record_path),
          expected_output: command.stdout,
          actual_output: actual_stdout,
          expected_stderr: command.stderr,
          actual_stderr: actual_stderr,
          expected_status: command.status,
          actual_status: command.status,
          command_executed: false
        )
      elsif matcher
        # Custom matcher verification
        actual_data = {}

        # Use RSpec's and_wrap_original to capture actual output
        if command.class == Open3::Capture3
          allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
            stdout, stderr, status = original_method.call(*args)
            actual_data["stdout"] = stdout
            actual_data["stderr"] = stderr
            actual_data["status"] = status.exitstatus
            [stdout, stderr, status]
          end
        elsif command.class == Kernel::System
          allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
            # Capture output using capture3 for verification
            stdout, stderr, status = Open3.capture3(*args)
            actual_data["stdout"] = stdout
            actual_data["stderr"] = stderr
            actual_data["status"] = status.exitstatus
            # Still call the original system method
            result = original_method.call(receiver, *args)
            result
          end
        end

        yield

        # Call custom matcher - convert command back to hash format for matcher
        recorded_data = command.to_h
        verified = matcher.call(recorded_data, actual_data)

        VerifyResult.new(
          verified: verified,
          record_path: Pathname.new(record_path),
          expected_output: command.stdout,
          actual_output: actual_data["stdout"],
          expected_stderr: command.stderr,
          actual_stderr: actual_data["stderr"],
          expected_status: command.status,
          actual_status: actual_data["status"]
        )
      else
        # Default strict mode
        actual_stdout = nil
        actual_stderr = nil
        actual_status = nil

        # Use RSpec's and_wrap_original to capture actual output
        if command.class == Open3::Capture3
          allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
            actual_stdout, actual_stderr, actual_status = original_method.call(*args)
            [actual_stdout, actual_stderr, actual_status]
          end
        elsif command.class == Kernel::System
          capture3_original = Open3.method(:capture3)
          allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
            # Handle different forms of system calls
            capture_args = if args.empty? && receiver.is_a?(String)
              # Single string form: system("command and args")
              [receiver]
            else
              # Multi-arg form: system("command", "arg1", "arg2")
              args
            end

            # Capture output using original capture3 for verification
            actual_stdout, actual_stderr, actual_status = capture3_original.call(*capture_args)
            # Still call the original system method
            result = original_method.call(receiver, *args)
            result
          end
        end

        yield

        # Compare outputs
        actual_exit_status = actual_status&.exitstatus
        verified =
          command.stdout == actual_stdout &&
          command.stderr == actual_stderr &&
          command.status == actual_exit_status

        VerifyResult.new(
          verified: verified,
          record_path: Pathname.new(record_path),
          expected_output: command.stdout,
          actual_output: actual_stdout,
          expected_stderr: command.stderr,
          actual_stderr: actual_stderr,
          expected_status: command.status,
          actual_status: actual_exit_status
        )
      end
    end

    def verify!(record_name, mode: :strict, matcher: nil, &block)
      result = verify(record_name, mode: mode, matcher: matcher, &block)

      unless result.verified?
        error_message = "Backspin verification failed!\n"
        error_message += "Record: #{result.record_path}\n"
        error_message += "Expected output:\n#{result.expected_output}\n"
        error_message += "Actual output:\n#{result.actual_output}\n"

        if result.diff && !result.diff.empty?
          error_message += "Diff:\n#{result.diff}\n"
        end

        if result.stderr_diff && !result.stderr_diff.empty?
          error_message += "Stderr diff:\n#{result.stderr_diff}\n"
        end

        # Raise RSpec's expectation failure for proper integration
        raise RSpec::Expectations::ExpectationNotMetError, error_message
      end

      result
    end

    private

    def replay_record(record_path, &block)
      record = Record.load_or_create(record_path)
      unless record.exists?
        raise RecordNotFoundError, "Record not found: #{record_path}"
      end

      if record.empty?
        raise RecordNotFoundError, "No commands found in record"
      end

      # Use RSpec to stub Open3.capture3 to return recorded data
      allow(Open3).to receive(:capture3) do |*args|
        command = record.next_command

        # Make sure this is a capture3 command
        unless command.class == Open3::Capture3
          raise RecordNotFoundError, "Expected Open3::Capture3 command but got #{command.class.name}"
        end

        recorded_stdout = command.stdout
        recorded_stderr = command.stderr
        recorded_status = OpenStruct.new(exitstatus: command.status)

        [recorded_stdout, recorded_stderr, recorded_status]
      rescue NoMoreRecordingsError => e
        raise RecordNotFoundError, e.message
      end

      # Also stub system calls
      allow_any_instance_of(Object).to receive(:system) do |receiver, *args|
        command = record.next_command

        # Make sure this is a system command
        unless command.class == Kernel::System
          raise RecordNotFoundError, "Expected Kernel::System command but got #{command.class.name}"
        end

        # Return true if exit status was 0, false otherwise
        command.status == 0
      rescue NoMoreRecordingsError => e
        raise RecordNotFoundError, e.message
      end

      block_return_value = yield

      # Return stdout, stderr, status if the block returned capture3 results
      # Otherwise return the block's return value
      if block_return_value.is_a?(Array) && block_return_value.size == 3 &&
          block_return_value[0].is_a?(String) && block_return_value[1].is_a?(String)
        # Convert status to integer for consistency
        stdout, stderr, status = block_return_value
        status_int = status.respond_to?(:exitstatus) ? status.exitstatus : status
        [stdout, stderr, status_int]
      else
        block_return_value
      end
    end

    def record_and_save_record(record_path, filter: nil, &block)
      commands = []
      capture3_original = Open3.method(:capture3)

      # Use RSpec's and_wrap_original to record and save
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
        commands << command

        # Return original result
        [stdout, stderr, status]
      end

      # Also intercept system calls
      system_intercepted = false
      allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
        system_intercepted = true

        # Handle different forms of system calls
        # When called as system("echo hello"), receiver is the command string and args is empty
        # When called as system("echo", "hello"), receiver is the test object and args contains the command
        if args.empty? && receiver.is_a?(String)
          # Single string form: system("command and args")
          command_string = receiver
          capture_args = [command_string]
        else
          # Multi-arg form: system("command", "arg1", "arg2")
          capture_args = args
        end

        # For system calls, we capture output by using the original capture3
        # directly to avoid triggering our stub
        stdout, stderr, status = capture3_original.call(*capture_args)

        # Execute the real system call to get the correct return value
        result = original_method.call(receiver, *args)

        # Parse command args based on how system was called
        cmd_args = if args.empty? && receiver.is_a?(String)
          # Single string form - split the command string
          receiver.split(" ")
        elsif capture_args.length == 1 && capture_args.first.is_a?(String)
          # Single string passed to capture3
          capture_args.first.split(" ")
        else
          # Multi-arg form
          capture_args
        end

        # Create command with interaction data
        command = Command.new(
          method_class: Kernel::System,
          args: cmd_args,
          stdout: stdout,
          stderr: stderr,
          status: status.exitstatus,
          recorded_at: Time.now.iso8601
        )
        commands << command

        # Return original result (true/false)
        result
      end

      block_return_value = yield

      # Save commands using new format
      FileUtils.mkdir_p(File.dirname(record_path))
      # Don't load existing data when creating new record
      record = Record.new(record_path)
      record.clear  # Clear any loaded data
      commands.each { |cmd| record.add_command(cmd) }
      record.save(filter: filter)

      # Return appropriate value
      if block_return_value.is_a?(Array) && block_return_value.size == 3
        # Return stdout, stderr, status as integers
        stdout, stderr, status = block_return_value
        [stdout, stderr, status.respond_to?(:exitstatus) ? status.exitstatus : status]
      else
        block_return_value
      end
    end

    def record_new_episode(record_path, filter: nil, &block)
      # For new_episodes mode, we'd need to track which commands have been seen
      # For now, simplified implementation that just appends
      record = Record.load_or_create(record_path)
      new_commands = []
      capture3_original = Open3.method(:capture3)

      # Use RSpec's and_wrap_original to record new episodes
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
        new_commands << command

        # Return original result
        [stdout, stderr, status]
      end

      # Also intercept system calls
      system_intercepted = false
      allow_any_instance_of(Object).to receive(:system).and_wrap_original do |original_method, receiver, *args|
        system_intercepted = true

        # Handle different forms of system calls
        # When called as system("echo hello"), receiver is the command string and args is empty
        # When called as system("echo", "hello"), receiver is the test object and args contains the command
        if args.empty? && receiver.is_a?(String)
          # Single string form: system("command and args")
          command_string = receiver
          capture_args = [command_string]
        else
          # Multi-arg form: system("command", "arg1", "arg2")
          capture_args = args
        end

        # For system calls, we capture output by using the original capture3
        # directly to avoid triggering our stub
        stdout, stderr, status = capture3_original.call(*capture_args)

        # Execute the real system call to get the correct return value
        result = original_method.call(receiver, *args)

        # Parse command args based on how system was called
        cmd_args = if args.empty? && receiver.is_a?(String)
          # Single string form - split the command string
          receiver.split(" ")
        elsif capture_args.length == 1 && capture_args.first.is_a?(String)
          # Single string passed to capture3
          capture_args.first.split(" ")
        else
          # Multi-arg form
          capture_args
        end

        # Create command with interaction data
        command = Command.new(
          method_class: Kernel::System,
          args: cmd_args,
          stdout: stdout,
          stderr: stderr,
          status: status.exitstatus,
          recorded_at: Time.now.iso8601
        )
        new_commands << command

        # Return original result (true/false)
        result
      end

      result = yield

      # Save all recordings (existing + new)
      if new_commands.any?
        new_commands.each { |cmd| record.add_command(cmd) }
        record.save(filter: filter)
      end

      # Return appropriate value
      if result.is_a?(Array) && result.size == 3
        stdout, stderr, status = result
        [stdout, stderr, status.respond_to?(:exitstatus) ? status.exitstatus : status]
      else
        result
      end
    end

    def build_record_path(name)
      backspin_dir = configuration.backspin_dir
      backspin_dir.mkpath

      # For verify, we still support auto-naming
      if name.nil? || name.empty?
        name = RSpecMetadata.record_name_from_example
      end

      File.join(backspin_dir, "#{name}.yaml")
    end
  end
end

# Define the Open3::Capture3 class for identification
module Open3
  class Capture3; end
end

# Define the Kernel::System class for identification
module Kernel
  class System; end
end
