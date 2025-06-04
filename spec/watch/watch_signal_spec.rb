require "spec_helper"
require "timeout"

RSpec.describe "rux watch signal handling" do
  it "runs indefinitely until receiving SIGINT signal" do
    # Start rux watch without timeout (runs indefinitely)
    pid = nil
    output = []
    
    Dir.chdir(rux_ruby_dir) do
      # Use IO.popen to get the process PID
      IO.popen("#{rux_binary} watch 2>&1") do |io|
        pid = io.pid
        
        begin
          # Start a thread to read output
          reader_thread = Thread.new do
            io.each_line { |line| output << line.chomp }
          end
          
          # Wait for process to start
          sleep 1
          
          # Verify process is running
          expect(Process.kill(0, pid)).to eq(1)
          
          # Touch a spec file to trigger watcher
          spec_file = File.join(rux_ruby_dir, "spec", "calculator_spec.rb")
          FileUtils.touch(spec_file)
          
          # Wait for watcher to detect the change
          sleep 2
          
          # Send SIGINT to gracefully stop
          Process.kill("INT", pid)
          
          # Wait for process to exit with timeout
          Timeout.timeout(3) do
            Process.wait(pid)
          end
          
          reader_thread.join(0.5)
          reader_thread.kill if reader_thread.alive?
        rescue Timeout::Error
          # Force kill if it doesn't exit gracefully
          Process.kill("KILL", pid)
          Process.wait(pid)
          fail "Process did not exit gracefully after SIGINT"
        end
      end
    end
    
    # Verify output
    output_text = output.join("\n")
    
    expect(output_text).to include("Starting rux watch mode")
    expect(output_text).to include("Press Ctrl+C to stop")
    
    # May or may not detect the file change depending on timing
    if output_text.include?("Changed:")
      expect(output_text).to include("calculator_spec.rb")
    end
  end

  it "exits immediately if spec directory doesn't exist" do
    Dir.mktmpdir do |tmpdir|
      result = run_rux_watch(dir: tmpdir, timeout: 1)
      
      expect(result.success?).to be false
      expect(result.err).to match(/no directories to watch found/i)
    end
  end
end