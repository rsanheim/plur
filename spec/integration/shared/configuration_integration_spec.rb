require "spec_helper"

RSpec.describe "Configuration integration" do
  let(:config_fixture_dir) { project_fixture("config-test") }

  describe "configuration file loading" do
    context "with valid config file" do
      it "loads configuration from specified file" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "valid.toml"},
            "plur", "--dry-run"
          )
        end

        expect(status).to be_success
        # Config has color = false
        expect(error).to include("--no-color")
        # Config has workers = 2
        expect(error).to include("Worker 0:")
        expect(error).to include("Worker 1:")
        expect(error).not_to include("Worker 2:")
      end
    end

    context "with invalid TOML syntax" do
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

    context "with nested configuration sections" do
      it "applies command-specific configuration" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "command-specific.toml"},
            "plur", "spec", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("echo 'SPEC:'")
      end
    end
  end

  describe "configuration precedence" do
    it "CLI flags override configuration file" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "valid.toml"},
          "plur", "--workers=1", "--color", "--dry-run"
        )
      end

      expect(status).to be_success
      # CLI flag --color should override config's color=false
      expect(error).to match(/--force-color|--color/)
      # CLI flag --workers=1 should override config's workers=2
      expect(error).not_to include("Worker 1:")
    end

    it "environment variables have lower precedence than config files" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {
            "PLUR_CONFIG_FILE" => "valid.toml",
            "PARALLEL_TEST_PROCESSORS" => "16"
          },
          "plur", "--dry-run"
        )
      end

      expect(status).to be_success
      # Config file value (2) should win over env var (16)
      expect(error).to include("Worker 0:")
      expect(error).to include("Worker 1:")
      expect(error).not_to include("Worker 2:")
    end
  end

  describe "watch mode configuration" do
    it "applies watch-specific configuration" do
      # Skip watch tests if CI since they need the watcher binary
      skip "Watch tests require watcher binary" if ENV["CI"]

      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "command-specific.toml"},
          "plur", "watch", "run", "--timeout=1", "--debug", stdin_data: ""
        )
      end

      expect(status).to be_success
      # Check that watch mode uses the configured debounce
      expect(error).to include("debounce=75")
    end
  end

  describe "minitest configuration" do
    it "respects minitest type configuration" do
      # Create a minitest config
      minitest_config = config_fixture_dir.join("minitest.toml")
      File.write(minitest_config, <<~TOML)
        [spec]
        type = "minitest"
      TOML

      _, error, status = Dir.chdir(project_fixture("minitest-success")) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => minitest_config.to_s},
          "plur", "spec", "--dry-run"
        )
      end

      expect(status).to be_success
      expect(error).to include("bundle exec ruby -Itest")
      # Should find test files, not spec files
      expect(error).to match(/test.*\.rb/)
    end
  end

  describe "doctor command" do
    it "runs successfully with PLUR_CONFIG_FILE" do
      output, _, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "doctor-test.toml"},
          "plur", "doctor"
        )
      end

      expect(status).to be_success
      expect(output).to include("Plur Doctor")
      expect(output).to include("Configuration:")
      # Doctor shows what's on disk, not what's loaded via env var
      expect(output).to include("Using defaults")
    end
  end

  describe "task configuration loading" do
    context "with TOML task definitions" do
      it "loads custom task configurations" do
        _, output, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "with-tasks.toml"},
            "plur", "spec", "--dry-run", "--command=custom-override"
          )
        end

        expect(status).to be_success
        # Should use custom command override
        expect(output).to include("custom-override")
      end

      it "applies task-specific run commands from TOML" do
        _, output, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "with-tasks.toml"},
            "plur", "spec", "--dry-run", "--type=custom"
          )
        end

        expect(status).to be_success
        # Should use the custom task's run command
        expect(output).to include("echo 'CUSTOM TASK:'")
      end
    end
  end
end
