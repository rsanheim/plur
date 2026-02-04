require_relative "../../spec_helper"

RSpec.describe "Minitest Integration" do
  before(:all) do
    chdir(project_fixture!("minitest-success")) do
      Bundler.with_unbundled_env do
        system("bundle install", exception: true)
      end
    end
  end

  context "with minitest-success project" do
    let(:project_dir) { project_fixture!("minitest-success") }

    it "runs minitest tests successfully" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1")
          expect(result).to be_success
          expect(result.err).to include("plur version")
          expect(result.err).to include("Running 2 tests [minitest]")
          # Minitest shows the final summary
          expect(result.out).to match(/\d+ runs?, \d+ assertions?, 0 failures?, 0 errors?/)
        end
      end
    end

    it "formats minitest duration using RSpec-style formatting" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1")
          expect(result).to be_success
          # Should use "seconds" not "s" suffix
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          # Should NOT use the old "Xs" format
          expect(result.out).not_to match(/Finished in [\d.]+s\./)
        end
      end
    end

    it "discovers minitest test files" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "--dry-run")
          expect(result).to be_success
          expect(result.err).to include("test/calculator_test.rb")
          expect(result.err).to include("test/string_helper_test.rb")
        end
      end
    end
  end

  context "with minitest-failures project" do
    let(:project_dir) { project_fixture!("minitest-failures") }

    it "reports minitest test failures" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", allow_error: true)
          expect(result).to be_failure
          # Check for failure output in stdout (not stderr)
          # In parallel execution, individual failure details are not shown
          # Check that the summary shows failures
          expect(result.out).to match(/\d+ failures?/)
        end
      end
    end
  end

  context "with auto-detection" do
    it "detects minitest project by test directory" do
      chdir(project_fixture!("minitest-success")) do
        Bundler.with_unbundled_env do
          result = run_plur("--dry-run")
          expect(result).to be_success
          # Should use minitest commands
          expect(result.err).to include("ruby -Itest")
        end
      end
    end
  end

  context "with grouped minitest command output" do
    let(:project_dir) { project_fixture!("minitest-success") }
    let(:minitest_seed) { "8378" }

    def normalize_minitest_output(output)
      output
        .gsub(/Run options: --seed \d+/, "Run options: --seed [SEED]")
        .gsub(/Finished in [\d.]+s, [\d.]+ runs\/s, [\d.]+ assertions\/s\./,
          "Finished in [DURATION]s, [RUNS_PER_SEC] runs/s, [ASSERTIONS_PER_SEC] assertions/s.")
    end

    def run_grouped_minitest(seed:)
      # Use a single ruby command with a quoted -e script to require each file.
      command = [
        "bundle", "exec", "ruby",
        "-Itest",
        "-e", '["calculator_test", "string_helper_test"].each { |f| require f }',
        "--", "--seed", seed
      ]

      Open3.capture3(*command)
    end

    def run_plur_minitest_serial
      Open3.capture3("plur", "--use", "minitest", "-n", "1", "--no-color")
    end

    it "records grouped minitest output from a ruby -e require list" do
      stdout_matcher = ->(record, actual) {
        normalize_minitest_output(record) == normalize_minitest_output(actual)
      }
      stderr_matcher = ->(_record, _actual) { true }

      Backspin.run!("minitest_grouped_ruby_command_output",
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}) do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            run_grouped_minitest(seed: minitest_seed)
          end
        end
      end
    end

    it "compares plur -n1 output to grouped minitest output" do
      pending("plur output does not match raw minitest output yet")

      stdout_matcher = ->(record, actual) {
        normalize_minitest_output(record) == normalize_minitest_output(actual)
      }
      stderr_matcher = ->(_record, _actual) { true }

      Backspin.run!("minitest_grouped_ruby_command_output",
        mode: :verify,
        matcher: {stdout: stdout_matcher, stderr: stderr_matcher}) do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            run_plur_minitest_serial
          end
        end
      end
    end
  end
end
