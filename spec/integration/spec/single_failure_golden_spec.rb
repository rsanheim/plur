require "spec_helper"

RSpec.describe "single failure golden test" do
  def normalize_single_failure_snapshot(snapshot)
    stdout = snapshot.fetch("stdout", "").gsub(
      /Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)"
    )
    # plur writes a version/worker banner to stderr; rspec writes nothing there
    snapshot.merge("stdout" => stdout.strip, "stderr" => "")
  end

  it "matches rspec's colorized failure output and filtered backtrace" do
    result = chdir project_fixture("failing_specs") do
      Backspin.compare(
        reference: ["bundle", "exec", "rspec", "spec/single_failure_spec.rb", "--force-color"],
        actual: [plur_binary, "--color=always", "spec/single_failure_spec.rb"],
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
