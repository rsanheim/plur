require "spec_helper"
require "open3"

RSpec.describe "Rux integration tests" do
  before do
    # Clean up any existing test databases
    Dir.chdir(default_rails_dir) do
      FileUtils.rm_f(Dir.glob("storage/test*.sqlite3"))
    end
  end

  describe "basic functionality" do
    it "has a built rux binary" do
      expect(File.exist?(rux_binary)).to be true
    end

    it "shows help information" do
      stdout, _, status = Open3.capture3(rux_binary, "--help")

      expect(status.success?).to be true
      expect(stdout).to include("rux - A fast Go-based test runner for Ruby/RSpec")
      expect(stdout).to include("db:create")
      expect(stdout).to include("db:setup")
      expect(stdout).to include("db:migrate")
    end
  end

  describe "database tasks" do
    it "shows dry-run output for database creation" do
      Dir.chdir(default_rails_dir) do
        stdout, _, status = Open3.capture3(rux_binary, "db:create", "--dry-run", "-n", "3")

        expect(status.success?).to be true
        expect(stdout).to include("[dry-run] Would run database task 'db:create' with 3 workers")
        expect(stdout).to include("Worker 0: RAILS_ENV=test bundle exec rake db:create")
        expect(stdout).to include("Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:create")
        expect(stdout).to include("Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:create")
      end
    end

    it "creates and migrates databases for parallel testing", pending: "Database setup needs investigation" do
      Dir.chdir(default_rails_dir) do
        # Create databases
        _, stderr, status = Open3.capture3(rux_binary, "db:create", "-n", "3")
        expect(status.success?).to eq(true), "db:create failed: #{stderr}"

        # Check that databases were created
        expect(File.exist?("storage/test.sqlite3")).to be true
        expect(File.exist?("storage/test2.sqlite3")).to be true
        expect(File.exist?("storage/test3.sqlite3")).to be true

        # Run migrations
        _, stderr, status = Open3.capture3(rux_binary, "db:migrate", "-n", "3")
        expect(status.success?).to eq(true), "db:migrate failed: #{stderr}"
      end
    end
  end

  describe "parallel test execution" do
    it "runs tests in parallel", pending: "Database setup needs investigation" do
      Dir.chdir(default_rails_dir) do
        # Set up databases first
        system(rux_binary, "db:create", "-n", "3", out: File::NULL, err: File::NULL)
        system(rux_binary, "db:migrate", "-n", "3", out: File::NULL, err: File::NULL)

        # Run tests
        stdout, stderr, status = Open3.capture3(rux_binary, "-n", "3")

        expect(status.success?).to eq(true), "parallel test execution failed: #{stderr}"
        expect(stdout).to include("spec files in parallel using")
        expect(stdout).to include("examples, 0 failures")
      end
    end

    it "assigns different TEST_ENV_NUMBER to workers", pending: "Database setup needs investigation" do
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
        stdout, stderr, status = Open3.capture3(rux_binary, "--dry-run", "-n", "2")

        expect(status.success?).to be true
        output = stdout + stderr
        expect(output).to include("[dry-run] Found")
        expect(output).to include("spec files")
        expect(output).to include("bundle exec rspec")
      end
    end
  end
end
