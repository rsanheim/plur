require "spec_helper"

RSpec.describe "RSpec CLI args" do
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
end
