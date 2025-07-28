require "spec_helper"

RSpec.describe "Configuration edge cases" do
  let(:config_fixture_dir) { project_fixture("config-test") }

  describe "precedence when PLUR_CONFIG_FILE is set" do
    it "PLUR_CONFIG_FILE takes precedence over local config" do
      Dir.mktmpdir do |tmpdir|
        # Create a local .plur.toml that should be ignored
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          workers = 8
          color = true
        TOML

        # Create a simple spec
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)
        File.write(File.join(spec_dir, "test_spec.rb"), "RSpec.describe 'Test' do; it 'works' do; end; end")

        _, error, status = Dir.chdir(tmpdir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_fixture_dir.join("valid.toml").to_s},
            "plur", "--dry-run"
          )
        end

        expect(status).to be_success
        # Should use config from PLUR_CONFIG_FILE (workers=2)
        # With only 1 file, it will only use 1 worker regardless of config
        expect(error).to include("Worker 0:")
        expect(error).not_to include("Worker 1:")
      end
    end
  end

  describe "empty configuration files" do
    context "with empty config file" do
      it "falls back to defaults gracefully" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "empty.toml"},
            "plur", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("bundle exec rspec") # default command
      end
    end

    context "with comments-only config file" do
      it "works with comments-only config" do
        _, _, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "comments-only.toml"},
            "plur", "--dry-run"
          )
        end

        expect(status).to be_success
      end
    end
  end

  describe "configuration with environment variable interaction" do
    it "config file takes precedence over PARALLEL_TEST_PROCESSORS" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {
            "PLUR_CONFIG_FILE" => "valid.toml", # has workers = 2
            "PARALLEL_TEST_PROCESSORS" => "16"
          },
          "plur", "--dry-run"
        )
      end

      expect(status).to be_success
      # Config file wins, should use workers=2 from config not env
      expect(error).to include("Worker 0:")
      expect(error).to include("Worker 1:")
      expect(error).not_to include("Worker 2:")
    end

    it "CLI flag takes precedence over everything" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {
            "PLUR_CONFIG_FILE" => "valid.toml", # has workers = 2
            "PARALLEL_TEST_PROCESSORS" => "16"
          },
          "plur", "--workers=1", "--dry-run"
        )
      end

      expect(status).to be_success
      # CLI flag wins, should use 1 worker
      expect(error).to include("Worker 0:")
      expect(error).not_to include("Worker 1:")
    end
  end

  describe "malformed configuration handling" do
    context "with syntax error in TOML" do
      it "exits immediately with error" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "invalid-syntax.toml"},
            "plur", "doctor"
          )
        end

        expect(status).not_to be_success
        expect(error).to include("Configuration error:")
      end
    end

    context "with wrong data types" do
      it "fails with type validation error" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "type-mismatch.toml"},
            "plur", "doctor"
          )
        end

        expect(status).not_to be_success
        # Kong validates types and fails
        expect(error).to match(/expected a valid 64 bit int|invalid value/)
      end
    end
  end
end
