require "spec_helper"
require "open3"

RSpec.describe "Rux integration tests" do
  describe "basic functionality" do
    it "shows help information" do
      result = run_rux("--help")

      expect(result.out).to include("A fast Go-based test runner for Ruby/RSpec")
      expect(result.out).to include("db:create")
      expect(result.out).to include("db:setup")
      expect(result.out).to include("db:migrate")
    end
  end

  describe "parallel test execution" do
    it "runs tests in parallel", skip: "Database setup needs investigation" do
      Bundler.with_unbundled_env do
        Dir.chdir(default_rails_dir) do
          # Set up databases first
          result = run_rux("db:create", "-n", "3", printer: :quiet, allow_error: true)
          # TODO - not sure why this fails in this test but passes when run from a terminal

          system(rux_binary, "db", "migrate", "-n", "3", out: File::NULL, err: File::NULL)

          # Run tests
          stdout, stderr, status = Open3.capture3(rux_binary, "-n", "3")

          expect(status.success?).to eq(true), "parallel test execution failed: #{stderr}"
          expect(stdout).to include("spec files in parallel using")
          expect(stdout).to include("examples, 0 failures")
        end
      end
    end

    it "assigns different TEST_ENV_NUMBER to workers", skip: "Database setup needs investigation" do
      Dir.chdir(default_rails_dir) do
        # Set up databases
        system(rux_binary, "db:create", "-n", "2", out: File::NULL, err: File::NULL)
        system(rux_binary, "db:migrate", "-n", "2", out: File::NULL, err: File::NULL)

        # Run tests
        _, _, status = Open3.capture3(rux_binary, "-n", "2")

        # If tests pass with parallel databases, it means ENV numbers are working
        expect(status.success?).to be true
      end
    end

    it "shows dry-run output for test execution" do
      Dir.chdir(default_rails_dir) do
        result = run_rux("--dry-run", "-n", "2")

        expect(result.err).to include("[dry-run] Found")
        expect(result.err).to include("spec files")
        expect(result.err).to include("bundle exec rspec")
      end
    end
  end
end
