require "spec_helper"

# Guards the aggregate-failure numbering fix against a realistic mixed file
# (passing examples + plain failures + an aggregate failure). RSpec numbers the
# aggregate's sub-failures relative to their parent (e.g. "2.1)", "2.2)"). Each
# plur worker emits the "‽" placeholder (it can't know the global failure
# number) and plur renumbers the aggregated output afterwards; a regression
# leaks the raw "‽" or mis-numbers failures around the aggregate.
RSpec.describe "aggregate failure golden test" do
  def normalize_snapshot(snapshot)
    stdout = snapshot.fetch("stdout", "").gsub(
      /Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [fake-time] seconds (files took [fake-time] seconds to load)"
    )
    # plur writes a version/worker banner to stderr; rspec writes nothing there
    snapshot.merge("stdout" => stdout.strip, "stderr" => "")
  end

  it "numbers a mixed pass/fail/aggregate run the same as rspec (no ‽ placeholder leak)" do
    result = chdir project_fixture("failing_specs") do
      Backspin.compare(
        reference: ["bundle", "exec", "rspec", "spec/aggregate_failure_spec.rb", "--force-color"],
        actual: [plur_binary, "--color=always", "spec/aggregate_failure_spec.rb"],
        filter: ->(snapshot) { normalize_snapshot(snapshot) }
      )
    end

    # Concretely: the aggregate is the 2nd failure, so its sub-failures inherit
    # "2.x", and the top-level counter continues to "3" for the plain failure
    # that follows it — with no leaked placeholder anywhere.
    expect(result.actual.stdout).to include("2.1)")
    expect(result.actual.stdout).to include("2.2)")
    expect(result.actual.stdout).to include("2.3)")
    expect(result.actual.stdout).to include("3) Order accepts only numeric coupon codes")
    expect(result.actual.stdout).not_to include("‽")
  end
end
