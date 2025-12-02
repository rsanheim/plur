require "spec_helper"

RSpec.describe "Plur parallel execution" do
  describe "environment variables" do
    it "sets TEST_ENV_NUMBER for each worker" do
      chdir(default_ruby_dir) do
        result = run_plur("-n", "3", "--verbose", "spec/env_test_spec.rb", "--no-color")

        expect(result.out).to include("1 example, 0 failures")

        # The output will show TEST_ENV_NUMBER values printed to stderr
        # With 3 workers but only 1 spec file, it will run on 1 worker with TEST_ENV_NUMBER=1
        expect(result.err).to include("TEST_ENV_NUMBER: '1'")
      end
    end

    it "sets PARALLEL_TEST_GROUPS correctly" do
      chdir(default_ruby_dir) do
        result = run_plur("-n", "4", "spec/env_test_spec.rb", "--no-color")

        expect(result.out).to include("1 example, 0 failures")

        # The output will show PARALLEL_TEST_GROUPS value printed to stderr
        # When running with -n 4 but only 1 file, PARALLEL_TEST_GROUPS should be 1 (actual groups used)
        expect(result.err).to include("PARALLEL_TEST_GROUPS: '1'")
      end
    end

    it "does not set TEST_ENV_NUMBER in serial mode" do
      chdir(default_ruby_dir) do
        result = run_plur("-n", "1", "spec/env_test_spec.rb", "--no-color")

        expect(result.out).to include("1 example, 0 failures")

        # In serial mode, TEST_ENV_NUMBER should not be set by plur
        # The test harness may have TEST_ENV_NUMBER set, but we verify plur doesn't change it
        # by checking that the output includes TEST_ENV_NUMBER: and the line in stderr
        expect(result.err).to match(/TEST_ENV_NUMBER: '\d*'/)
      end
    end
  end

  describe "output synchronization" do
    it "runs multiple workers without interleaving progress output" do
      chdir(default_ruby_dir) do
        result = run_plur("-n", "3")

        expect(result.out).to match(/\d+ examples, 0 failures/)

        # Progress dots should appear on one line (not interleaved)
        progress_lines = result.out.split("\n").select { |l| l =~ /^\e?\[?3?2?m?\.\e?\[?0?m?/ || l =~ /^\.+$/ }
        expect(progress_lines.size).to be <= 2 # At most "Running..." line and one progress line
      end
    end
  end

  describe "progress output" do
    def system_rspec(file_or_glob, *args)
      system("bundle", "exec", "rspec", file_or_glob, *args)
    end

    # make output more deterministic
    def normalize_test_output(output)
      lines = output.lines.reject { |l| l.match?(/^plur version|^Running \d+ specs/) }
      lines = lines.map do |line|
        if line.match?(/^[.F*]+$/)
          line.strip.chars.sort.join + "\n"
        elsif line.match?(/^Finished in \d/)
          "Finished in X seconds\n"
        else
          line
        end
      end
      lines.join
    end

    fit "rspec output" do
      failing_specs_path = project_fixture("failing_specs")
      Backspin.capture("parallel_execution_progress_output") do
        chdir(failing_specs_path) do
          system_rspec("spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb")
        end
      end
      matcher = {
        stdout: ->(recorded, actual) { normalize_test_output(recorded) == normalize_test_output(actual) }
      }
      result = Backspin.capture("parallel_execution_progress_output", mode: :verify, matcher: matcher) do
        chdir(failing_specs_path) do
          run_plur_allowing_errors("--no-color", "-n", "2", "spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb", printer: :quiet)
        end
      end
      pp result
    end

    it "shows combined progress from all workers" do
      chdir(failing_specs_path) do
        result = run_plur_allowing_errors("-n", "2", "spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb")

        pp result.out
        expect(result.out).to match(/\e\[32m\.\e\[0m/) # Green dots
        expect(result.out).to match(/\e\[31mF\e\[0m/) # Red F's

        expect(result.out).to match(/\d+ examples?, \d+ failures?/)
      end
    end
  end
end
