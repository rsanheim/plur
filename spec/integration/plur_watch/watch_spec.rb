require "spec_helper"

RSpec.describe "plur watch command", :skip_if_ci do
  include PlurWatchHelper

  context "basic functionality" do
    it "starts successfully when spec directory exists" do
      result = run_plur_watch(timeout: 1)

      expect(result.err).to include("plur watch starting!")
      expect(result.err).to include("directories=[lib spec]")
      expect(result.err).to include("timeout=1")
      expect(result.out).to include("Timeout reached, exiting!")
      expect(result.success?).to be true
    end

    it "shows watcher availability on supported platforms" do
      result = run_plur_watch(timeout: 1)

      if RUBY_PLATFORM.include?("darwin") && RUBY_PLATFORM.include?("arm64")
        expect(result.err).to include("e-dant/watcher installed")
        expect(result.err).to include("path=#{ENV["HOME"]}/.plur/bin/watcher-aarch64-apple-darwin")
      end
    end
  end

  context "file change detection" do
    it "detects when files are modified" do
      result = run_plur_watch(timeout: 2) do
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

      # Watch now maps file changes to jobs and executes them
      expect(result.err).to include("File changed")
      expect(result.err).to include("calculator_spec.rb")
      expect(result.success?).to be true
    end
  end

  context "debouncing" do
    it "uses default debounce of 100ms when not specified" do
      result = run_plur_watch(timeout: 1)
      expect(result.err).to include("Debounce delay ms=100")
    end

    it "respects custom debounce delay" do
      result = run_plur_watch(timeout: 1, debounce: 250)
      expect(result.err).to include("Debounce delay ms=250")
    end
  end
end
