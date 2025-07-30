require "spec_helper"

RSpec.describe "Plur database tasks" do
  context "db:setup command (dry-run)" do
    it "runs db:setup in dry-run mode for Rails app" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "3").out

        expect(output).to include("[dry-run] Would run database task 'db:setup' with 3 workers")
        expect(output).to include("[dry-run] Worker 0: TEST_ENV_NUMBER=1 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "runs db:setup with legacy behavior when --no-first-is-1" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "3", "--no-first-is-1").out

        expect(output).to include("[dry-run] Would run database task 'db:setup' with 3 workers")
        expect(output).to include("[dry-run] Worker 0: TEST_ENV_NUMBER= RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "does not set TEST_ENV_NUMBER in serial mode" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:setup", "-n", "1").out

        expect(output).to include("[dry-run] Would run database task 'db:setup' with 1 workers")
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

        expect(output).to include("[dry-run] Would run database task 'db:create' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:create")
      end
    end
  end

  describe "db:migrate command" do
    it "runs db:migrate in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "db:migrate", "-n", "2").out

        expect(output).to include("[dry-run] Would run database task 'db:migrate' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:migrate")
      end
    end
  end

  describe "db:test:prepare command" do
    it "runs db:test:prepare in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        result = run_plur("--dry-run", "db:test:prepare", "-n", "2", allow_error: true)

        expect(result.status).to eq(0)
        expect(result.out).to include("[dry-run] Would run database task 'db:test:prepare' with 2 workers")
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
          expect(result.err).to include("worker 1 failed: exit status 1")

          # Should exit with error
          expect(result.exitstatus).to eq(1)
        end
      end
    end
  end
end
