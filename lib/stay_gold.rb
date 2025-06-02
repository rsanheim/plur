require 'yaml'
require 'fileutils'
require 'open3'
require 'pathname'
require 'ostruct'
require 'rspec/mocks'

module StayGold
  class CassetteNotFoundError < StandardError; end
  
  # Include RSpec mocks methods
  extend RSpec::Mocks::ExampleMethods
  
  class Result
    attr_reader :commands, :cassette_path

    def initialize(commands:, cassette_path:)
      @commands = commands
      @cassette_path = cassette_path
    end
  end
  
  class VerifyResult
    attr_reader :cassette_path, :expected_output, :actual_output, :diff, :stderr_diff
    
    def initialize(verified:, cassette_path:, expected_output: nil, actual_output: nil, 
                   expected_stderr: nil, actual_stderr: nil, expected_status: nil, actual_status: nil,
                   command_executed: true)
      @verified = verified
      @cassette_path = cassette_path
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
    def to_h
      {
        "stdout" => @stdout,
        "stderr" => @stderr,
        "status" => @status,
        "recorded_at" => @recorded_at
      }
    end
    
    # Create from hash (for loading from YAML)
    def self.from_h(data)
      new(
        method_class: Open3::Capture3,
        args: [], # Args not stored in legacy format, but not needed for replay
        stdout: data["stdout"],
        stderr: data["stderr"], 
        status: data["status"],
        recorded_at: data["recorded_at"]
      )
    end
  end

  class << self
    attr_accessor :last_output

    def record(record_as: nil)
      commands = []
      cassette_path = build_cassette_path(record_as)
      
      # Use RSpec's and_wrap_original to intercept calls
      allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
        # Execute the real command
        stdout, stderr, status = original_method.call(*args)
        
        # Parse command args
        cmd_args = if args.length == 1 && args.first.is_a?(String)
          args.first.split(' ')
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
        StayGold.last_output = stdout
        
        # Return original result
        [stdout, stderr, status]
      end
      
      yield
      
      # Save commands to cassette as array
      FileUtils.mkdir_p(File.dirname(cassette_path))
      cassette_data = commands.map(&:to_h)
      File.write(cassette_path, cassette_data.to_yaml)
      
      Result.new(commands: commands, cassette_path: Pathname.new(cassette_path))
    end

    def output
      last_output
    end
    
    def use_cassette(cassette_name = nil, options = {}, &block)
      cassette_path = build_cassette_path(cassette_name)
      record_mode = options[:record] || :once
      
      case record_mode
      when :none
        # Never record, only replay
        unless File.exist?(cassette_path)
          raise CassetteNotFoundError, "Cassette not found: #{cassette_path}"
        end
        replay_cassette(cassette_path, &block)
      when :all
        # Always record
        record_and_save_cassette(cassette_path, &block)
      when :once
        # Record if doesn't exist, replay if exists
        if File.exist?(cassette_path)
          replay_cassette(cassette_path, &block)
        else
          record_and_save_cassette(cassette_path, &block)
        end
      when :new_episodes
        # Record new commands not in cassette
        # For now, simplified: just append new recordings
        record_new_episode(cassette_path, &block)
      else
        raise ArgumentError, "Unknown record mode: #{record_mode}"
      end
    end
    
    def verify(cassette: nil, mode: :strict, matcher: nil, &block)
      cassette_path = build_cassette_path(cassette)
      
      unless File.exist?(cassette_path)
        raise CassetteNotFoundError, "Cassette not found: #{cassette_path}"
      end
      
      recordings_data = YAML.load_file(cassette_path)
      
      # Convert hash data to Command objects
      unless recordings_data.is_a?(Array)
        raise CassetteNotFoundError, "Invalid cassette format: expected array"
      end
      
      # For verify, we only handle single command verification for now
      # Use the first command
      if recordings_data.empty?
        raise CassetteNotFoundError, "No commands found in cassette"
      end
      
      command = Command.from_h(recordings_data.first)
      
      if mode == :playback
        # Playback mode: return recorded output without running command
        actual_stdout = command.stdout
        actual_stderr = command.stderr
        actual_status = OpenStruct.new(exitstatus: command.status)
        
        # Use RSpec to stub Open3.capture3
        allow(Open3).to receive(:capture3).and_return([actual_stdout, actual_stderr, actual_status])
        
        yield
        
        # In playback mode, always verified
        VerifyResult.new(
          verified: true,
          cassette_path: Pathname.new(cassette_path),
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
        allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
          stdout, stderr, status = original_method.call(*args)
          actual_data["stdout"] = stdout
          actual_data["stderr"] = stderr
          actual_data["status"] = status.exitstatus
          [stdout, stderr, status]
        end
        
        yield
        
        # Call custom matcher - convert command back to hash format for matcher
        recorded_data = command.to_h
        verified = matcher.call(recorded_data, actual_data)
        
        VerifyResult.new(
          verified: verified,
          cassette_path: Pathname.new(cassette_path),
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
        allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
          actual_stdout, actual_stderr, actual_status = original_method.call(*args)
          [actual_stdout, actual_stderr, actual_status]
        end
        
        yield
        
        # Compare outputs
        actual_exit_status = actual_status ? actual_status.exitstatus : nil
        verified = (
          command.stdout == actual_stdout &&
          command.stderr == actual_stderr &&
          command.status == actual_exit_status
        )
        
        VerifyResult.new(
          verified: verified,
          cassette_path: Pathname.new(cassette_path),
          expected_output: command.stdout,
          actual_output: actual_stdout,
          expected_stderr: command.stderr,
          actual_stderr: actual_stderr,
          expected_status: command.status,
          actual_status: actual_exit_status
        )
      end
    end
    
    def verify!(cassette: nil, mode: :strict, matcher: nil, &block)
      result = verify(cassette: cassette, mode: mode, matcher: matcher, &block)
      
      unless result.verified?
        error_message = "StayGold verification failed!\n"
        error_message += "Cassette: #{result.cassette_path}\n"
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
    
    def replay_cassette(cassette_path, &block)
      recordings_data = YAML.load_file(cassette_path)
      
      # Cassettes are always arrays now
      unless recordings_data.is_a?(Array)
        raise CassetteNotFoundError, "Invalid cassette format: expected array"
      end
      
      # Convert hash data to Command objects
      commands = recordings_data.map { |data| Command.from_h(data) }
      current_command_index = 0
      
      # Use RSpec to stub Open3.capture3 to return recorded data
      allow(Open3).to receive(:capture3) do |*args|
        if current_command_index < commands.size
          command = commands[current_command_index]
          current_command_index += 1
          
          recorded_stdout = command.stdout
          recorded_stderr = command.stderr
          recorded_status = OpenStruct.new(exitstatus: command.status)
          
          [recorded_stdout, recorded_stderr, recorded_status]
        else
          # If we run out of recordings, raise an error
          raise CassetteNotFoundError, "No more recordings available for replay"
        end
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
    
    def record_and_save_cassette(cassette_path, &block)
      commands = []
      
      # Use RSpec's and_wrap_original to record and save
      allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
        # Execute the real command
        stdout, stderr, status = original_method.call(*args)
        
        # Parse command args
        cmd_args = if args.length == 1 && args.first.is_a?(String)
          args.first.split(' ')
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
      
      block_return_value = yield
      
      # Save commands to cassette as array
      FileUtils.mkdir_p(File.dirname(cassette_path))
      cassette_data = commands.map(&:to_h)
      File.write(cassette_path, cassette_data.to_yaml)
      
      # Return appropriate value
      if block_return_value.is_a?(Array) && block_return_value.size == 3
        # Return stdout, stderr, status as integers
        stdout, stderr, status = block_return_value
        [stdout, stderr, status.respond_to?(:exitstatus) ? status.exitstatus : status]
      else
        block_return_value
      end
    end
    
    def record_new_episode(cassette_path, &block)
      # For new_episodes mode, we'd need to track which commands have been seen
      # For now, simplified implementation that just appends
      existing_recordings = if File.exist?(cassette_path)
        YAML.load_file(cassette_path) || []
      else
        []
      end
      
      new_commands = []
      
      # Use RSpec's and_wrap_original to record new episodes
      allow(Open3).to receive(:capture3).and_wrap_original do |original_method, *args|
        # Execute the real command
        stdout, stderr, status = original_method.call(*args)
        
        # Parse command args
        cmd_args = if args.length == 1 && args.first.is_a?(String)
          args.first.split(' ')
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
      
      result = yield
      
      # Save all recordings (existing + new)
      if new_commands.any?
        new_command_data = new_commands.map(&:to_h)
        all_recordings = existing_recordings + new_command_data
        FileUtils.mkdir_p(File.dirname(cassette_path))
        File.write(cassette_path, all_recordings.to_yaml)
      end
      
      # Return appropriate value
      if result.is_a?(Array) && result.size == 3
        stdout, stderr, status = result
        [stdout, stderr, status.respond_to?(:exitstatus) ? status.exitstatus : status]
      else
        result
      end
    end

    def build_cassette_path(name)
      cassette_dir = File.join(Dir.pwd, "tmp", "stay_gold")
      
      if name.nil? || name.empty?
        # Auto-generate name from RSpec metadata
        if defined?(RSpec) && RSpec.respond_to?(:current_example)
          example = RSpec.current_example
          if example
            # Build path from example group hierarchy
            path_parts = []
            
            # Traverse up through parent example groups
            current_group = example.example_group
            while current_group && current_group.respond_to?(:metadata) && current_group.metadata
              description = current_group.metadata[:description]
              path_parts.unshift(sanitize_filename(description)) if description && !description.empty?
              current_group = current_group.superclass if current_group.superclass.respond_to?(:metadata)
            end
            
            # Add the example description
            path_parts << sanitize_filename(example.description)
            
            name = path_parts.join('/')
          end
        end
      end
      
      File.join(cassette_dir, "#{name}.yaml")
    end
    
    def sanitize_filename(str)
      # Replace spaces with underscores and remove special characters
      str.gsub(/\s+/, '_').gsub(/[^a-zA-Z0-9_-]/, '')
    end
  end
end

# Define the Open3::Capture3 class for identification
module Open3
  class Capture3; end
end