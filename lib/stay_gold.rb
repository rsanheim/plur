require 'yaml'
require 'fileutils'
require 'open3'
require 'pathname'
require 'ostruct'

module StayGold
  class CassetteNotFoundError < StandardError; end
  
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
    attr_reader :args

    def initialize(method_class:, args:)
      @method_class = method_class
      @args = args
    end

    def class
      @method_class
    end
  end

  class << self
    attr_accessor :last_output

    def record(record_as: nil)
      commands = []
      cassette_path = build_cassette_path(record_as)
      
      # Temporarily override Open3.capture3
      original_capture3 = Open3.method(:capture3)
      
      Open3.define_singleton_method(:capture3) do |*args|
        stdout, stderr, status = original_capture3.call(*args)
        
        # Parse command args
        cmd_args = if args.length == 1 && args.first.is_a?(String)
          args.first.split(' ')
        else
          args
        end
        
        # Record the command
        command = Command.new(
          method_class: Open3::Capture3,
          args: cmd_args
        )
        commands << command
        
        # Save to cassette
        cassette_data = {
          "stdout" => stdout,
          "stderr" => stderr,
          "status" => status.exitstatus,
          "recorded_at" => Time.now.iso8601
        }
        
        FileUtils.mkdir_p(File.dirname(cassette_path))
        File.write(cassette_path, cassette_data.to_yaml)
        
        # Store output for later access
        StayGold.last_output = stdout
        
        [stdout, stderr, status]
      end
      
      yield
      
      # Restore original method
      Open3.define_singleton_method(:capture3, original_capture3)
      
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
      
      recorded_data = YAML.load_file(cassette_path)
      
      if mode == :playback
        # Playback mode: return recorded output without running command
        actual_stdout = recorded_data["stdout"]
        actual_stderr = recorded_data["stderr"]
        actual_status = OpenStruct.new(exitstatus: recorded_data["status"])
        
        # Override Open3.capture3 to return recorded data
        original_capture3 = Open3.method(:capture3)
        
        Open3.define_singleton_method(:capture3) do |*args|
          [actual_stdout, actual_stderr, actual_status]
        end
        
        yield
        
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
        
        # In playback mode, always verified
        VerifyResult.new(
          verified: true,
          cassette_path: Pathname.new(cassette_path),
          expected_output: recorded_data["stdout"],
          actual_output: actual_stdout,
          expected_stderr: recorded_data["stderr"],
          actual_stderr: actual_stderr,
          expected_status: recorded_data["status"],
          actual_status: recorded_data["status"],
          command_executed: false
        )
      elsif matcher
        # Custom matcher verification
        actual_data = {}
        
        # Temporarily override Open3.capture3
        original_capture3 = Open3.method(:capture3)
        
        Open3.define_singleton_method(:capture3) do |*args|
          stdout, stderr, status = original_capture3.call(*args)
          actual_data["stdout"] = stdout
          actual_data["stderr"] = stderr
          actual_data["status"] = status.exitstatus
          [stdout, stderr, status]
        end
        
        yield
        
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
        
        # Call custom matcher
        verified = matcher.call(recorded_data, actual_data)
        
        VerifyResult.new(
          verified: verified,
          cassette_path: Pathname.new(cassette_path),
          expected_output: recorded_data["stdout"],
          actual_output: actual_data["stdout"],
          expected_stderr: recorded_data["stderr"],
          actual_stderr: actual_data["stderr"],
          expected_status: recorded_data["status"],
          actual_status: actual_data["status"]
        )
      else
        # Default strict mode
        actual_stdout = nil
        actual_stderr = nil
        actual_status = nil
        
        # Temporarily override Open3.capture3
        original_capture3 = Open3.method(:capture3)
        
        Open3.define_singleton_method(:capture3) do |*args|
          actual_stdout, actual_stderr, actual_status = original_capture3.call(*args)
          [actual_stdout, actual_stderr, actual_status]
        end
        
        yield
        
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
        
        # Compare outputs
        actual_exit_status = actual_status ? actual_status.exitstatus : nil
        verified = (
          recorded_data["stdout"] == actual_stdout &&
          recorded_data["stderr"] == actual_stderr &&
          recorded_data["status"] == actual_exit_status
        )
        
        VerifyResult.new(
          verified: verified,
          cassette_path: Pathname.new(cassette_path),
          expected_output: recorded_data["stdout"],
          actual_output: actual_stdout,
          expected_stderr: recorded_data["stderr"],
          actual_stderr: actual_stderr,
          expected_status: recorded_data["status"],
          actual_status: actual_exit_status
        )
      end
    end

    private
    
    def replay_cassette(cassette_path, &block)
      recorded_data = YAML.load_file(cassette_path)
      
      # Support both single recording and array of recordings
      recordings = recorded_data.is_a?(Array) ? recorded_data : [recorded_data]
      current_recording_index = 0
      
      # Store the block's return value
      block_return_value = nil
      
      # Override Open3.capture3 to return recorded data
      original_capture3 = Open3.method(:capture3)
      
      Open3.define_singleton_method(:capture3) do |*args|
        if current_recording_index < recordings.size
          recording = recordings[current_recording_index]
          current_recording_index += 1
          
          stdout = recording["stdout"]
          stderr = recording["stderr"]
          status = OpenStruct.new(exitstatus: recording["status"])
          
          [stdout, stderr, status]
        else
          # If we run out of recordings, call the original
          original_capture3.call(*args)
        end
      end
      
      begin
        # Execute the block and capture its return value
        block_return_value = yield
      ensure
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
      end
      
      # Return stdout, stderr, status if the block returned capture3 results
      # Otherwise return the block's return value
      if block_return_value.is_a?(Array) && block_return_value.size == 3 &&
         block_return_value[0].is_a?(String) && block_return_value[1].is_a?(String)
        # Convert status to integer for consistency
        stdout, stderr, status = block_return_value
        status_int = status.is_a?(OpenStruct) ? status.exitstatus : status.exitstatus
        [stdout, stderr, status_int]
      else
        block_return_value
      end
    end
    
    def record_and_save_cassette(cassette_path, &block)
      stdout = nil
      stderr = nil
      status = nil
      block_return_value = nil
      
      # Override Open3.capture3 to capture output
      original_capture3 = Open3.method(:capture3)
      
      Open3.define_singleton_method(:capture3) do |*args|
        stdout, stderr, status = original_capture3.call(*args)
        [stdout, stderr, status]
      end
      
      begin
        block_return_value = yield
      ensure
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
      end
      
      # Save to cassette if capture3 was called
      if stdout && stderr && status
        cassette_data = {
          "stdout" => stdout,
          "stderr" => stderr,
          "status" => status.exitstatus,
          "recorded_at" => Time.now.iso8601
        }
        
        FileUtils.mkdir_p(File.dirname(cassette_path))
        File.write(cassette_path, cassette_data.to_yaml)
      end
      
      # Return appropriate value
      if block_return_value.is_a?(Array) && block_return_value.size == 3
        # Return stdout, stderr, status as integers
        stdout, stderr, status = block_return_value
        [stdout, stderr, status.exitstatus]
      else
        block_return_value
      end
    end
    
    def record_new_episode(cassette_path, &block)
      # For new_episodes mode, we'd need to track which commands have been seen
      # For now, simplified implementation that just appends
      existing_recordings = if File.exist?(cassette_path)
        data = YAML.load_file(cassette_path)
        data.is_a?(Array) ? data : [data]
      else
        []
      end
      
      new_recordings = []
      
      # Override Open3.capture3
      original_capture3 = Open3.method(:capture3)
      
      Open3.define_singleton_method(:capture3) do |*args|
        stdout, stderr, status = original_capture3.call(*args)
        
        # Record this interaction
        new_recordings << {
          "stdout" => stdout,
          "stderr" => stderr,  
          "status" => status.exitstatus,
          "recorded_at" => Time.now.iso8601
        }
        
        [stdout, stderr, status]
      end
      
      begin
        result = yield
      ensure
        # Restore original method
        Open3.define_singleton_method(:capture3, original_capture3)
      end
      
      # Save all recordings (existing + new)
      if new_recordings.any?
        all_recordings = existing_recordings + new_recordings
        FileUtils.mkdir_p(File.dirname(cassette_path))
        File.write(cassette_path, all_recordings.to_yaml)
      end
      
      # Return appropriate value
      if result.is_a?(Array) && result.size == 3
        stdout, stderr, status = result
        [stdout, stderr, status.exitstatus]
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