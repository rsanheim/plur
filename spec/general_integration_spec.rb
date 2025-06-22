require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Rux general integration" do
  let(:expected_spec_files) { 12 }

  describe "basic functionality" do
    it "runs all specs when no arguments are provided" do
      chdir(default_ruby_dir) do
        result = run_rux

        expect(result.out).to include("Running #{expected_spec_files} spec files in parallel")
        expect(result.out).to include("examples")
        expect(result.out).to include("failures")
        expect(result.out).to include("Finished in")
      end
    end

    it "runs specific spec files when provided as arguments" do
      chdir(default_ruby_dir) do
        result = run_rux("spec/calculator_spec.rb", "spec/string_utils_spec.rb")

        expect(result.out).to include("Running 2 spec files in parallel")
        expect(result.out).to include("Finished in")
      end
    end

    it "exits with non-zero status when tests fail" do
      failing_specs_path = project_fixture("failing_specs")
      chdir(failing_specs_path) do
        result = run_rux_allowing_errors("spec/expectation_failures_spec.rb")
        expect(result.exit_status).to eq(1)
      end
    end

    it "exits with zero status when all tests pass" do
      chdir(default_ruby_dir) do
        result = run_rux("spec/calculator_spec.rb")
        expect(result.exit_status).to eq(0)
      end
    end
  end

  describe "worker configuration" do
    it "respects the -n flag for worker count" do
      chdir(default_ruby_dir) do
        result = run_rux("-n", "4")

        expect(result.out).to include("using 4 workers")
      end
    end

    it "respects PARALLEL_TEST_PROCESSORS environment variable" do
      chdir(default_ruby_dir) do
        result = run_rux(env: {"PARALLEL_TEST_PROCESSORS" => "3"})

        expect(result.out).to include("using 3 workers")
      end
    end

    it "prioritizes -n flag over environment variable" do
      chdir(default_ruby_dir) do
        result = run_rux("-n", "5", env: {"PARALLEL_TEST_PROCESSORS" => "3"})

        expect(result.out).to include("using 5 workers")
      end
    end

    it "limits workers to number of spec files when fewer files than workers" do
      chdir(default_ruby_dir) do
        result = run_rux("-n", "20", "spec/calculator_spec.rb", "spec/string_utils_spec.rb")

        expect(result.out).to include("Running 2 spec files in parallel using 2 workers")
      end
    end
  end

  describe "dry-run mode" do
    it "shows what would be executed without running tests" do
      chdir(default_ruby_dir) do
        result = run_rux("--dry-run")

        expect(result.err).to include("[dry-run] Found #{expected_spec_files} spec files")
        expect(result.err).to include("[dry-run] Worker")
        expect(result.err).to include("bundle exec rspec")
        expect(result.out).not_to include("Finished in")
        expect(result.out).not_to include("examples, ")
      end
    end

    it "shows auto bundle install in dry-run mode" do
      chdir(default_ruby_dir) do
        result = run_rux("--dry-run", "--auto")

        expect(result.err).to include("[dry-run] bundle install")
        expect(result.err).to include("[dry-run] Found #{expected_spec_files} spec files")
      end
    end
  end

  # TODO: Re-enable when we implement JSON file output with streaming formatter
  # describe "JSON output" do
  #   it "saves detailed test results when --json flag is used" do
  #     Dir.chdir(default_ruby_dir) do
  #       # Clean up any existing JSON files first
  #       FileUtils.rm_f(Dir.glob("tmp/rux-results-*.json"))
  #
  #       # Run with --json flag
  #       output = `#{rux_binary} --json spec/calculator_spec.rb 2>&1`
  #
  #       # Check that JSON files were created in the project's tmp directory
  #       json_files = Dir.glob("tmp/rux-results-*.json")
  #       expect(json_files).not_to be_empty
  #
  #       # Verify JSON content is valid
  #       json_content = File.read(json_files.first)
  #       parsed = JSON.parse(json_content)
  #
  #       expect(parsed).to have_key("version")
  #       expect(parsed).to have_key("examples")
  #       expect(parsed).to have_key("summary")
  #
  #       # Clean up
  #       FileUtils.rm_f(json_files)
  #     end
  #   end
  # end

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
          result = run_rux("--auto", "test_spec.rb")

          expect(result.out).to include("Installing dependencies...")
          expect(result.out).to include("Running 1 spec files")
        end
      end
    end
  end
end
