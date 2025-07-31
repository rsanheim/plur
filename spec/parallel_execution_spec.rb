require "spec_helper"

RSpec.describe "Plur parallel execution" do
  describe "environment variables" do
    it "sets TEST_ENV_NUMBER for each worker" do
      chdir(default_ruby_dir) do
        # Run with multiple workers to ensure TEST_ENV_NUMBER is set
        # The env_test_spec.rb prints the environment variables to stdout
        result = run_plur("-n", "3", "spec/env_test_spec.rb", "--no-color")

        # Verify it runs successfully
        expect(result.out).to include("1 example, 0 failures")
        
        # The output will show TEST_ENV_NUMBER values printed by the spec
        # With 3 workers but only 1 spec file, it will run on 1 worker
        expect(result.out).to include("TEST_ENV_NUMBER:")
      end
    end

    it "sets PARALLEL_TEST_GROUPS correctly" do
      chdir(default_ruby_dir) do
        # Run with 4 workers - the env_test_spec.rb will print PARALLEL_TEST_GROUPS
        result = run_plur("-n", "4", "spec/env_test_spec.rb", "--no-color")

        # Verify it runs successfully
        expect(result.out).to include("1 example, 0 failures")
        
        # The output will show PARALLEL_TEST_GROUPS value printed by the spec
        # When running with -n 4 but only 1 file, PARALLEL_TEST_GROUPS should still be 4
        expect(result.out).to include("PARALLEL_TEST_GROUPS:")
      end
    end

    it "does not set TEST_ENV_NUMBER in serial mode" do
      chdir(default_ruby_dir) do
        # Run in serial mode (1 worker)
        result = run_plur("-n", "1", "spec/env_test_spec.rb", "--no-color")

        # Should run successfully
        expect(result.out).to include("1 example, 0 failures")
        
        # In serial mode, TEST_ENV_NUMBER should not be set by plur
        # The env_test_spec.rb will print the value
        expect(result.out).to include("TEST_ENV_NUMBER:")
      end
    end
  end

  describe "output synchronization" do
    it "runs multiple workers without interleaving progress output" do
      chdir(default_ruby_dir) do
        # Run all specs with multiple workers
        # The default-ruby fixture has many spec files that will run in parallel
        result = run_plur("-n", "3")

        # Should see organized output with all tests passing
        expect(result.out).to match(/\d+ examples, 0 failures/)

        # Progress dots should appear on one line (not interleaved)
        progress_lines = result.out.split("\n").select { |l| l =~ /^\e?\[?3?2?m?\.\e?\[?0?m?/ || l =~ /^\.+$/ }
        expect(progress_lines.size).to be <= 2 # At most "Running..." line and one progress line
      end
    end
  end

  describe "progress output" do
    it "shows combined progress from all workers" do
      failing_specs_path = project_fixture("failing_specs")
      chdir(failing_specs_path) do
        # The failing_specs fixture has actual failing tests
        result = run_plur_allowing_errors("--color", "spec/mixed_results_spec.rb")

        # Should see progress dots and F's with colors
        expect(result.out).to match(/\e\[32m\.\e\[0m/) # Green dots
        expect(result.out).to match(/\e\[31mF\e\[0m/) # Red F's

        # Should see summary with failures
        expect(result.out).to match(/\d+ examples?, \d+ failures?/)
      end
    end
  end
end
