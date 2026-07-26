require "spec_helper"

RSpec.describe "single failure golden test" do
  def fixture_path(name)
    project_fixture(name)
  end

  def rspec_single_failure_command
    ["bundle", "exec", "rspec", "spec/single_failure_spec.rb", "--force-color"]
  end

  def plur_single_failure_command
    [plur_binary, "--color=always", "spec/single_failure_spec.rb"]
  end

  # Replace timing information to make output deterministic
  def make_summary_line_consistent(str)
    str.gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)")
  end

  def normalize_single_failure_snapshot(snapshot)
    snapshot.merge(
      "stdout" => make_summary_line_consistent(snapshot.fetch("stdout", "")).strip,
      "stderr" => ""
    )
  end

  it "matches rspec's colorized failure output and filtered backtrace" do
    result = chdir fixture_path("failing_specs") do
      Backspin.compare(
        reference: rspec_single_failure_command,
        actual: plur_single_failure_command,
        filter: ->(snapshot) { normalize_single_failure_snapshot(snapshot) }
      )
    end

    expect(result.actual.stdout).to include("\e[31m") # Red color for failures
    expect(result.actual.stdout).to include("\e[32m") # Green color for syntax highlighting
    expect(result.actual.stdout).to include("\e[36m") # Cyan color for file paths
    expect(result.actual.stdout).to include("./spec/single_failure_spec.rb:6")
    expect(result.actual.stdout).to include("Single Failure fails due to strings not matching")
  end
end
