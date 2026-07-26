require "spec_helper"

RSpec.describe "pending specs output" do
  def run_plur(file_or_glob, *args)
    cmd_array = [plur_binary, file_or_glob]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  def normalize_pending_output_snapshot(snapshot)
    stdout = snapshot.fetch("stdout", "").gsub(
      /Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/,
      "Finished in [TIME] seconds (files took [TIME] seconds to load)"
    )
    # plur writes a version/worker banner to stderr; rspec writes nothing there
    snapshot.merge("stdout" => stdout.strip, "stderr" => "")
  end

  describe "pending section output" do
    it "shows pending section before failures like RSpec" do
      # --color=always for parity with rspec's --force-color
      result = chdir project_fixture("failing_specs") do
        Backspin.compare(
          reference: ["bundle", "exec", "rspec", "spec/mixed_results_spec.rb", "--force-color"],
          actual: [plur_binary, "--color=always", "spec/mixed_results_spec.rb"],
          filter: ->(snapshot) { normalize_pending_output_snapshot(snapshot) }
        )
      end

      expect(result.actual.stdout).to include("Pending:")
      expect(result.actual.stdout).to include("Failures listed here are expected")
      expect(result.actual.stdout).to include("Failures:")

      # Verify pending appears before failures
      pending_pos = result.actual.stdout.index("Pending:")
      failures_pos = result.actual.stdout.index("Failures:")
      expect(pending_pos).to be < failures_pos
    end

    it "includes pending messages and reasons" do
      chdir project_fixture("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb")

        expect(stdout).to include("# Not implemented yet")
        expect(stdout).to include("# Temporarily skipped with xit")
        expect(stdout).to include("# Waiting for feature")
      end
    end
  end

  describe "pending progress indicators" do
    it "shows yellow * for pending specs with color" do
      chdir project_fixture("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb", "--color=always")

        # Yellow * for pending specs
        expect(stdout).to include("\e[33m*\e[0m")
      end
    end
  end

  describe "pending count in summary" do
    it "shows N pending in summary line" do
      chdir project_fixture("failing_specs") do
        stdout, _stderr, _status = run_plur("spec/mixed_results_spec.rb")

        # mixed_results_spec.rb has 3 pending specs
        expect(stdout).to match(/\d+ examples, \d+ failures?, 3 pending/)
      end
    end
  end
end
