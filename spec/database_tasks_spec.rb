require "spec_helper"
require "tmpdir"

RSpec.describe "Rux database tasks" do
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }
  let(:test_app_path) { File.join(__dir__, "..", "test_app") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
    expect(File.exist?(test_app_path)).to be(true)
  end

  describe "db:setup command" do
    it "runs db:setup in dry-run mode for Rails app" do
      Dir.chdir(test_app_path) do
        output = `#{rux_binary} db:setup --dry-run -n 3 2>&1`

        expect(output).to include("[dry-run] Would run database task 'db:setup' with 3 workers")
        expect(output).to include("[dry-run] Worker 0: RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:setup")
        expect(output).to include("[dry-run] Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:setup")
      end
    end

    it "shows completion message when running actual task" do
      # Use a simple Rakefile in a temp dir to avoid actually running db:setup on test_app
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "Rakefile"), <<~RUBY)
          task "db:setup" do
            puts "DB setup for \#{ENV['TEST_ENV_NUMBER'] || '1'}"
          end
        RUBY

        Dir.chdir(tmpdir) do
          output = `#{rux_binary} db:setup -n 2 2>&1`

          expect(output).to include("Running database task 'db:setup' with 2 workers...")
          expect(output).to include("Database task 'db:setup' completed successfully")
        end
      end
    end
  end

  describe "db:create command" do
    it "runs db:create in dry-run mode" do
      Dir.chdir(test_app_path) do
        output = `#{rux_binary} db:create --dry-run -n 2 2>&1`

        expect(output).to include("[dry-run] Would run database task 'db:create' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:create")
      end
    end
  end

  describe "db:migrate command" do
    it "runs db:migrate in dry-run mode" do
      Dir.chdir(test_app_path) do
        output = `#{rux_binary} db:migrate --dry-run -n 2 2>&1`

        expect(output).to include("[dry-run] Would run database task 'db:migrate' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:migrate")
      end
    end
  end

  describe "db:test:prepare command" do
    it "runs db:test:prepare in dry-run mode" do
      Dir.chdir(test_app_path) do
        output = `#{rux_binary} db:test:prepare --dry-run -n 2 2>&1`

        expect(output).to include("[dry-run] Would run database task 'db:test:prepare' with 2 workers")
        expect(output).to include("RAILS_ENV=test bundle exec rake db:test:prepare")
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
          output = `#{rux_binary} db:setup -n 3 2>&1`

          expect(output).to include("Running database task 'db:setup' with 3 workers...")
          expect(output).to include("Error: database task failed:")
          expect(output).to include("worker 1 failed: exit status 1")

          # Should exit with error
          expect($?.exitstatus).to eq(1)
        end
      end
    end
  end
end
