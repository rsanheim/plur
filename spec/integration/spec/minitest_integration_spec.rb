require_relative "../../spec_helper"

# minitest-outcomes is the canonical fixture: 12 tests (8 pass, 2 fail, 1
# error, 1 skip) with mid-run stdout in the styles that stress line handling.
# minitest-failures covers what outcomes cannot: failures across two files.
#
# minitest randomizes test order and plur cannot forward --seed to multi-file
# runs, so every expectation holds for ANY order. Fixture stdout tokens avoid
# ".FES" and counts are asserted across the whole progress region, so live
# interleaving of output and progress characters cannot skew them.
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

  # Counts each progress character across the progress region only - failure
  # details and the summary legitimately contain F/E/dots. Skips render "*".
  def progress_alphabet(out)
    region = out.split("\nFailures:").first.split("\nFinished in").first
    %w[. F E S *].to_h { |char| [char, region.count(char)] }
  end

  # Order- and runner-independent facts: outcome counts and which tests failed.
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
          # No --use: the one real exercise of framework auto-detection.
          result = run_plur("-n", "1", "--color=never", "test/passing_test.rb")
          expect(result).to be_success

          expect(result.err).to include("plur version")
          expect(result.err).to include("Running 1 test [minitest]")

          expect(progress_alphabet(result.out)).to eq(
            "." => 5, "F" => 0, "E" => 0, "S" => 0, "*" => 0
          )
          expect(result.out).to include("5 runs, 6 assertions, 0 failures, 0 errors, 0 skips")

          # RSpec's "seconds" wording, not minitest's bare "Xs."
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          expect(result.out).not_to match(/Finished in [\d.]+s\./)

          # Stdout streams live even on a passing run.
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

          # An unterminated print shares a physical line with the next row;
          # the parser splits them, losing neither the dot nor the text.
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
          %w[OUT_B4_KABOOM OUT_MID_RUN PARTIAL_A].each do |token|
            expect(result.out).to include(token)
          end
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

          # Both workers' failures land under one header, renumbered across them.
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

    it "shows an errored worker's output once, not duplicated at the end" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          error_file = "test/error_load_test.rb"
          File.write(error_file, <<~RUBY)
            puts "STDOUT_BEFORE_CRASH"
            require "minitest/autorun"
            require "this_gem_does_not_exist_surely"
          RUBY

          result = run_plur("-v", "--use", "minitest", "-n", "2", "--color=never",
            "test/passing_test.rb", error_file, allow_error: true)
          expect(result).to be_failure
          puts result.out
          puts result.err

          expect(result.out.scan("STDOUT_BEFORE_CRASH").length).to eq(1),
            "errored worker stdout should appear once (streamed live), not twice (re-dumped)"
        ensure
          FileUtils.rm_f(error_file)
        end
      end
    end
  end

  context "plugin loading" do
    # On minitest 6 (opt-in plugin loading) plur loads only its own plugin.
    # The fixture ships a discoverable minitest/noisy_plugin.rb that a stock
    # `ruby -Itest` run would not activate; neither should plur.
    it "does not activate other minitest plugins on the project's behalf" do
      chdir(project_fixture!("minitest-extra-plugin")) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--color=never")
          expect(result).to be_success

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
