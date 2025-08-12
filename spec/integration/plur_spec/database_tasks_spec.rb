require "spec_helper"

RSpec.describe "Plur database tasks" do
  context "db:setup command (dry-run)" do
    it "runs db:setup in dry-run mode for Rails app" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "3").out

        expect(output).to include("[dry-run] Worker 0: TEST_ENV_NUMBER=1 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "runs db:setup with legacy behavior when --no-first-is-1" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "3", "--no-first-is-1").out

        expect(output).to include("[dry-run] Worker 0: TEST_ENV_NUMBER= RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "does not set TEST_ENV_NUMBER in serial mode" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "1").out

        expect(output).to include("[dry-run] Worker 0: RAILS_ENV=test bundle exec rake db:setup")
        expect(output).not_to include("TEST_ENV_NUMBER")
      end
    end
  end

  context "db:create command (for-real)" do
    before do
      # Clean up any existing test databases
      Dir.chdir(default_rails_dir) do
        FileUtils.rm_f(Dir.glob("storage/*.sqlite3"))
      end
    end

    it "creates and migrates databases for parallel testing", pending: "Needs investigation, db tasks are failing" do
      Dir.chdir(default_rails_dir) do
        system("bundle install")

        result = run_plur("db:create", "-n", "3", allow_error: true)
        pp result
        expect(result.status).to eq(0), "db:create failed: #{result.err}"

        # Check that databases were created
        expect(File.exist?("storage/test.sqlite3")).to be true
        expect(File.exist?("storage/test2.sqlite3")).to be true
        expect(File.exist?("storage/test3.sqlite3")).to be true

        # Run migrations
        result = run_plur("db", "migrate", "-n", "3")
        expect(result.exit_status).to eq(0), "db:migrate failed: #{result.err}"
      end
    end
  end

  describe "db:create command" do
    it "runs db:create in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:create", "-n", "2").out

        expect(output).to include("RAILS_ENV=test bundle exec rake db:create")
      end
    end
  end

  describe "db:migrate command" do
    it "runs db:migrate in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:migrate", "-n", "2").out

        expect(output).to include("RAILS_ENV=test bundle exec rake db:migrate")
      end
    end
  end

  describe "db:test:prepare command" do
    it "runs db:test:prepare in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        result = run_plur("--dry-run", "db:test:prepare", "-n", "2", allow_error: true)

        expect(result.status).to eq(0)
        expect(result.out).to include("RAILS_ENV=test bundle exec rake db:test:prepare")
      end
    end
  end

  describe "error handling" do
    it "reports failures from database tasks" do
      Dir.mktmpdir do |tmpdir|
        rakefile_path = File.join(tmpdir, "Rakefile")
        File.write(rakefile_path, <<~RUBY)
          task "db:setup" do
            env_num = ENV['TEST_ENV_NUMBER'] || ''
            if env_num == "2"
              STDERR.puts "Error: Database 2 setup failed"
              exit 1
            end
            puts "Setting up test database\#{env_num}"
          end
        RUBY

        Dir.chdir(tmpdir) do
          result = run_plur("db:setup", "-n", "3", allow_error: true)

          expect(result.status).to eq(1)
          expect(result.out).to include("Running database task 'db:setup' with 3 workers...")
          expect(result.err).to include("Command failed error=database task failed:")
          expect(result.err).to include("worker 1 failed:")
          # Now shows the actual error output
          expect(result.err).to include("Error: Database 2 setup failed")

          # Should exit with error
          expect(result.exitstatus).to eq(1)
        end
      end
    end

    it "shows stderr output when database tasks fail" do
      Dir.mktmpdir do |tmpdir|
        rakefile_path = File.join(tmpdir, "Rakefile")
        File.write(rakefile_path, <<~RUBY)
          task "db:setup" do
            env_num = ENV['TEST_ENV_NUMBER'] || ''
            STDERR.puts "Error: Database connection failed for test\#{env_num}"
            STDERR.puts "Could not connect to server: Connection refused"
            STDERR.puts "Is the server running on host 'localhost' and accepting TCP/IP connections on port 5432?"
            exit 1
          end
        RUBY

        Dir.chdir(tmpdir) do
          result = run_plur("db:setup", "-n", "2", allow_error: true)

          expect(result.status).to eq(1)
          expect(result.out).to include("Running database task 'db:setup' with 2 workers...")

          # Now we should see the actual stderr output in the error message
          expect(result.err).to include("database task failed:")
          expect(result.err).to include("Error: Database connection failed")
          expect(result.err).to include("Could not connect to server: Connection refused")
          expect(result.err).to include("Is the server running on host 'localhost'")
        end
      end
    end

    it "deduplicates identical errors across all workers" do
      Dir.mktmpdir do |tmpdir|
        rakefile_path = File.join(tmpdir, "Rakefile")
        File.write(rakefile_path, <<~RUBY)
          task "db:setup" do
            # All workers will fail with the same error
            STDERR.puts "Error: PostgreSQL is not installed"
            STDERR.puts "Please install PostgreSQL and try again"
            exit 1
          end
        RUBY

        Dir.chdir(tmpdir) do
          result = run_plur("db:setup", "-n", "3", allow_error: true)

          expect(result.status).to eq(1)
          expect(result.out).to include("Running database task 'db:setup' with 3 workers...")

          # Should show that all workers failed with the same error
          expect(result.err).to include("All 3 workers failed with the same error:")
          expect(result.err).to include("Error: PostgreSQL is not installed")
          expect(result.err).to include("Please install PostgreSQL and try again")

          # Should NOT show the error 3 times
          expect(result.err.scan("PostgreSQL is not installed").length).to eq(1)
        end
      end
    end

    it "shows different errors separately when workers fail differently" do
      Dir.mktmpdir do |tmpdir|
        rakefile_path = File.join(tmpdir, "Rakefile")
        File.write(rakefile_path, <<~RUBY)
          task "db:setup" do
            env_num = ENV['TEST_ENV_NUMBER'] || ''
            case env_num
            when "1", ""
              STDERR.puts "Error: Database test\#{env_num} already exists"
              exit 1
            when "2"
              STDERR.puts "Error: Permission denied for database test2"
              exit 1
            else
              puts "Successfully created test\#{env_num}"
            end
          end
        RUBY

        Dir.chdir(tmpdir) do
          result = run_plur("db:setup", "-n", "3", allow_error: true)

          expect(result.status).to eq(1)
          expect(result.out).to include("Running database task 'db:setup' with 3 workers...")

          # Should show each unique error
          expect(result.err).to include("database task failed:")
          expect(result.err).to include("Database test1 already exists")
          expect(result.err).to include("Permission denied for database test2")

          # Worker 2 (TEST_ENV_NUMBER=3) should succeed, so only 2 errors
          expect(result.err).to include("worker 0 failed")
          expect(result.err).to include("worker 1 failed")
        end
      end
    end
  end
end
