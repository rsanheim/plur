require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Rux general integration" do
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }
  let(:test_project_path) { File.join(__dir__, "..", "rux-ruby") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  describe "basic functionality" do
    it "runs all specs when no arguments are provided" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} 2>&1`

        expect(output).to include("Running 11 spec files in parallel")
        expect(output).to include("examples")
        expect(output).to include("failures")
        expect(output).to include("Finished in")
      end
    end

    it "runs specific spec files when provided as arguments" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} spec/calculator_spec.rb spec/string_utils_spec.rb 2>&1`

        expect(output).to include("Running 2 spec files in parallel")
        expect(output).to include("Finished in")
      end
    end

    it "exits with non-zero status when tests fail" do
      failing_specs_path = File.join(__dir__, "..", "test_fixtures", "failing_specs")
      Dir.chdir(failing_specs_path) do
        system("#{rux_binary} spec/expectation_failures_spec.rb 2>&1", out: File::NULL)
        expect($?.exitstatus).to eq(1)
      end
    end

    it "exits with zero status when all tests pass" do
      Dir.chdir(test_project_path) do
        system("#{rux_binary} spec/calculator_spec.rb 2>&1", out: File::NULL)
        expect($?.exitstatus).to eq(0)
      end
    end
  end

  describe "worker configuration" do
    it "respects the -n flag for worker count" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} -n 4 2>&1`

        expect(output).to include("using 4 workers")
      end
    end

    it "respects PARALLEL_TEST_PROCESSORS environment variable" do
      Dir.chdir(test_project_path) do
        output = `PARALLEL_TEST_PROCESSORS=3 #{rux_binary} 2>&1`

        expect(output).to include("using 3 workers")
      end
    end

    it "prioritizes -n flag over environment variable" do
      Dir.chdir(test_project_path) do
        output = `PARALLEL_TEST_PROCESSORS=3 #{rux_binary} -n 5 2>&1`

        expect(output).to include("using 5 workers")
      end
    end

    it "limits workers to number of spec files when fewer files than workers" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} -n 20 spec/calculator_spec.rb spec/string_utils_spec.rb 2>&1`

        expect(output).to include("Running 2 spec files in parallel using 2 workers")
      end
    end
  end

  describe "dry-run mode" do
    it "shows what would be executed without running tests" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} --dry-run 2>&1`

        expect(output).to include("[dry-run] Found 11 spec files")
        expect(output).to include("[dry-run] bundle exec rspec")
        expect(output).not_to include("Finished in")
        expect(output).not_to include("examples, ")
      end
    end

    it "shows auto bundle install in dry-run mode" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} --dry-run --auto 2>&1`

        expect(output).to include("[dry-run] bundle install")
        expect(output).to include("[dry-run] Found 11 spec files")
      end
    end
  end

  # TODO: Re-enable when we implement JSON file output with streaming formatter
  # describe "JSON output" do
  #   it "saves detailed test results when --json flag is used" do
  #     Dir.chdir(test_project_path) do
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

        Dir.chdir(tmpdir) do
          output = `#{rux_binary} --auto test_spec.rb 2>&1`

          expect(output).to include("Installing dependencies...")
          expect(output).to include("Running 1 spec files")
        end
      end
    end
  end
end
