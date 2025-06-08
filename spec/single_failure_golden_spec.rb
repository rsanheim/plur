require "spec_helper"

RSpec.describe "single failure golden test" do
  def fixture_path(name)
    File.join(__dir__, "..", "test_fixtures", name)
  end

  def run_rux(file_or_glob, *args)
    cmd_array = %W[#{rux_binary} #{file_or_glob}]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  def run_rspec(file_or_glob, *args)
    cmd_array = %W[bundle exec rspec #{file_or_glob}]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  # Replace timing information to make output deterministic
  def make_summary_line_consistent(str)
    str.gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)")
  end

  it "shows filtered backtrace same as rspec using Backspin" do
    stdout_matcher = ->(stdout_1, stdout_2) {
      normalized_1 = make_summary_line_consistent(stdout_1)
      normalized_2 = make_summary_line_consistent(stdout_2)

      # Skip rux preamble if present
      if normalized_2.include?("rux version")
        lines = normalized_2.lines
        normalized_2 = lines[2..].join if lines.size > 2
      end

      normalized_1.strip == normalized_2.strip
    }

    stderr_matcher = ->(stderr_1, stderr_2) {
      # rspec has empty stderr, rux has runtime info - both are valid
      true
    }

    # Record rspec output
    Backspin.run("rspec_vs_rux_backtrace_comparison",
      match_on: [[:stdout, stdout_matcher], [:stderr, stderr_matcher]]) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--tty", "--force-color")
      end
    end

    # Verify rux output matches
    result = Backspin.run!("rspec_vs_rux_backtrace_comparison",
      mode: :auto,
      match_on: [[:stdout, stdout_matcher], [:stderr, stderr_matcher]]) do
      chdir fixture_path("failing_specs") do
        run_rux("spec/single_failure_spec.rb")
      end
    end

    # Verify outputs contain expected elements (with color codes)
    expect(result.stdout).to include("\e[31m") # Red color for failures
    expect(result.stdout).to include("\e[32m") # Green color for syntax highlighting
    expect(result.stdout).to include("\e[36m") # Cyan color for file paths
    expect(result.stdout).to include("./spec/single_failure_spec.rb:6")
    expect(result.stdout).to include("Single Failure fails due to strings not matching")
  end

  it "matches rspec colorized output using Backspin verification" do
    stdout_matcher = ->(stdout_1, stdout_2) {
      normalized_1 = make_summary_line_consistent(stdout_1)
      normalized_2 = make_summary_line_consistent(stdout_2)

      # Skip rux preamble if present
      if normalized_2.include?("rux version")
        lines = normalized_2.lines
        normalized_2 = lines[2..].join if lines.size > 2
      end

      normalized_1.strip == normalized_2.strip
    }

    stderr_matcher = ->(stderr_1, stderr_2) {
      # rspec has empty stderr, rux has runtime info - both are valid
      true
    }

    # Record rspec output
    Backspin.run("rspec_vs_rux_colorized_comparison",
      match_on: [[:stdout, stdout_matcher], [:stderr, stderr_matcher]]) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color", "--tty")
      end
    end

    # Verify rux output matches
    result = Backspin.run!("rspec_vs_rux_colorized_comparison",
      mode: :auto,
      match_on: [[:stdout, stdout_matcher], [:stderr, stderr_matcher]]) do
      chdir fixture_path("failing_specs") do
        run_rux("spec/single_failure_spec.rb")
      end
    end

    # Verify color codes are preserved
    expect(result.stdout).to include("\e[31m") # Red
    expect(result.stdout).to include("\e[32m") # Green
    expect(result.stdout).to include("\e[36m") # Cyan
  end

  it "demonstrates Backspin playback mode for fast tests" do
    stdout_matcher = ->(stdout_1, stdout_2) {
      make_summary_line_consistent(stdout_1) == make_summary_line_consistent(stdout_2)
    }

    # Ensure recording exists
    Backspin.run("rspec_playback_demo",
      match_on: [:stdout, stdout_matcher]) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color")
      end
    end

    # Playback is instant - no actual command execution
    start_time = Time.now
    result = Backspin.run("rspec_playback_demo", mode: :playback) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color")
      end
    end

    expect(Time.now - start_time).to be < 0.1
    expect(result.stdout).to include("Single Failure")
  end
end
