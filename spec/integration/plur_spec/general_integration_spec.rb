require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Plur general integration" do
  let(:expected_spec_files) { 12 }

  describe "basic functionality" do
    it "runs all specs when no arguments are provided" do
      result = run_plur("-C", default_ruby_dir)

      expect(result.out).to include("Running #{expected_spec_files} spec files in parallel")
      expect(result.out).to include("examples")
      expect(result.out).to include("failures")
      expect(result.out).to include("Finished in")
    end

    it "runs specific spec files when provided as arguments" do
      result = run_plur("-C", default_ruby_dir, "spec/calculator_spec.rb", "spec/string_utils_spec.rb")

      expect(result.out).to include("Running 2 spec files in parallel")
      expect(result.out).to include("Finished in")
    end

    it "exits with non-zero status when tests fail" do
      failing_specs_path = project_fixture("failing_specs")
      result = run_plur_allowing_errors("-C", failing_specs_path, "spec/expectation_failures_spec.rb")
      expect(result.exit_status).to eq(1)
    end

    it "exits with zero status when all tests pass" do
      result = run_plur("-C", default_ruby_dir, "spec/calculator_spec.rb")
      expect(result.exit_status).to eq(0)
    end
  end

  describe "worker configuration" do
    it "respects the -n flag for worker count" do
      result = run_plur("-C", default_ruby_dir, "-n", "4")

      expect(result.out).to include("using 4 workers")
    end

    it "respects PARALLEL_TEST_PROCESSORS environment variable" do
      result = run_plur("-C", default_ruby_dir, env: {"PARALLEL_TEST_PROCESSORS" => "3"})

      expect(result.out).to include("using 3 workers")
    end

    it "prioritizes -n flag over environment variable" do
      result = run_plur("-C", default_ruby_dir, "-n", "5", env: {"PARALLEL_TEST_PROCESSORS" => "3"})

      expect(result.out).to include("using 5 workers")
    end

    it "limits workers to number of spec files when fewer files than workers" do
      result = run_plur("-C", default_ruby_dir, "-n", "20", "spec/calculator_spec.rb", "spec/string_utils_spec.rb")

      expect(result.out).to include("Running 2 spec files in parallel using 2 workers")
    end
  end

  describe "dry-run mode" do
    it "shows what would be executed without running tests" do
      result = run_plur("-C", default_ruby_dir, "--dry-run")

      expect(result.err).to include("[dry-run] Found #{expected_spec_files} spec files")
      expect(result.err).to include("[dry-run] Worker")
      expect(result.err).to include("rspec")
      expect(result.out).not_to include("Finished in")
      expect(result.out).not_to include("examples, ")
    end

    it "shows auto bundle install in dry-run mode" do
      result = run_plur("-C", default_ruby_dir, "--dry-run", "--auto")

      expect(result.err).to include("[dry-run] bundle install")
      expect(result.err).to include("[dry-run] Found #{expected_spec_files} spec files")
    end
  end

  # TODO: Re-enable when we implement JSON file output with streaming formatter
  # describe "JSON output" do

  describe "duration formatting" do
    it "formats sub-second durations with appropriate precision" do
      # Run a single fast test
      result = run_plur("-C", default_ruby_dir, "spec/calculator_spec.rb")

      # Should show sub-second duration with up to 5 decimal places
      expect(result.out).to match(/Finished in 0\.\d{1,5} seconds/)
      # Should not show excessive precision (no more than 5 decimals)
      expect(result.out).not_to match(/Finished in 0\.\d{6,} seconds/)
    end

    it "formats 1-119 second durations with up to 2 decimal places" do
      result = run_plur("-C", default_ruby_dir)

      # Check the output format - should have at most 2 decimal places
      if result.out =~ /Finished in (\d+\.?\d*) seconds/
        duration = $1.to_f
        if duration >= 1 && duration < 120
          # Should have at most 2 decimal places
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,2})? seconds/)
          expect(result.out).not_to match(/Finished in \d+\.\d{3,} seconds/)
        end
      end
    end

    it "strips trailing zeros from duration output" do
      result = run_plur("-C", default_ruby_dir, "spec/calculator_spec.rb")

      # Should not have unnecessary trailing zeros like "1.00" or "0.10000"
      expect(result.out).not_to match(/Finished in \d+\.0{2,} seconds/)
      expect(result.out).not_to match(/Finished in \d+\.\d*0{2,} seconds/)
    end

    it "formats load time with same precision rules" do
      result = run_plur("-C", default_ruby_dir, "spec/calculator_spec.rb")

      # Check that load time also follows the formatting rules
      expect(result.out).to match(/files took \d+(?:\.\d{1,5})? seconds to load/)
      # No excessive precision
      expect(result.out).not_to match(/files took \d+\.\d{6,} seconds to load/)
    end
  end

  describe "auto bundle install" do
    it "runs bundle install before tests with --auto flag" do
      Dir.mktmpdir do |tmpdir|
        # Create a simple project with Gemfile
        gemfile_path = File.join(tmpdir, "Gemfile")
        File.write(gemfile_path, <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        spec_path = File.join(tmpdir, "test_spec.rb")
        File.write(spec_path, <<~RUBY)
          RSpec.describe "test" do
            it "works" do
              expect(1).to eq(1)
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_plur("--auto", "test_spec.rb")

          expect(result.out).to include("Installing dependencies...")
          expect(result.out).to include("Running 1 spec files")
        end
      end
    end
  end
end
