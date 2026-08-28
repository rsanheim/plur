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
        expect(result.err).to include("path=\"#{ENV["HOME"]}/.plur/bin/watcher-aarch64-apple-darwin\"")
      end
    end
  end

  context "interactive output" do
    it "prints the startup prompt without a trailing newline" do
      result = run_plur_watch_interactive(commands: ["exit"], timeout: 3)

      expect(result.out).to include("exit (Ctrl-C)        Exit watch mode\n\n[plur] > Exiting watch mode...\n")
    end

    it "separates the manual run status, command, and next prompt" do
      with_temp_watch_project do |project|
        project.join(".plur.toml").write(<<~TOML)
          use = "manual"

          [job.manual]
          cmd = ["ruby", "-e", "puts 'manual done'"]

          [[watch]]
          source = "spec/**/*_spec.rb"
          jobs = ["manual"]
        TOML

        requested = false
        exited = false
        result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
          if !requested && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
            process.stdin.puts("")
            requested = true
          elsif requested && !exited && process.out.include?("manual done\n\n[plur] > ")
            process.stdin.puts("exit")
            process.close_stdin
            exited = true
          end
        end

        expect(result.out).to include(
          "[plur] > Running all tests...\n\n[plur] ruby -e puts 'manual done'\n" \
          "manual done\n\n[plur] > Exiting watch mode..."
        )
      end
    end
  end

  context "file change detection" do
    it "detects when files are modified" do
      with_temp_watch_project do |project_dir|
        spec_file = project_dir.join("spec", "calculator_spec.rb")
        original_content = spec_file.read

        result = run_plur_watch(dir: project_dir, timeout: 10, stop_on: "calculator_spec.rb") do
          spec_file.write("# Modified by watch_spec\n#{original_content}")
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
        expect(result.err).to include("calculator_spec.rb")
        expect(result.success?).to be true
      end
    end
  end

  context "debouncing" do
    it "uses default debounce of 30ms when not specified" do
      result = run_plur_watch(timeout: 1)
      expect(result.err).to include("Debounce delay ms=30")
    end

    it "respects custom debounce delay" do
      result = run_plur_watch(timeout: 1, debounce: 250)
      expect(result.err).to include("Debounce delay ms=250")
    end
  end
end
