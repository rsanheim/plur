require "spec_helper"

RSpec.describe "turbo_tests migration: tag filtering with directory args" do
  def normalize_turbo_dry_run_output(output)
    lines = output
      .lines
      .map(&:chomp)
      .map do |line|
        line
          .gsub(/^plur version version=.*$/, "plur version version=[VERSION]")
          .gsub(%r{-r\s+\S+/formatter/json_rows_formatter\.rb}, "-r [FORMATTER_PATH]")
          .gsub(/\bTEST_ENV_NUMBER=\d+\s+/, "")
      end

    # Sort worker lines because parallel worker ordering is nondeterministic
    worker_lines = lines.select { |line| line.start_with?("[dry-run] Worker ") }
      .map { |line| line.sub(/^\[dry-run\] Worker \d+:\s+/, "[dry-run] Worker: ") }
      .sort

    non_worker_lines = lines.reject { |line| line.start_with?("[dry-run] Worker ") }

    (non_worker_lines + worker_lines).join("\n")
  end

  def normalize_turbo_dry_run_snapshot(snapshot)
    snapshot.merge("stderr" => normalize_turbo_dry_run_output(snapshot.fetch("stderr", "")))
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
      Dir.chdir(default_ruby_dir) do
        command = ["plur", "--dry-run", "-n", "8", "--tag=~type:system", "spec/models"]
        result = Backspin.run(
          command,
          name: "turbo_tests_tag_filtering_dry_run",
          filter: ->(snapshot) { normalize_turbo_dry_run_snapshot(snapshot) }
        )

        expect(result.actual.stderr).to include("--tag ~type:system")
        expect(result.actual.stderr).to include("spec/models/")
      end
    end
  end
end
