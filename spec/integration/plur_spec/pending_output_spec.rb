require "spec_helper"

RSpec.describe "pending specs output" do
  def fixture_path(name)
    project_fixture(name)
  end

  def run_plur(file_or_glob, *args)
    cmd_array = [plur_binary, file_or_glob]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  # Replace timing information to make output deterministic
  def normalize_timing(str)
    str
      .gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
        "Finished in [TIME] seconds (files took [TIME] seconds to load)")
      .gsub(/plur version version=[\w.-]+\n/, "")
      .gsub(/Running \d+ specs? \[rspec\] in parallel using \d+ workers?\n/, "")
  end

  def normalize_pending_output_snapshot(snapshot)
    snapshot.merge(
      "args" => ["[OUTPUT_COMPARISON_COMMAND]"],
      "stdout" => normalize_timing(snapshot.fetch("stdout", "")).strip,
      "stderr" => ""
    )
  end

  describe "pending section output" do
    it "shows pending section before failures like RSpec" do
      # Record rspec output as baseline
      chdir fixture_path("failing_specs") do
        Backspin.run(
          ["bundle", "exec", "rspec", "spec/mixed_results_spec.rb", "--force-color"],
          name: "pending_output_comparison",
          filter: ->(snapshot) { normalize_pending_output_snapshot(snapshot) }
        )
      end

      # Verify plur output matches
      result = chdir fixture_path("failing_specs") do
        Backspin.run(
          [plur_binary, "spec/mixed_results_spec.rb"],
          name: "pending_output_comparison",
          filter: ->(snapshot) { normalize_pending_output_snapshot(snapshot) }
        )
      end

      # Verify output structure
      expect(result.actual.stdout).to include("Pending:")
      expect(result.actual.stdout).to include("Failures listed here are expected")
      expect(result.actual.stdout).to include("Failures:")

      # Verify pending appears before failures
      pending_pos = result.actual.stdout.index("Pending:")
      failures_pos = result.actual.stdout.index("Failures:")
      expect(pending_pos).to be < failures_pos
    end

    it "includes pending messages and reasons" do
      chdir fixture_path("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb")

        expect(stdout).to include("# Not implemented yet")
        expect(stdout).to include("# Temporarily skipped with xit")
        expect(stdout).to include("# Waiting for feature")
      end
    end
  end

  describe "pending progress indicators" do
    it "shows yellow * for pending specs with color" do
      chdir fixture_path("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb")

        # Yellow * for pending specs
        expect(stdout).to include("\e[33m*\e[0m")
      end
    end
  end

  describe "pending count in summary" do
    it "shows N pending in summary line" do
      chdir fixture_path("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb")

        # mixed_results_spec.rb has 3 pending specs
        expect(stdout).to match(/\d+ examples, \d+ failures?, 3 pending/)
      end
    end
  end
end
