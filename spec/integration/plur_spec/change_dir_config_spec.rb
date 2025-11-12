require_relative "../../spec_helper"

RSpec.describe "plur -C with config files", type: :integration do
  let(:temp_dir) { Dir.mktmpdir }
  let(:project_dir) { File.join(temp_dir, "test_project") }

  before do
    # Create a test project structure
    FileUtils.mkdir_p(File.join(project_dir, "spec"))

    # Create multiple spec files to test worker count
    File.write(File.join(project_dir, "spec", "example_spec.rb"), <<~RUBY)
      RSpec.describe "Example" do
        it "passes" do
          expect(true).to be true
        end
      end
    RUBY

    File.write(File.join(project_dir, "spec", "another_spec.rb"), <<~RUBY)
      RSpec.describe "Another" do
        it "also passes" do
          expect(true).to be true
        end
      end
    RUBY

    File.write(File.join(project_dir, "spec", "third_spec.rb"), <<~RUBY)
      RSpec.describe "Third" do
        it "passes too" do
          expect(true).to be true
        end
      end
    RUBY

    # Create Gemfile
    File.write(File.join(project_dir, "Gemfile"), <<~GEMFILE)
      source 'https://rubygems.org'
      gem 'rspec', '~> 3.12'
    GEMFILE
  end

  after do
    FileUtils.rm_rf(temp_dir)
  end

  context "when .plur.toml exists in target directory" do
    before do
      File.write(File.join(project_dir, ".plur.toml"), <<~TOML)
        use = "rspec"
        workers = 2

        [job.rspec]
        cmd = ["rspec"]
        target_pattern = "spec/**/*_spec.rb"
      TOML
    end

    it "loads config from the target directory when using -C" do
      # Run from temp_dir (parent), using -C to change to project_dir
      Dir.chdir(temp_dir) do
        result = run_plur("-C", "test_project", "--dry-run")

        # Should use "rspec" from config, not default "bundle exec rspec"
        expect(result.err).to include("rspec")
        expect(result.err).not_to include("bundle exec")

        # Should use 2 workers from config (check in the grouping output)
        expect(result.err).to include("2 groups")
      end
    end

    it "ignores config from current directory when using -C" do
      # Create different config in current directory
      File.write(File.join(temp_dir, ".plur.toml"), <<~TOML)
        command = "bundle exec rspec --format progress"
        workers = 8
      TOML

      Dir.chdir(temp_dir) do
        result = run_plur("-C", "test_project", "--dry-run")

        # Should use config from test_project, not from temp_dir
        expect(result.err).to include("rspec")
        expect(result.err).not_to include("--format progress")
        expect(result.err).to include("2 groups")
        expect(result.err).not_to include("8 groups")
      end
    end
  end

  context "when no config exists in target directory" do
    it "uses defaults when no config is found" do
      Dir.chdir(temp_dir) do
        result = run_plur("-C", "test_project", "--dry-run")

        # Should use default command
        expect(result.err).to include("bundle exec rspec")
      end
    end
  end

  context "with different -C flag formats" do
    before do
      File.write(File.join(project_dir, ".plur.toml"), <<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["rspec"]
      TOML
    end

    it "works with -C dir format" do
      Dir.chdir(temp_dir) do
        result = run_plur("-C", "test_project", "--dry-run")
        expect(result.err).to include("rspec")
        expect(result.err).not_to include("bundle exec")
      end
    end

    it "works with -C=dir format" do
      Dir.chdir(temp_dir) do
        result = run_plur("-C=test_project", "--dry-run")
        expect(result.err).to include("rspec")
        expect(result.err).not_to include("bundle exec")
      end
    end

    it "works with --change-dir=dir format" do
      Dir.chdir(temp_dir) do
        result = run_plur("--change-dir=test_project", "--dry-run")
        expect(result.err).to include("rspec")
        expect(result.err).not_to include("bundle exec")
      end
    end
  end

  context "error handling" do
    it "fails gracefully when directory doesn't exist" do
      Dir.chdir(temp_dir) do
        result = run_plur_allowing_errors("-C", "nonexistent", "--dry-run")
        expect(result.err).to include("failed to change directory")
        expect(result.exit_status).not_to eq(0)
      end
    end

    it "fails when -C is provided without argument" do
      Dir.chdir(temp_dir) do
        result = run_plur_allowing_errors("-C")
        expect(result.err).to include("requires a directory argument")
        expect(result.exit_status).not_to eq(0)
      end
    end
  end
end
