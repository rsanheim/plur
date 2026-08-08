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
# exactly in raw minitest output.
#
# minitest-failures is kept for the one thing outcomes cannot express: failures
# in more than one file, so a parallel run has two *failing* workers to
# aggregate.
#
# Determinism: minitest randomizes test order per run, and plur cannot
# currently forward --seed to the multi-file run form, so every expectation
# here holds for ANY order. Exact progress counts are asserted only on the
# files whose stdout writes are all newline-terminated (progress characters
# then always lead their line); the unterminated-print file is covered by an
# order-independent invariant instead.
RSpec.describe "Minitest integration" do
  before(:all) do
    %w[minitest-outcomes minitest-failures].each do |fixture|
      chdir(project_fixture!(fixture)) do
        Bundler.with_unbundled_env do
          system("bundle check", out: File::NULL, err: File::NULL) ||
            system("bundle install", out: File::NULL, err: File::NULL, exception: true)
        end
      end
    end
  end

  let(:project_dir) { project_fixture!("minitest-outcomes") }

  # Occurrences of each progress character plur can render. Skips show as "*"
  # (matching RSpec pending), never "S".
  def alphabet(line)
    %w[. F E S *].to_h { |char| [char, line.count(char)] }
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
    it "reports a green run with a full progress line and RSpec-style duration" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          # No --use: this is the one real (non-dry-run) exercise of framework
          # auto-detection, proven by the [minitest] tag below.
          result = run_plur("-n", "1", "--color=never", "test/passing_test.rb")
          expect(result).to be_success

          expect(result.err).to include("plur version")
          expect(result.err).to include("Running 1 test [minitest]")

          # Two of the five tests write to stdout mid-run, so their text lands
          # on the same physical line as minitest's progress characters. All
          # five dots still reach the first line of plur's output.
          expect(result.out.split("\n").first).to eq("." * 5)
          expect(result.out).to include("5 runs, 6 assertions, 0 failures, 0 errors, 0 skips")

          # Duration uses RSpec's "seconds" wording, not minitest's bare "Xs."
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          expect(result.out).not_to match(/Finished in [\d.]+s\./)

          # Passing workers' stdout is currently dropped entirely.
          expect(result.out).not_to include("OUT_MID_RUN")
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

    it "renders every outcome type in the progress line with exact counts" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never",
            "test/passing_test.rb", "test/outcomes_test.rb", allow_error: true)
          expect(result).to be_failure

          expect(alphabet(result.out.split("\n").first)).to eq(
            "." => 5, "F" => 2, "E" => 1, "S" => 0, "*" => 1
          )
          expect(result.out).to include("9 runs, 8 assertions, 2 failures, 1 error, 1 skip")
        end
      end
    end

    it "under-counts progress when an unterminated print swallows the line" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never", allow_error: true)

          # 12 tests, but the unterminated `print` test's own progress dot is
          # always glued to its output ("PARTIAL_APARTIAL_B.") and the current
          # parser only extracts leading progress characters, so at least that
          # dot is dropped - more when later silent tests glue onto the same
          # line, which depends on the random order. A known limitation: when
          # minitest output handling improves, this becomes exactly 12.
          progress = result.out.split("\n").first
          expect(progress).to match(/\A[.FES*]+\z/)
          expect(progress.length).to be_between(1, 11)
          expect(result.out).to include("12 runs, 11 assertions, 2 failures, 1 error, 1 skip")
        end
      end
    end

    it "aggregates outcomes across parallel workers" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "3", "--color=never", allow_error: true)
          expect(result).to be_failure

          expect(result.out).to include("12 runs, 11 assertions, 2 failures, 1 error, 1 skip")

          # Stdout is currently shown only for failed workers: the failing
          # worker's output survives, the passing workers' output is dropped.
          expect(result.out).to include("OUT_B4_KABOOM")
          expect(result.out).not_to include("OUT_MID_RUN")
          expect(result.out).not_to include("PARTIAL_A")
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

          # Both failing workers' detail blocks survive. plur does not renumber
          # minitest failures across workers, so each block starts at "1)".
          expect(result.out.scan(/^ {2}1\) /).length).to eq(2)
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
