require_relative "../../spec_helper"

# All minitest-specific run behavior lives here, grouped by aspect; framework
# selection, args, and configuration live with their own specs.
#
# minitest-outcomes is the canonical minitest fixture: every outcome the
# progress line can show (pass, failure, error, skip), stdout written mid-run
# in the styles that stress line handling (interleaved puts, unterminated
# print, multi-line), and a unique greppable token for every case.
#
# 12 tests: 8 pass, 2 fail, 1 error, 1 skip. Tokens written to stdout during
# the run avoid the characters ".FES" so progress characters can be counted
# exactly even when test output interleaves with the progress stream.
#
# minitest-failures is kept for the one thing outcomes cannot express: failures
# in more than one file, so a parallel run has two *failing* workers to
# aggregate.
#
# Determinism: minitest randomizes test order per run, and plur cannot
# currently forward --seed to the multi-file run form, so every expectation
# here holds for ANY order. Test stdout streams live, so progress characters
# and output lines interleave order-dependently; counts are asserted across
# the whole progress region (everything before the failure details/summary)
# rather than on any single line.
RSpec.describe "Minitest integration" do
  before(:all) do
    %w[minitest-outcomes minitest-failures minitest-extra-plugin].each do |fixture|
      chdir(project_fixture!(fixture)) do
        Bundler.with_unbundled_env do
          system("bundle check", out: File::NULL, err: File::NULL) ||
            system("bundle install", out: File::NULL, err: File::NULL, exception: true)
        end
      end
    end
  end

  let(:project_dir) { project_fixture!("minitest-outcomes") }

  # Occurrences of each progress character plur can render, counted across the
  # progress region (everything before failure details and the summary, which
  # legitimately contain F/E/dots). Fixture stdout tokens avoid ".FES*" so
  # interleaved test output cannot skew the counts. Skips show as "*"
  # (matching RSpec pending), never "S".
  def progress_alphabet(out)
    region = out.split("\nFailures:").first.split("\nFinished in").first
    %w[. F E S *].to_h { |char| [char, region.count(char)] }
  end

  # Reduces a run to the facts that must not depend on which runner printed
  # them or what order the tests ran in: the outcome counts (normalized across
  # plur's correct pluralization vs minitest's naive one) and which tests
  # failed or errored.
  def outcome_facts(snapshot)
    stdout = snapshot.fetch("stdout", "")
    counts = stdout.scan(/(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?/).last
    ids = stdout.scan(/[A-Z]\w*Test#test_\w+/).uniq.sort
    snapshot.merge("stdout" => "counts=#{counts&.join(",")}\nids=#{ids.join(",")}", "stderr" => "")
  end

  context "outcomes and progress" do
    it "reports a green run with full progress, live stdout, and RSpec-style duration" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          # No --use: this is the one real (non-dry-run) exercise of framework
          # auto-detection, proven by the [minitest] tag below.
          result = run_plur("-n", "1", "--color=never", "test/passing_test.rb")
          expect(result).to be_success

          expect(result.err).to include("plur version")
          expect(result.err).to include("Running 1 test [minitest]")

          expect(progress_alphabet(result.out)).to eq(
            "." => 5, "F" => 0, "E" => 0, "S" => 0, "*" => 0
          )
          expect(result.out).to include("5 runs, 6 assertions, 0 failures, 0 errors, 0 skips")

          # Duration uses RSpec's "seconds" wording, not minitest's bare "Xs."
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          expect(result.out).not_to match(/Finished in [\d.]+s\./)

          # Test stdout streams live, even on a passing run.
          expect(result.out.scan("OUT_MID_RUN").length).to eq(1)
          expect(result.out.scan("OUT_GLOBAL_IO").length).to eq(1)
        end
      end
    end

    it "sums all-green workers into one passing summary" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "2", "--color=never",
            "test/passing_test.rb", "test/output_test.rb")
          expect(result).to be_success

          # The progress line is not asserted here: output_test's unterminated
          # print glues progress characters order-dependently (see the
          # characterization below). The summed counts and exit are the point.
          expect(result.out).to include("8 runs, 9 assertions, 0 failures, 0 errors, 0 skips")
        end
      end
    end

    it "renders every outcome type with exact progress counts" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never",
            "test/passing_test.rb", "test/outcomes_test.rb", allow_error: true)
          expect(result).to be_failure

          expect(progress_alphabet(result.out)).to eq(
            "." => 5, "F" => 2, "E" => 1, "S" => 0, "*" => 1
          )
          expect(result.out).to include("9 runs, 8 assertions, 2 failures, 1 error, 1 skip")
        end
      end
    end

    it "counts every test even when an unterminated print shares its line" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never", allow_error: true)

          # The unterminated `print` test's output lands on the same physical
          # line as the reporter's structured row; the parser splits them, so
          # neither the progress character nor the partial text is lost.
          expect(progress_alphabet(result.out)).to eq(
            "." => 8, "F" => 2, "E" => 1, "S" => 0, "*" => 1
          )
          expect(result.out).to include("PARTIAL_APARTIAL_B")
          expect(result.out).to include("12 runs, 11 assertions, 2 failures, 1 error, 1 skip")
        end
      end
    end

    it "aggregates outcomes across parallel workers with all stdout visible" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "3", "--color=never", allow_error: true)
          expect(result).to be_failure

          expect(result.out).to include("12 runs, 11 assertions, 2 failures, 1 error, 1 skip")

          # Every worker's stdout streams, passing and failing alike.
          expect(result.out).to include("OUT_B4_KABOOM")
          expect(result.out).to include("OUT_MID_RUN")
          expect(result.out).to include("PARTIAL_A")
        end
      end
    end
  end

  context "failure reporting" do
    it "attributes each failure and error to its test with its message" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never", allow_error: true)

          expect(result.out).to include("OutcomesTest#test_failing_assertion")
          expect(result.out).to include(%(Expected: "TOKEN_WANTED"))
          expect(result.out).to include("OutcomesTest#test_raising_error")
          expect(result.out).to include("ArgumentError: TOKEN_BOOM")
          expect(result.out).to include("OutcomesTest#test_output_then_failure")
          expect(result.out).to include("TOKEN_FLUNKED")
        end
      end
    end

    it "sums and reports failures from two failing workers" do
      chdir(project_fixture!("minitest-failures")) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "2", "--color=never", allow_error: true)
          expect(result).to be_failure

          expect(result.out).to include("13 runs, 16 assertions, 6 failures, 1 error, 0 skips")

          # Both failing workers' detail blocks survive under one "Failures:"
          # header, renumbered sequentially across workers.
          expect(result.out).to include("Failures:")
          expect(result.out.scan(/^ {2}(\d+)\) /).flatten).to eq(%w[1 2 3 4 5 6 7])
          expect(result.out).to include("MixedResultsTest#test_display_name_failure")
          expect(result.out).to include("ArrayOperationsTest#test_average_calculation_failure")
          expect(result.out).to include("ArrayOperationsTest#test_find_max_with_nil")
        end
      end
    end
  end

  context "stdout visibility" do
    it "shows a failed worker's mid-run stdout once, without streaming a duplicate" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never", allow_error: true)

          %w[OUT_MID_RUN OUT_GLOBAL_IO OUT_B4_KABOOM MULTI_1].each do |token|
            expect(result.out.scan(token).length).to eq(1), "expected #{token} exactly once"
          end
        end
      end
    end
  end

  context "plugin loading" do
    # minitest 6 made plugin loading opt-in. plur must load only its own
    # plugin, not re-enable autodiscovery of every minitest/*_plugin.rb on
    # the load path - which would activate plugins the project deliberately
    # left off by upgrading to minitest 6, and can corrupt plur's own output.
    it "does not activate other minitest plugins on the project's behalf" do
      chdir(project_fixture!("minitest-extra-plugin")) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never")
          expect(result).to be_success

          # A stock `ruby -Itest` run on minitest 6 does not load this plugin;
          # neither should plur.
          expect(result.out).not_to include("NOISY_PLUGIN_LOADED")
          expect(result.out).to include("1 run, 1 assertion, 0 failures, 0 errors, 0 skips")
        end
      end
    end
  end

  context "parity with raw minitest" do
    it "reports the same outcome counts and failing tests as running minitest directly" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = Backspin.compare(
            reference: ["bundle", "exec", "ruby", "-Itest",
              "-e", %(require "passing_test"; require "outcomes_test"; require "output_test")],
            actual: [plur_binary, "--use", "minitest", "-n", "1", "--color=never"],
            filter: ->(snapshot) { outcome_facts(snapshot) }
          )

          expect(result.verified?).to be true
          expect(result.actual.status).to eq(1)
        end
      end
    end
  end

  context "discovery and detection" do
    it "detects minitest from the test directory and discovers every test file" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--dry-run", "-n", "2")
          expect(result).to be_success

          expect(result.err).to include("[minitest]")
          expect(result.err).to include("ruby -Itest")
          expect(result.err).to include("passing_test")
          expect(result.err).to include("outcomes_test")
          expect(result.err).to include("output_test")
        end
      end
    end
  end
end
