require "spec_helper"

RSpec.describe "turbo_tests migration: tag filtering with directory args" do
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
end
