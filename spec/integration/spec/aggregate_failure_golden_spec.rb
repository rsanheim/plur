require "spec_helper"

# Guards the aggregate-failure numbering fix against a realistic mixed file
# (passing examples + plain failures + an aggregate failure). RSpec numbers the
# aggregate's sub-failures relative to their parent (e.g. "2.1)", "2.2)"). Each
# plur worker emits the "‽" placeholder (it can't know the global failure
# number) and plur renumbers the aggregated output afterwards; a regression
# leaks the raw "‽" or mis-numbers failures around the aggregate.
RSpec.describe "aggregate failure golden test" do
  def fixture_path(name)
    project_fixture(name)
  end

  def rspec_command
    ["bundle", "exec", "rspec", "spec/aggregate_failure_spec.rb", "--force-color"]
  end

  def plur_command
    [plur_binary, "--color=always", "spec/aggregate_failure_spec.rb"]
  end

  def make_summary_line_consistent(str)
    str.gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)")
  end

  def normalize_snapshot(snapshot)
    snapshot.merge(
      "args" => ["[AGGREGATE_FAILURE_COMMAND]"],
      "stdout" => make_summary_line_consistent(snapshot.fetch("stdout", "")).strip,
      "stderr" => ""
    )
  end

  it "numbers a mixed pass/fail/aggregate run the same as rspec (no ‽ placeholder leak)" do
    # Record rspec's own output as the baseline
    chdir fixture_path("failing_specs") do
      Backspin.run(
        rspec_command,
        name: "aggregate_failure_comparison",
        filter: ->(snapshot) { normalize_snapshot(snapshot) }
      )
    end

    result = chdir fixture_path("failing_specs") do
      Backspin.run(
        plur_command,
        name: "aggregate_failure_comparison",
        filter: ->(snapshot) { normalize_snapshot(snapshot) }
      )
    end

    # plur output matches rspec byte-for-byte after normalization
    expect(result.verified?).to be(true)

    # And concretely: the aggregate is the 2nd failure, so its sub-failures
    # inherit "2.x", and the top-level counter continues to "3" for the plain
    # failure that follows it — with no leaked placeholder anywhere.
    expect(result.actual.stdout).to include("2.1)")
    expect(result.actual.stdout).to include("2.2)")
    expect(result.actual.stdout).to include("2.3)")
    expect(result.actual.stdout).to include("3) Order accepts only numeric coupon codes")
    expect(result.actual.stdout).not_to include("‽")
  end
end
