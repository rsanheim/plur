require "spec_helper"

RSpec.describe "plur rails:init command" do
  def create_rails_project(dir, database_yml:)
    FileUtils.mkdir_p(File.join(dir, "config"))
    File.write(File.join(dir, "Gemfile"), "source 'https://rubygems.org'\ngem 'rails'\n")
    File.write(File.join(dir, "config/application.rb"), "# Rails app\n")
    File.write(File.join(dir, "config/database.yml"), database_yml)
  end

  context "with a postgresql Rails project" do
    it "adds TEST_ENV_NUMBER to the test database name" do
      Dir.mktmpdir do |dir|
        create_rails_project(dir, database_yml: <<~YAML)
          default: &default
            adapter: postgresql
            pool: 5

          test:
            <<: *default
            database: myapp_test
        YAML

        Dir.chdir(dir) do
          result = run_plur("rails:init")
          expect(result.out).to include("Updated config/database.yml")

          content = File.read("config/database.yml")
          expect(content).to include("myapp_test<%= ENV['TEST_ENV_NUMBER'] %>")
        end
      end
    end
  end

  context "with multi-database configuration" do
    it "adds TEST_ENV_NUMBER to all database names in the test section" do
      Dir.mktmpdir do |dir|
        create_rails_project(dir, database_yml: <<~YAML)
          default: &default
            adapter: postgresql
            pool: 5

          test:
            primary:
              <<: *default
              database: myapp_test
            cache:
              <<: *default
              database: myapp_test_cache
        YAML

        Dir.chdir(dir) do
          result = run_plur("rails:init")
          expect(result.out).to include("Updated config/database.yml")

          content = File.read("config/database.yml")
          expect(content).to include("myapp_test<%= ENV['TEST_ENV_NUMBER'] %>")
          expect(content).to include("myapp_test_cache<%= ENV['TEST_ENV_NUMBER'] %>")
        end
      end
    end
  end

  context "idempotency" do
    it "detects already-configured projects and makes no changes" do
      Dir.mktmpdir do |dir|
        create_rails_project(dir, database_yml: <<~YAML)
          default: &default
            adapter: postgresql
            pool: 5

          test:
            <<: *default
            database: myapp_test
        YAML

        Dir.chdir(dir) do
          run_plur("rails:init")
          first_content = File.read("config/database.yml")

          result = run_plur("rails:init")
          second_content = File.read("config/database.yml")

          expect(first_content).to eq(second_content)
          expect(result.out).to include("already configured")
        end
      end
    end
  end

  context "dry-run mode" do
    it "shows proposed changes without modifying files" do
      Dir.mktmpdir do |dir|
        db_yml = <<~YAML
          default: &default
            adapter: postgresql
            pool: 5

          test:
            <<: *default
            database: myapp_test
        YAML

        create_rails_project(dir, database_yml: db_yml)

        Dir.chdir(dir) do
          result = run_plur("--dry-run", "rails:init")

          expect(result.out).to include("[dry-run]")
          expect(result.out).to include("TEST_ENV_NUMBER")
          expect(result.out).to include("Run 'plur rails:init' without --dry-run")

          current_content = File.read("config/database.yml")
          expect(current_content).to eq(db_yml)
        end
      end
    end
  end

  context "outside a Rails project" do
    it "fails with a clear error" do
      Dir.mktmpdir do |dir|
        Dir.chdir(dir) do
          result = run_plur("rails:init", allow_error: true)
          expect(result.exit_status).not_to eq(0)
          expect(result.err).to include("config/database.yml not found")
        end
      end
    end
  end

  context "next steps" do
    it "shows follow-up commands after making changes" do
      Dir.mktmpdir do |dir|
        create_rails_project(dir, database_yml: <<~YAML)
          default: &default
            adapter: postgresql
            pool: 5

          test:
            <<: *default
            database: myapp_test
        YAML

        Dir.chdir(dir) do
          result = run_plur("rails:init")
          expect(result.out).to include("plur db:create")
          expect(result.out).to include("plur db:migrate")
          expect(result.out).to include("plur spec")
        end
      end
    end
  end

  context "warnings" do
    it "warns about .env.test files with service URLs" do
      Dir.mktmpdir do |dir|
        create_rails_project(dir, database_yml: <<~YAML)
          test:
            database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>
        YAML

        File.write(File.join(dir, ".env.test"), "REDIS_URL=redis://localhost:6379/0\n")

        Dir.chdir(dir) do
          result = run_plur("rails:init")
          expect(result.out).to include(".env.test")
          expect(result.out).to include("service URLs")
        end
      end
    end
  end
end
