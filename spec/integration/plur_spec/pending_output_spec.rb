require "spec_helper"

RSpec.describe "pending specs output" do
  def fixture_path(name)
    project_fixture(name)
  end

  def run_plur(file_or_glob, *args)
    cmd_array = %W[plur #{file_or_glob}]
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

  describe "pending section output" do
    it "shows pending section before failures like RSpec" do
      stdout_matcher = ->(stdout_1, stdout_2) {
        normalized_1 = normalize_timing(stdout_1)
        normalized_2 = normalize_timing(stdout_2)
        normalized_1.strip == normalized_2.strip
      }

      stderr_matcher = ->(_stderr_1, _stderr_2) { true }

      # Record rspec output as baseline
      Backspin.run("pending_output_comparison",
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}) do
        chdir fixture_path("failing_specs") do
          run_rspec("spec/mixed_results_spec.rb", "--force-color")
        end
      end

      # Verify plur output matches
      result = Backspin.run!("pending_output_comparison",
        mode: :auto,
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}) do
        chdir fixture_path("failing_specs") do
          run_plur("spec/mixed_results_spec.rb")
        end
      end

      # Verify output structure
      expect(result.stdout).to include("Pending:")
      expect(result.stdout).to include("Failures listed here are expected")
      expect(result.stdout).to include("Failures:")

      # Verify pending appears before failures
      pending_pos = result.stdout.index("Pending:")
      failures_pos = result.stdout.index("Failures:")
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
