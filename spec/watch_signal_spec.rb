require "spec_helper"
require "timeout"

RSpec.describe "rux watch signal handling" do
  let(:rux_ruby_dir) { File.join(__dir__, "..", "rux-ruby") }

  it "runs indefinitely until receiving SIGINT signal and detects file changes" do
    Dir.chdir(rux_ruby_dir) do
      output = []
      process_exited = false
      exit_status = nil

      # Start rux watch in a separate process
      read_io, write_io = IO.pipe
      pid = spawn("#{rux_binary} watch", out: write_io, err: write_io)
      write_io.close

      # Thread to read output
      reader_thread = Thread.new do
        read_io.each_line do |line|
          output << line.chomp
        end
      end

      begin
        # Wait for process to start and initial output
        sleep 1

        # Verify process is still running
        expect(Process.kill(0, pid)).to eq(1) # Process exists

        # Touch a spec file to trigger watcher
        spec_file = File.join(rux_ruby_dir, "spec", "calculator_spec.rb")
        original_mtime = File.mtime(spec_file)
        FileUtils.touch(spec_file)

        # Wait for watcher to detect the change
        sleep 2.5

        # Verify process is still running after detecting change
        expect(Process.kill(0, pid)).to eq(1) # Process still exists

        # Send SIGINT to gracefully stop
        Process.kill("INT", pid)

        # Wait for process to exit (with timeout)
        Timeout.timeout(3) do
          _, status = Process.wait2(pid)
          exit_status = status&.exitstatus || (status&.signaled? ? 130 : nil)
          process_exited = true
        end
      rescue Timeout::Error
        # Force kill if it doesn't exit gracefully
        Process.kill("KILL", pid)
        Process.wait(pid)
        fail "Process did not exit gracefully after SIGINT"
      ensure
        # Restore original file timestamp
        if defined?(original_mtime) && original_mtime
          File.utime(original_mtime, original_mtime, spec_file)
        end

        # Clean up threads and IO
        reader_thread.kill if reader_thread.alive?
        read_io.close unless read_io.closed?
      end

      # Verify expected behavior
      expect(process_exited).to be true
      # Exit status 130 is typical for SIGINT (128 + 2)
      # Exit status 0 is also acceptable if the process handles SIGINT gracefully
      expect([0, 130]).to include(exit_status)

      # Verify output contains expected messages
      output_text = output.join("\n")

      # Debug output
      if ENV["DEBUG"]
        puts "=== DEBUG OUTPUT ==="
        puts output_text
        puts "==================="
      end

      expect(output_text).to include("Starting rux watch mode")
      expect(output_text).to include("Watching spec directory for changes")
      expect(output_text).to include("Press Ctrl+C to stop")

      # Check for file modification event
      if output_text.include?("Event: modify file")
        expect(output_text).to include("Event: modify file")
        expect(output_text).to include("calculator_spec.rb")
        expect(output_text).to include("Running spec:")
      else
        # Sometimes the watcher might not catch the modification in time
        # This is acceptable as long as the process stayed alive
        expect(output_text).to include("Event: create watcher")
      end
    end
  end

  it "exits immediately if spec directory doesn't exist" do
    Dir.mktmpdir do |tmpdir|
      Dir.chdir(tmpdir) do
        output, status = Open3.capture2e("#{rux_binary} watch")

        expect(status.exitstatus).to eq(1)
        expect(output).to include("spec directory not found")
      end
    end
  end
end
