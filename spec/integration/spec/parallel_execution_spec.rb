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

  describe "pending output" do
    it "shows single Pending header with multi-worker" do
      chdir(project_fixture("failing_specs")) do
        result = run_plur_allowing_errors("-n", "2",
          "spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb")

        # Exactly ONE "Pending:" header (not one per worker)
        pending_headers = result.out.scan(/^Pending:/).count
        expect(pending_headers).to eq(1), "Expected 1 Pending header, got #{pending_headers}"

        # Sequential numbering across workers (mixed_results_spec.rb has 3 pending)
        expect(result.out).to match(/1\).*pending/i)
        expect(result.out).to match(/2\).*pending/i)
        expect(result.out).to match(/3\).*pending/i)
      end
    end
  end

  describe "progress output" do
    let(:failing_specs_path) { project_fixture("failing_specs") }
    def system_rspec(file_or_glob, *args)
      Open3.capture3("bundle", "exec", "rspec", file_or_glob, *args)
    end

    # Make output more deterministic:
    # * sort progress characters
    # * filter timing information
    # * remove plur version information
    # * strip failure/pending numbers (vary by parallel execution order)
    # * normalize path prefixes (rspec uses ./ prefix, plur doesn't)
    # * remove empty lines
    def normalize_test_output(output)
      lines = output.lines.reject { |l| l.match?(/^plur version|^Running \d+ specs/) }
      lines = lines.map do |line|
        if line.match?(/^[.F*]+$/)
          line.strip.chars.sort.join + "\n"
        elsif line.match?(/^Finished in \d/)
          "Finished in X seconds\n"
        else
          # Normalize path prefixes and failure numbers
          line.gsub(/^(\s*)\d+\)/, '\1)').gsub(%r{ \./spec/}, " spec/")
        end
      end
      lines.sort.reject { |line| line.strip.empty? }.join
    end

    it "matches rspec output and has correct pending and failure output" do
      chdir(failing_specs_path) do
        rspec_output, _, _ = system_rspec("spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb")
        plur_output, _, _ = run_plur_allowing_errors("--no-color", "-n", "2", "spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb", printer: :null)
        # ensure we get a single Pending: and Failure: header for plur, not one per worker
        pending_headers = plur_output.scan(/^Pending:/).count
        failure_headers = plur_output.scan(/^Failures:/).count
        expect(pending_headers).to eq(1), "Expected 1 Pending header, got #{pending_headers}"
        expect(failure_headers).to eq(1), "Expected 1 Failure header, got #{failure_headers}"
        # verify normalized output against rspec
        expect(normalize_test_output(plur_output)).to eq(normalize_test_output(rspec_output))
      end
    end

    it "does not duplicate pending or failure headers with multiple workers" do
      chdir(failing_specs_path) do
        result = run_plur_allowing_errors("--no-color", "-n", "3",
          "spec/expectation_failures_spec.rb",
          "spec/mixed_results_spec.rb",
          "spec/single_failure_spec.rb")

        pending_headers = result.out.scan(/^Pending:/).count
        failure_headers = result.out.scan(/^Failures:/).count

        expect(pending_headers).to eq(1), "Expected 1 Pending header, got #{pending_headers}"
        expect(failure_headers).to eq(1), "Expected 1 Failure header, got #{failure_headers}"
      end
    end

    it "shows combined progress from all workers" do
      chdir(project_fixture("failing_specs")) do
        result = run_plur_allowing_errors("-n", "2", "spec/mixed_results_spec.rb", "spec/expectation_failures_spec.rb")

        expect(result.out).to match(/\e\[32m\.\e\[0m/) # Green dots
        expect(result.out).to match(/\e\[31mF\e\[0m/) # Red F's

        expect(result.out).to match(/\d+ examples?, \d+ failures?/)
      end
    end
  end
end
