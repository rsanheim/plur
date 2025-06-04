require "spec_helper"
require "tempfile"

RSpec.describe "rux watch command" do
  DEFAULT_RUX_TIMEOUT = 2

  def run_rux_watch_in_dir(dir, rux_timeout: DEFAULT_RUX_TIMEOUT, &block)
    Dir.chdir(dir) do
      cmd = "#{rux_binary} watch --timeout #{rux_timeout}"

      stdout, stderr, status = nil

      # Run the command in a thread so we can do file modifications
      thread = Thread.new do
        stdout, stderr, status = Open3.capture3(cmd)
      end

      # Give watch time to start
      sleep 0.5

      # Perform actions while watch is running
      yield if block_given?

      # Wait for thread to complete (timeout will exit it)
      thread.join

      [stdout, stderr, status]
    end
  end

  context "with rux-ruby test project" do
    it "starts successfully when spec directory exists" do
      stdout, stderr, status = run_rux_watch_in_dir(rux_ruby_dir)

      expect(stderr).to be_empty
      expect(stdout).to include("Starting rux watch mode")
      expect(stdout).to include("Watching directories: spec, lib")
      expect(stdout).to include("Will exit after 2 seconds")
      expect(stdout).to include("Timeout reached, exiting watch mode")
      expect(status.exitstatus).to eq(0)
    end

    it "detects and runs spec when file is modified" do
      stdout, stderr, status = run_rux_watch_in_dir(rux_ruby_dir, rux_timeout: 2) do
        # Wait longer for watcher to fully initialize
        sleep 1

        # Write back the same file to trigger a modify event
        spec_file = Pathname.new(rux_ruby_dir).join("spec", "calculator_spec.rb")
        spec_file.write(spec_file.read)
      end

      # Debug output if test fails
      if ENV["DEBUG"]
        puts "=== WATCH OUTPUT ==="
        puts stdout
        puts "=== STDERR ==="
        puts stderr
        puts "=================="
      end

      expect(stdout).to include("running:")
      expect(stdout).to include(/bundle exec rspec/)
      expect(stdout).to include("calculator_spec.rb")
      expect(status.exitstatus).to eq(0)
    end

    it "creates initial watcher event on startup" do
      stdout, _, status = run_rux_watch_in_dir(rux_ruby_dir)

      puts stdout
      expect(stdout).to match(/watching directories: spec, lib/i)
      expect(status.exitstatus).to eq(0)
    end
  end

  context "error handling" do
    it "fails gracefully when spec directory doesn't exist" do
      Dir.mktmpdir do |tmpdir|
        Dir.chdir(tmpdir) do
          _, stderr, status = Open3.capture3("#{rux_binary} watch --timeout 1")

          expect(status.exitstatus).to eq(1)
          expect(stderr).to match(/no directories to watch found/i)
          expect(stderr).to include("spec")
        end
      end
    end
  end

  context "with backspin golden testing" do
    it "produces consistent watch output" do
      # Skip recording if watcher binary is not available
      skip "Watcher binary not available" unless File.exist?(File.expand_path("~/.cache/rux/bin/watcher-aarch64-apple-darwin"))

      # Use backspin with :once mode - records if file doesn't exist, replays if it does
      Backspin.use_dubplate("rux_watch_golden", record: :once) do
        stdout, _, status = run_rux_watch_in_dir(rux_ruby_dir, rux_timeout: 1)

        # Normalize output for consistent comparison
        normalized_stdout = stdout
          .gsub(/Event: create watcher.*/, "Event: create watcher [PATH]")
          .gsub(/\/Users\/[^\/]+/, "/Users/[USER]")
          .gsub(/Will exit after \d+ seconds/, "Will exit after [N] seconds")
          .gsub(/Absolute watch path:.*/, "Absolute watch path: [PATH]")

        expect(normalized_stdout).to include("Starting rux watch mode")
        expect(normalized_stdout).to include("Watching spec directory for changes")
        expect(normalized_stdout).to include("Event: create watcher [PATH]")
        expect(status.exitstatus).to eq(0)
      end
    end
  end
end
