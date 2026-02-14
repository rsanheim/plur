require "spec_helper"

RSpec.describe "single failure golden test" do
  def fixture_path(name)
    project_fixture(name)
  end

  def rspec_single_failure_command
    ["bundle", "exec", "rspec", "spec/single_failure_spec.rb", "--force-color"]
  end

  def plur_single_failure_command
    [plur_binary, "spec/single_failure_spec.rb"]
  end

  # Replace timing information to make output deterministic
  def make_summary_line_consistent(str)
    str.gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)")
  end

  def normalize_single_failure_snapshot(snapshot)
    snapshot.merge(
      "args" => ["[SINGLE_FAILURE_COMMAND]"],
      "stdout" => make_summary_line_consistent(snapshot.fetch("stdout", "")).strip,
      "stderr" => ""
    )
  end

  it "shows filtered backtrace same as rspec using Backspin" do
    # Record rspec output
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_vs_plur_backtrace_comparison",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    # Verify output matches
    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_vs_plur_backtrace_comparison",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
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
    # Record rspec output
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_vs_plur_colorized_comparison",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    # Verify output matches
    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_vs_plur_colorized_comparison",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    # Verify color codes are preserved
    expect(result.actual.stdout).to include("\e[31m") # Red
    expect(result.actual.stdout).to include("\e[32m") # Green
    expect(result.actual.stdout).to include("\e[36m") # Cyan
  end

  it "uses Backspin snapshot result fields in verify mode" do
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_single_failure_command,
        name: "rspec_verify_mode_demo",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_single_failure_command,
        name: "rspec_verify_mode_demo",
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    expect(result.verified?).to be(true)
    expect(result.actual.stdout).to include("Single Failure")
    expect(result.expected.stdout).to include("Single Failure")
  end
end
