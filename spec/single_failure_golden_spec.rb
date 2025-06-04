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
    # Record RSpec output as the golden master with normalized times
    Backspin.use_dubplate("rspec_backtrace_golden", record: :once) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--tty", "--force-color")
      end
    end

    # Verify Rux output contains the same backtrace line as RSpec
    Backspin.verify!("rspec_backtrace_golden",
      matcher: ->(recorded, actual) {
        # Normalize timing in both outputs
        recorded_normalized = make_summary_line_consistent(recorded["stdout"])
        actual_normalized = make_summary_line_consistent(actual["stdout"])

        # Both should fail with exit status 1
        return false unless recorded["status"] == 1 && actual["status"] == 1

        # Find the specific backtrace line from RSpec output
        rspec_lines = recorded_normalized.split("\n")
        backtrace_line = rspec_lines.find { |line| line.include?("./spec/single_failure_spec.rb:6") }

        # Verify Rux output contains the same backtrace line
        return false if backtrace_line.nil?
        actual_normalized.include?(backtrace_line)
      }) do
      chdir fixture_path("failing_specs") do
        run_rux("spec/single_failure_spec.rb")
      end
    end
  end

  it "matches rspec colorized output using Backspin verification" do
    # Record RSpec output as the golden master with normalized times
    Backspin.use_dubplate("rspec_colorized_output_golden", record: :once) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color", "--tty")
      end
    end

    # Verify Rux output matches the recorded RSpec output
    Backspin.verify!("rspec_colorized_output_golden",
      matcher: ->(recorded, actual) {
        # Normalize timing information in both outputs
        recorded_normalized = make_summary_line_consistent(recorded["stdout"])
        actual_normalized = make_summary_line_consistent(actual["stdout"])

        # Extract lines without the first 2 preamble lines from Rux
        actual_lines = actual_normalized.lines
        actual_without_preamble = actual_lines[2..] # Skip Rux's worker info

        recorded_lines = recorded_normalized.lines

        # Compare the content (allowing for minor differences)
        actual_without_preamble == recorded_lines
      }) do
      chdir fixture_path("failing_specs") do
        run_rux("spec/single_failure_spec.rb")
      end
    end
  end

  it "demonstrates Backspin playback mode for fast tests" do
    # First run: record the RSpec output (slow)
    Time.now
    Backspin.use_dubplate("rspec_playback_demo", record: :once) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color")
      end
    end
    Time.now

    # Second run: playback from cassette (fast)
    start_time = Time.now
    playback_result = Backspin.verify("rspec_playback_demo",
      mode: :playback) do
      chdir fixture_path("failing_specs") do
        run_rspec("spec/single_failure_spec.rb", "--force-color")
      end
    end
    playback_time = Time.now - start_time

    # Verify playback was much faster
    expect(playback_time).to be < 0.1 # Should be nearly instant
    expect(playback_result.command_executed?).to be false
    expect(playback_result.verified?).to be true
    expect(playback_result.output).to include("Single Failure")
  end
end
