require "spec_helper"

RSpec.describe "turbo_tests migration: tag filtering with directory args" do
  def normalize_turbo_dry_run_output(output)
    output
      .gsub(/^plur version version=.*$/, "plur version version=[VERSION]")
      .gsub(%r{-r\s+\S+/formatter/json_rows_formatter\.rb}, "-r [FORMATTER_PATH]")
  end

  context "dry-run command construction" do
    it "places --tag before file args and expands directory to spec files" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "-n", "8", "--tag=~type:system", "spec/models")

        expect(result.err).to include("[dry-run] Running 2 specs [rspec]")
        worker_line = result.err.lines.find { |line| line.include?("[dry-run] Worker") }
        expect(worker_line).to include("--tag ~type:system")
        expect(worker_line).to include("spec/models/")
        expect(worker_line.index("--tag ~type:system")).to be < worker_line.index("spec/models/")
      end
    end

    it "caps workers to file count when -n exceeds available files" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "-n", "8", "--tag=~type:system", "spec/models")

        expect(result.err).to include("using 2 workers")
      end
    end
  end

  context "actual test execution" do
    it "excludes system-tagged specs with --tag=~type:system" do
      result = run_plur("-C", default_ruby_dir, "-n", "2", "--tag=~type:system", "spec/models")

      expect(result.out).to include("3 examples, 0 failures")
    end

    it "runs all specs in directory without tag filter (baseline)" do
      result = run_plur("-C", default_ruby_dir, "-n", "2", "spec/models")

      expect(result.out).to include("5 examples, 0 failures")
    end

    it "includes only system-tagged specs with --tag=type:system" do
      result = run_plur("-C", default_ruby_dir, "-n", "2", "--tag=type:system", "spec/models")

      expect(result.out).to include("2 examples, 0 failures")
    end
  end

  context "snapshot coverage" do
    it "captures dry-run command output for turbo_tests-style tag filtering" do
      stderr_matcher = ->(recorded, actual) {
        normalize_turbo_dry_run_output(recorded) == normalize_turbo_dry_run_output(actual)
      }

      Dir.chdir(default_ruby_dir) do
        command = ["plur", "--dry-run", "-n", "8", "--tag=~type:system", "spec/models"]
        result = Backspin.run(
          command,
          name: "turbo_tests_tag_filtering_dry_run",
          matcher: {stderr: stderr_matcher}
        )

        expect(result.actual.stderr).to include("--tag ~type:system")
        expect(result.actual.stderr).to include("spec/models/")
      end
    end
  end
end
