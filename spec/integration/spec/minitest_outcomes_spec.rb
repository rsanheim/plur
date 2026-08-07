require_relative "../../spec_helper"

# minitest-outcomes is the canonical minitest fixture: every outcome the
# progress line can show (pass, failure, error, skip), stdout written mid-run
# in the styles that stress line handling (interleaved puts, unterminated
# print, multi-line), and a unique greppable token for every case.
#
# 12 tests: 8 pass, 2 fail, 1 error, 1 skip. Tokens written to stdout during
# the run avoid the characters ".FES" so progress characters can be counted
# exactly in raw minitest output.
#
# Determinism: minitest randomizes test order per run, and plur cannot
# currently forward --seed to the multi-file run form, so every expectation
# here holds for ANY order. Exact progress counts are asserted only on the
# files whose stdout writes are all newline-terminated (progress characters
# then always lead their line); the unterminated-print file is covered by an
# order-independent invariant instead.
RSpec.describe "Minitest outcomes fixture" do
  before(:all) do
    chdir(project_fixture!("minitest-outcomes")) do
      Bundler.with_unbundled_env do
        system("bundle check", out: File::NULL, err: File::NULL) ||
          system("bundle install", out: File::NULL, err: File::NULL, exception: true)
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
