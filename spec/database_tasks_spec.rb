require "spec_helper"

RSpec.describe "Rux database tasks" do
  describe "db:setup command" do
    it "runs db:setup in dry-run mode for Rails app" do
      Dir.chdir(default_rails_dir) do
        output = run_rux("--dry-run", "db:setup", "-n", "3").out

        expect(output).to include("[dry-run] Would run database task 'db:setup' with 3 workers")
        expect(output).to include("[dry-run] Worker 0: RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "shows completion message when running actual task" do
      # Use a simple Rakefile in a temp dir to avoid actually running db:setup on test_app
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "Gemfile"), <<~RUBY)
          source "https://rubygems.org"
          gem "rake"
        RUBY

        File.write(File.join(tmpdir, "Rakefile"), <<~RUBY)
          task "db:setup" do
            puts "DB setup for \#{ENV['TEST_ENV_NUMBER'] || '1'}"
          end
        RUBY

        Dir.chdir(tmpdir) do
          output = run_rux("db:setup", "-n", "2").out

          expect(output).to include("Running database task 'db:setup' with 2 workers...")
          expect(output).to include("Database task 'db:setup' completed successfully")
        end
      end
    end
  end

  describe "db:create command" do
    it "runs db:create in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        output = run_rux("--dry-run", "db:create", "-n", "2").out

        expect(output).to include("[dry-run] Would run database task 'db:create' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:create")
      end
    end
  end

  describe "db:migrate command" do
    it "runs db:migrate in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        output = run_rux("--dry-run", "db:migrate", "-n", "2").out

        expect(output).to include("[dry-run] Would run database task 'db:migrate' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:migrate")
      end
    end
  end

  describe "db:test:prepare command" do
    it "runs db:test:prepare in dry-run mode" do
      Dir.chdir(default_rails_dir) do
        result = run_rux("--dry-run", "db:test:prepare", "-n", "2", allow_error: true)

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
          result = run_rux("db:setup", "-n", "3", allow_error: true)

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
