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
        use = "rspec"

        [job.rspec]
        cmd = ["echo", "'SPEC:'"]
        target_pattern = "spec/**/*_spec.rb"

        [[watch]]
        name = "custom-watch"
        source = "spec/**/*_custom_spec.rb"
        jobs = ["rspec"]
      TOML
    end

    it "uses spec-specific command for spec runs" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(plur_binary, "spec", "--dry-run")
      end

      expect(status).to be_success
      # Dry-run output goes to stderr
      expect(error).to include("echo 'SPEC:'")
      expect(error).not_to include("echo 'WATCH:'")
    end

    it "loads watch mappings from config for watch runs" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(plur_binary, "watch", "run", "--timeout=1", "--debug", stdin_data: "")
      end

      expect(status).to be_success
      # Debug output should include the custom watch mapping
      expect(error).to include("custom-watch")
    end
  end

  context "with global command setting" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        # Job command applies to RSpec job
        use = "rspec"

        [job.rspec]
        cmd = ["echo", "'GLOBAL:'"]
        target_pattern = "spec/**/*_spec.rb"
      TOML
    end

    it "uses global command setting when no command-specific setting exists" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(plur_binary, "spec", "--dry-run")
      end

      expect(status).to be_success
      # Global command is inherited by spec command
      # Dry-run output goes to stderr
      expect(error).to include("echo 'GLOBAL:'")
    end
  end
end
