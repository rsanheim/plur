require "spec_helper"

RSpec.describe "plur config:init command" do
  let(:test_dir) { Dir.mktmpdir }

  describe "creating local config files" do
    it "creates a simple config file by default" do
      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init")
      end

      expect(status).to be_success
      expect(output).to include("Created .plur.toml with simple template")

      config_content = File.read(File.join(test_dir, ".plur.toml"))
      expect(config_content).to include("workers = 4")
      expect(config_content).to include("command = \"bundle exec rspec\"")
    end

    it "creates a Rails config with --template=rails" do
      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init", "--template=rails")
      end

      expect(status).to be_success
      expect(output).to include("Created .plur.toml with rails template")

      config_content = File.read(File.join(test_dir, ".plur.toml"))
      expect(config_content).to include("workers = 8")
      expect(config_content).to include("bin/spring rspec")
    end

    it "creates a Minitest config with --template=minitest" do
      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init", "--template=minitest")
      end

      expect(status).to be_success
      expect(output).to include("Created .plur.toml with minitest template")

      config_content = File.read(File.join(test_dir, ".plur.toml"))
      expect(config_content).to include("type = \"minitest\"")
      expect(config_content).to include("bundle exec ruby -Itest")
    end
  end

  describe "error handling" do
    it "fails if config file already exists" do
      # Create existing config
      File.write(File.join(test_dir, ".plur.toml"), "existing = true")

      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init")
      end

      expect(status).not_to be_success
      expect(error).to include("already exists")
      expect(error).to include("use --force to overwrite")
    end

    it "overwrites existing config with --force" do
      # Create existing config
      File.write(File.join(test_dir, ".plur.toml"), "existing = true")

      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init", "--force")
      end

      expect(status).to be_success
      expect(output).to include("Created .plur.toml")

      # Should have new content, not old
      config_content = File.read(File.join(test_dir, ".plur.toml"))
      expect(config_content).not_to include("existing = true")
      expect(config_content).to include("workers = 4")
    end

    it "fails with unknown template" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init", "--template=unknown")
      end

      expect(status).not_to be_success
      expect(error).to include("unknown template")
      expect(error).to include("available: simple, rails, minitest")
    end
  end

  describe "global config creation" do
    let(:home_dir) { Dir.mktmpdir }

    around do |example|
      original_home = ENV["HOME"]
      ENV["HOME"] = home_dir
      example.run
    ensure
      ENV["HOME"] = original_home
    end

    it "creates config in home directory with --global" do
      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "config:init", "--global")
      end

      expect(status).to be_success
      expect(output).to include("Created #{home_dir}/.plur.toml")

      # Config should be in home dir, not current dir
      expect(File.exist?(File.join(home_dir, ".plur.toml"))).to be true
      expect(File.exist?(File.join(test_dir, ".plur.toml"))).to be false
    end
  end
end
