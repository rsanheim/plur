require "spec_helper"

RSpec.describe "Command-specific configuration" do
  let(:test_dir) { Dir.mktmpdir }

  before do
    # Create a simple spec file
    spec_dir = File.join(test_dir, "spec")
    FileUtils.mkdir_p(spec_dir)
    File.write(File.join(spec_dir, "example_spec.rb"), <<~RUBY)
      RSpec.describe "Example" do
        it "passes" do
          expect(true).to be true
        end
      end
    RUBY
  end

  context "with command-specific TOML configuration" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        # Global settings
        workers = 2
        
        [spec]
        command = "echo 'SPEC:'"
        
        [watch.run]
        command = "echo 'WATCH:'"
        debounce = 75
      TOML
    end

    it "uses spec-specific command for spec runs" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "spec", "--dry-run")
      end

      expect(status).to be_success
      # Dry-run output goes to stderr
      expect(error).to include("echo 'SPEC:'")
      expect(error).not_to include("echo 'WATCH:'")
    end

    it "uses watch-specific debounce for watch runs" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "watch", "run", "--timeout=1", "--debug", stdin_data: "")
      end

      expect(status).to be_success
      # Debug output should show the configured debounce
      expect(error).to include("debounce=75")
    end

    it "respects watch-specific debounce setting" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "watch", "run", "--timeout=1", "--debug", stdin_data: "")
      end

      expect(status).to be_success
      expect(error).to include("debounce=75")
    end
  end

  context "with CLI flag override" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        [spec]
        command = "echo 'FROM CONFIG:'"
      TOML
    end

    it "CLI flag takes precedence over config file" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "spec", "--command=echo 'FROM CLI:'", "--dry-run")
      end

      expect(status).to be_success
      # Dry-run output goes to stderr
      expect(error).to include("echo 'FROM CLI:'")
      expect(error).not_to include("echo 'FROM CONFIG:'")
    end
  end

  context "with global command setting" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        # Global command applies to all commands
        command = "echo 'GLOBAL:'"
        
        [spec]
        # No command specified here
      TOML
    end

    it "uses global command setting when no command-specific setting exists" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "spec", "--dry-run")
      end

      expect(status).to be_success
      # Global command is inherited by spec command
      # Dry-run output goes to stderr
      expect(error).to include("echo 'GLOBAL:'")
    end
  end
end
