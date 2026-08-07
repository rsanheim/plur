require_relative "../../spec_helper"

# The minitest-5 and minitest-6 fixtures contain identical test files and
# differ only in their minitest pin (~> 5.25 vs ~> 6.0). plur must behave the
# same against both majors. The backspin compares run the same suite twice -
# once under raw minitest, once under plur - and check the invariants that do
# not depend on which runner printed them: every test counted as progress, and
# the runs/assertions/failures/errors summary.
RSpec.describe "Minitest version compatibility" do
  minitest_fixtures = %w[minitest-5 minitest-6]

  before(:all) do
    %w[minitest-5 minitest-6].each do |name|
      chdir(project_fixture!(name)) do
        Bundler.with_unbundled_env do
          system("bundle check", out: File::NULL, err: File::NULL) ||
            system("bundle install", out: File::NULL, err: File::NULL, exception: true)
        end
      end
    end
  end

  # Reduces a run's stdout to what raw minitest and plur must agree on: the
  # number of progress dots and the summary counts line. Timing, seed, and each
  # runner's own chrome are stripped. The fixture tests all pass and their puts
  # text contains no "." characters, so counting dots before the "Finished in"
  # line is exact for both runners.
  def progress_and_counts(snapshot)
    lines = snapshot.fetch("stdout", "").split("\n")
    finished = lines.index { |line| line.start_with?("Finished in") } || lines.size
    dots = lines.first(finished).sum { |line| line.count(".") }
    counts = lines.find { |line| line.match?(/\A\d+ runs,/) }
    snapshot.merge("stdout" => "progress=#{dots}\n#{counts}", "stderr" => "")
  end

  minitest_fixtures.each do |fixture|
    context "with #{fixture}" do
      let(:project_dir) { project_fixture!(fixture) }

      it "runs the suite and counts every test in the progress line" do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            result = run_plur("--use", "minitest", "-n", "1", "--color=never")
            expect(result).to be_success
            expect(result.err).to include("Running 2 tests [minitest]")
            expect(result.out.split("\n").first).to eq("." * 8)
            expect(result.out).to include("8 runs, 23 assertions, 0 failures, 0 errors, 0 skips")
          end
        end
      end

      it "reports the same progress count and summary as running minitest directly" do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            result = Backspin.compare(
              reference: ["bundle", "exec", "ruby", "-Itest",
                "-e", %(require "calculator_test"; require "string_helper_test")],
              actual: [plur_binary, "--use", "minitest", "-n", "1", "--color=never"],
              filter: ->(snapshot) { progress_and_counts(snapshot) }
            )

            expect(result.verified?).to be true
            expect(result.actual.status).to eq(0)
          end
        end
      end

      # Characterizes a known limitation (see #106): stdout written by passing
      # tests is dropped, on both minitest majors. Raw minitest prints these
      # lines interleaved with the dots. When output handling changes, this
      # expectation should flip deliberately.
      it "currently drops test-written stdout on a passing run" do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            result = run_plur("--use", "minitest", "-n", "1", "--color=never")
            expect(result).to be_success
            expect(result.out).not_to include("in test_addition")
            expect(result.out).not_to include("in test_titleize")
          end
        end
      end
    end
  end
end
