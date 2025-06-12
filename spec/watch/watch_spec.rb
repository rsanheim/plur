require "spec_helper"

RSpec.describe "rux watch command" do
  include RuxWatchHelper
  context "basic functionality" do
    it "starts successfully when spec directory exists" do
      result = run_rux_watch(timeout: 2)

      expect(result.err).to include("rux watch starting!")
      expect(result.err).to include("directories=[spec lib]")
      expect(result.err).to include("timeout=2")
      expect(result.out).to include("Timeout reached, exiting!")
      expect(result.success?).to be true
    end

    it "shows watcher availability on supported platforms" do
      result = run_rux_watch

      if RUBY_PLATFORM.include?("darwin") && RUBY_PLATFORM.include?("arm64")
        expect(result.err).to include("rux using e-dant/watcher")
        expect(result.err).to include("path=/Users/rsanheim/.cache/rux/bin/watcher-aarch64-apple-darwin")
      end
    end

    it "fails gracefully when spec directory doesn't exist" do
      Dir.mktmpdir do |tmpdir|
        result = run_rux_watch(dir: tmpdir, timeout: 1)

        expect(result.success?).to be false
        expect(result.err).to match(/no directories to watch found/i)
        expect(result.err).to include("spec")
      end
    end
  end

  context "file change detection" do
    it "detects and runs spec when file is modified" do
      result = run_rux_watch(timeout: 2) do
        # Write back the same file to trigger a modify event
        spec_file = Pathname.new(default_ruby_dir).join("spec", "calculator_spec.rb")
        spec_file.write(spec_file.read)
      end

      # Debug output if test fails
      if ENV["DEBUG"]
        puts "=== WATCH OUTPUT ==="
        puts result.out
        puts "=== STDERR ==="
        puts result.err
        puts "=================="
      end

      expect(result.out).to include("running:")
      expect(result.out).to include(/bundle exec rspec/)
      expect(result.out).to include("calculator_spec.rb")
      expect(result.success?).to be true
    end
  end

  context "debouncing" do
    it "uses default debounce of 100ms when not specified" do
      result = run_rux_watch(timeout: 1)
      expect(result.err).to include("Debounce delay ms=100")
    end

    it "respects custom debounce delay" do
      result = run_rux_watch(timeout: 1, debounce: 250)
      expect(result.err).to include("Debounce delay ms=250")
    end
  end
end
