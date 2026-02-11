require "spec_helper"

RSpec.describe "RSpec CLI args" do
  def normalize_dry_run_output(output)
    output
      .gsub(/^plur version version=.*$/, "plur version version=[VERSION]")
      .gsub(%r{-r\s+\S+/formatter/json_rows_formatter\.rb}, "-r [FORMATTER_PATH]")
  end

  def normalize_dry_run_snapshot(snapshot)
    snapshot.merge("stderr" => normalize_dry_run_output(snapshot.fetch("stderr", "")))
  end

  context "with explicit --tag" do
    it "places tag args before file arguments" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "--tag=slow", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run] Running 1 spec [rspec]")
        worker_line = result.err.lines.find { |line| line.include?("[dry-run] Worker 0:") }
        expect(worker_line).to include("--tag slow")
        expect(worker_line.index("--tag slow")).to be < worker_line.index("spec/calculator_spec.rb")
      end
    end
  end

  context "with passthrough via --" do
    it "places passthrough args before file arguments" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/calculator_spec.rb", "--", "--seed", "1234", "--backtrace")

        expect(result.err).to include("[dry-run] Running 1 spec [rspec]")
        worker_line = result.err.lines.find { |line| line.include?("[dry-run] Worker 0:") }
        expect(worker_line).to include("--seed 1234")
        expect(worker_line).to include("--backtrace")
        expect(worker_line.index("--seed 1234")).to be < worker_line.index("spec/calculator_spec.rb")
      end
    end
  end

  context "with passthrough format options" do
    it "keeps plur formatter and includes user formatter/output args" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/calculator_spec.rb", "--", "--format", "documentation", "--out", "tmp/rspec.out")

        worker_line = result.err.lines.find { |line| line.include?("[dry-run] Worker 0:") }
        expect(worker_line).to include("--format Plur::JsonRowsFormatter")
        expect(worker_line).to include("--format documentation")
        expect(worker_line).to include("--out tmp/rspec.out")
        expect(worker_line.index("--format Plur::JsonRowsFormatter")).to be < worker_line.index("spec/calculator_spec.rb")
        expect(worker_line.index("--format documentation")).to be < worker_line.index("spec/calculator_spec.rb")
      end
    end
  end

  context "with explicit --tag on non-rspec framework" do
    it "errors with a clear message" do
      Dir.chdir(project_fixture("minitest-success")) do
        result = run_plur_allowing_errors("--dry-run", "--use=minitest", "--tag=slow")

        expect(result.exit_status).not_to eq(0)
        expect(result.err).to include("--tag is only supported for rspec")
        expect(result.err).to include("minitest")
      end
    end
  end

  context "snapshot coverage" do
    it "captures dry-run passthrough formatter output with Backspin" do
      command = [
        "plur", "--dry-run", "spec/calculator_spec.rb",
        "--", "--format", "documentation", "--out", "tmp/rspec.out"
      ]

      Dir.chdir(default_ruby_dir) do
        result = Backspin.run(
          command,
          name: "rspec_args_passthrough_formatter_dry_run",
          filter: ->(snapshot) { normalize_dry_run_snapshot(snapshot) }
        )

        expect(result.actual.stderr).to include("--format documentation")
        expect(result.actual.stderr).to include("--out tmp/rspec.out")
      end
    end
  end
end
