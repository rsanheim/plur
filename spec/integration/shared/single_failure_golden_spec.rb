require "spec_helper"

RSpec.describe "single failure golden test" do
  def fixture_path(name)
    project_fixture(name)
  end

  def rspec_single_failure_command
    ["bundle", "exec", "rspec", "spec/single_failure_spec.rb", "--force-color"]
  end

  def plur_single_failure_command
    ["plur", "spec/single_failure_spec.rb"]
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
      normalized_1.strip == normalized_2.strip
    }

    stderr_matcher = ->(stderr_1, stderr_2) { true }

    # Record rspec output
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_vs_plur_backtrace_comparison",
        matcher: {stdout: stdout_matcher}
      )
    end

    # Verify output matches
    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_vs_plur_backtrace_comparison",
        mode: :verify,
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}
      )
    end

    # Verify outputs contain expected elements (with color codes)
    expect(result.actual.stdout).to include("\e[31m") # Red color for failures
    expect(result.actual.stdout).to include("\e[32m") # Green color for syntax highlighting
    expect(result.actual.stdout).to include("\e[36m") # Cyan color for file paths
    expect(result.actual.stdout).to include("./spec/single_failure_spec.rb:6")
    expect(result.actual.stdout).to include("Single Failure fails due to strings not matching")
  end

  it "matches rspec colorized output using Backspin verification" do
    stdout_matcher = ->(stdout_1, stdout_2) {
      normalized_1 = make_summary_line_consistent(stdout_1)
      normalized_2 = make_summary_line_consistent(stdout_2)

      # Skip preamble if present
      normalized_1.strip == normalized_2.strip
    }

    stderr_matcher = ->(stderr_1, stderr_2) { true }

    # Record rspec output
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_vs_plur_colorized_comparison",
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}
      )
    end

    # Verify output matches
    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_vs_plur_colorized_comparison",
        mode: :verify,
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}
      )
    end

    # Verify color codes are preserved
    expect(result.actual.stdout).to include("\e[31m") # Red
    expect(result.actual.stdout).to include("\e[32m") # Green
    expect(result.actual.stdout).to include("\e[36m") # Cyan
  end

  it "uses Backspin snapshot result fields in verify mode" do
    stdout_matcher = ->(stdout_1, stdout_2) {
      make_summary_line_consistent(stdout_1) == make_summary_line_consistent(stdout_2)
    }

    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_verify_mode_demo",
        matcher: {stdout: stdout_matcher}
      )
    end

    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_verify_mode_demo",
        mode: :verify,
        matcher: {stdout: stdout_matcher}
      )
    end

    expect(result.verified?).to be(true)
    expect(result.actual.stdout).to include("Single Failure")
    expect(result.expected.stdout).to include("Single Failure")
  end
end
