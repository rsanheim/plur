require "spec_helper"

RSpec.describe "Plur Rails and Rake commands" do
  def dry_run_worker_line(output, worker)
    line = output.lines.find { |candidate| candidate.include?("[dry-run] Worker #{worker}:") }
    expect(line).not_to be_nil, "expected dry-run command for worker #{worker}, got:\n#{output}"
    line
  end

  def without_rails_env
    original = ENV.delete("RAILS_ENV")
    yield
  ensure
    ENV["RAILS_ENV"] = original if original
  end

  context "rails command (dry-run)" do
    it "runs a Rails command in dry-run mode for each worker" do
      without_rails_env do
        Dir.chdir(default_rails_dir) do
          output = run_plur("--dry-run", "rails", "db:prepare", "-n", "3").err

          worker0 = dry_run_worker_line(output, 0)
          expect(worker0).to include("PARALLEL_TEST_GROUPS=3")
          expect(worker0).to include("TEST_ENV_NUMBER=1")
          expect(worker0).to include("bin/rails db:prepare")
          expect(worker0).not_to include("RAILS_ENV=test")

          worker1 = dry_run_worker_line(output, 1)
          expect(worker1).to include("PARALLEL_TEST_GROUPS=3")
          expect(worker1).to include("TEST_ENV_NUMBER=2")
          expect(worker1).to include("bin/rails db:prepare")
          expect(worker1).not_to include("RAILS_ENV=test")

          worker2 = dry_run_worker_line(output, 2)
          expect(worker2).to include("PARALLEL_TEST_GROUPS=3")
          expect(worker2).to include("TEST_ENV_NUMBER=3")
          expect(worker2).to include("bin/rails db:prepare")
          expect(worker2).not_to include("RAILS_ENV=test")
        end
      end
    end

    it "uses legacy first worker env when --no-first-is-1 is set" do
      without_rails_env do
        Dir.chdir(default_rails_dir) do
          output = run_plur("--dry-run", "rails", "db:prepare", "-n", "3", "--no-first-is-1").err

          worker0 = dry_run_worker_line(output, 0)
          expect(worker0).to include("PARALLEL_TEST_GROUPS=3")
          expect(worker0).to include("TEST_ENV_NUMBER=")
          expect(worker0).to include("bin/rails db:prepare")
          expect(worker0).not_to include("RAILS_ENV=test")

          worker1 = dry_run_worker_line(output, 1)
          expect(worker1).to include("PARALLEL_TEST_GROUPS=3")
          expect(worker1).to include("TEST_ENV_NUMBER=2")
          expect(worker1).to include("bin/rails db:prepare")
        end
      end
    end

    it "does not set TEST_ENV_NUMBER in serial mode" do
      ENV.delete("TEST_ENV_NUMBER")

      without_rails_env do
        Dir.chdir(default_rails_dir) do
          output = run_plur("--dry-run", "rails", "db:prepare", "-n", "1").err

          worker0 = dry_run_worker_line(output, 0)
          expect(worker0).to include("PARALLEL_TEST_GROUPS=1")
          expect(worker0).to include("bin/rails db:prepare")
          expect(worker0).not_to include("TEST_ENV_NUMBER")
          expect(worker0).not_to include("RAILS_ENV=test")
        end
      end
    end

    it "passes command args literally instead of treating them as file patterns" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "rails", "db:migrate", "VERSION=20260429000000", "-n", "2").err

        expect(output).to include("bin/rails db:migrate VERSION=20260429000000")
        expect(output).not_to include("no test files found")
      end
    end

    it "shows explicit Rails env when the caller provides it" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "rails", "db:prepare", "-n", "1", env: {"RAILS_ENV" => "test"}).err

        expect(output).to include("RAILS_ENV=test")
        expect(output).to include("bin/rails db:prepare")
      end
    end

    it "passes rails-specific flags after a double dash" do
      Dir.chdir(default_rails_dir) do
        output = run_plur("--dry-run", "rails", "db:migrate", "-n", "2", "--", "--trace").err

        worker0 = dry_run_worker_line(output, 0)
        expect(worker0).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker0).to include("TEST_ENV_NUMBER=1")
        expect(worker0).to include("bin/rails db:migrate --trace")

        worker1 = dry_run_worker_line(output, 1)
        expect(worker1).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker1).to include("TEST_ENV_NUMBER=2")
        expect(worker1).to include("bin/rails db:migrate --trace")
      end
    end
  end

  context "rake alias (dry-run)" do
    it "uses the configured rake job through the Kong alias" do
      Dir.chdir(project_fixture("database-tasks")) do
        output = run_plur("--dry-run", "rake", "db:setup", "-n", "2").err

        expect(output).to include("bundle exec rake db:setup")
        expect(output).to include("PARALLEL_TEST_GROUPS=2")
        expect(output).to include("TEST_ENV_NUMBER=1")
        expect(output).to include("TEST_ENV_NUMBER=2")
      end
    end

    it "passes multiple rake tasks in order" do
      Dir.chdir(project_fixture("database-tasks")) do
        output = run_plur("--dry-run", "rake", "db:create", "db:migrate", "-n", "2").err

        worker0 = dry_run_worker_line(output, 0)
        expect(worker0).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker0).to include("TEST_ENV_NUMBER=1")
        expect(worker0).to include("bundle exec rake db:create db:migrate")

        worker1 = dry_run_worker_line(output, 1)
        expect(worker1).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker1).to include("TEST_ENV_NUMBER=2")
        expect(worker1).to include("bundle exec rake db:create db:migrate")
      end
    end

    it "passes rake-specific flags after a double dash" do
      Dir.chdir(project_fixture("database-tasks")) do
        output = run_plur("--dry-run", "rake", "db:setup", "-n", "2", "--", "--trace").err

        worker0 = dry_run_worker_line(output, 0)
        expect(worker0).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker0).to include("TEST_ENV_NUMBER=1")
        expect(worker0).to include("bundle exec rake db:setup --trace")

        worker1 = dry_run_worker_line(output, 1)
        expect(worker1).to include("PARALLEL_TEST_GROUPS=2")
        expect(worker1).to include("TEST_ENV_NUMBER=2")
        expect(worker1).to include("bundle exec rake db:setup --trace")
      end
    end

    it "passes rake-specific flags without requiring a task name" do
      Dir.chdir(project_fixture("database-tasks")) do
        output = run_plur("--dry-run", "rake", "-n", "1", "--", "--tasks").err

        worker0 = dry_run_worker_line(output, 0)
        expect(worker0).to include("PARALLEL_TEST_GROUPS=1")
        expect(worker0).to include("bundle exec rake --tasks")
        expect(worker0).not_to include("TEST_ENV_NUMBER")
      end
    end
  end

  context "rails database commands with default rails fixture project" do
    def ensure_default_rails_bundle_installed
      _, _, check_status = Open3.capture3("bundle check")
      return if check_status.success?

      _, stderr, status = Open3.capture3("bundle install")
      expect(status.exitstatus).to eq(0), "bundle install failed: #{stderr}"
    end

    def sqlite_tables(db_path)
      `sqlite3 #{db_path} ".tables"`.split
    end

    it "creates and migrates databases for parallel testing" do
      Dir.chdir(default_rails_dir) do
        Bundler.with_unbundled_env do
          ensure_default_rails_bundle_installed

          env = {"RAILS_ENV" => "test"}
          result = run_plur("rails", "db:create", "-n", "3", allow_error: true, env: env)
          expect(result.status).to eq(0), "rails db:create failed: #{result.err}"

          expect(File.exist?("storage/test1.sqlite3")).to be true
          expect(File.exist?("storage/test2.sqlite3")).to be true
          expect(File.exist?("storage/test3.sqlite3")).to be true

          result = run_plur("rails", "db:migrate", "-n", "3", env: env)
          expect(result.exit_status).to eq(0), "rails db:migrate failed: #{result.err}"
        end
      end
    end

    it "passes VERSION=... through to bin/rails db:migrate for each worker" do
      Dir.chdir(default_rails_dir) do
        Bundler.with_unbundled_env do
          ensure_default_rails_bundle_installed

          env = {"RAILS_ENV" => "test"}
          worker_count = 4
          test_dbs = (1..worker_count).map { |i| "storage/test#{i}.sqlite3" }
          test_dbs.each { |db| FileUtils.rm_f(db) }

          create_result = run_plur("rails", "db:create", "-n", worker_count.to_s, env: env)
          expect(create_result.exit_status).to eq(0), "rails db:create failed: #{create_result.err}"

          # Migrate up only to the first migration. If VERSION= got dropped or
          # mangled, all migrations would run and `posts` would appear.
          first_version = "20250522091529"
          partial = run_plur("rails", "db:migrate", "VERSION=#{first_version}", "-n", worker_count.to_s, env: env)
          expect(partial.exit_status).to eq(0), "rails db:migrate VERSION=#{first_version} failed: #{partial.err}"

          test_dbs.each do |db|
            tables = sqlite_tables(db)
            expect(tables).to include("users"), "expected users table in #{db}, got: #{tables.inspect}"
            expect(tables).not_to include("posts"), "expected posts NOT in #{db} after VERSION=#{first_version}, got: #{tables.inspect}"
          end

          # Then bring everything up to the latest migration; posts should appear.
          full = run_plur("rails", "db:migrate", "-n", worker_count.to_s, env: env)
          expect(full.exit_status).to eq(0), "rails db:migrate (full) failed: #{full.err}"

          test_dbs.each do |db|
            tables = sqlite_tables(db)
            expect(tables).to include("users", "posts"), "expected users and posts in #{db}, got: #{tables.inspect}"
          end
        end
      end
    end
  end

  context "verbose mode" do
    it "logs the full command when --verbose is used" do
      Dir.chdir(default_rails_dir) do
        result = run_plur("--verbose", "rails", "db:migrate", "-n", "2", allow_error: true, env: {"RAILS_ENV" => "test"})

        expect(result.err).to include("INFO")
        expect(result.err).to include("running")
        expect(result.err).to include("RAILS_ENV=test")
        expect(result.err).to include("bin/rails db:migrate")
      end
    end
  end

  describe "rake command error handling" do
    it "shows stderr output when worker commands fail" do
      Dir.chdir(project_fixture("database-tasks")) do
        env = {"DB_TASK_FILE" => "lib/tasks/multi_db_connection_error.rake"}
        result = run_plur("rake", "db:setup", "-n", "2", allow_error: true, env: env)

        expect(result.status).to eq(1)
        expect(result.err).to include("Error: Database connection failed")
        expect(result.err).to include("Could not connect to server: Connection refused")
        expect(result.err).to include("Is the server running on host 'localhost'")
      end
    end

    it "shows duplicate worker errors without hiding task output" do
      Dir.chdir(project_fixture("database-tasks")) do
        env = {"DB_TASK_FILE" => "lib/tasks/multi_db_duplicate_errors.rake"}
        result = run_plur("rake", "db:setup", "-n", "3", allow_error: true, env: env)

        expect(result.status).to eq(1)
        expect(result.err).to include("Error: PostgreSQL is not installed")
        expect(result.err).to include("Please install PostgreSQL and try again")
      end
    end

    it "shows different worker errors separately" do
      Dir.chdir(project_fixture("database-tasks")) do
        env = {"DB_TASK_FILE" => "lib/tasks/multi_db_different_errors.rake"}
        result = run_plur("rake", "db:setup", "-n", "3", allow_error: true, env: env)

        expect(result.status).to eq(1)
        expect(result.err).to include("Database test1 already exists")
        expect(result.err).to include("Permission denied for database test2")
        expect(result.out).to include("Successfully created test3")
      end
    end
  end
end
