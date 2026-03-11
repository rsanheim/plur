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
            plur_binary, "--dry-run"
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
            plur_binary, "--dry-run"
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
            plur_binary, "--dry-run"
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
          plur_binary, "--dry-run"
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
          plur_binary, "--workers=1", "--dry-run"
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
            plur_binary, "doctor"
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
            plur_binary, "doctor"
          )
        end

        expect(status).not_to be_success
        # Kong validates types and fails
        expect(error).to match(/expected a valid 64 bit int|invalid value/)
      end
    end

    context "with duplicate keys in TOML" do
      it "exits with a configuration parse error" do
        Dir.mktmpdir do |tmpdir|
          config_path = File.join(tmpdir, "duplicate-keys.toml")
          File.write(config_path, <<~TOML)
            workers = 2
            workers = 3
          TOML

          _, error, status = Dir.chdir(project_fixture("default-ruby")) do
            Open3.capture3(
              {"PLUR_CONFIG_FILE" => config_path},
              plur_binary, "doctor"
            )
          end

          expect(status).not_to be_success
          expect(error).to include("Configuration error:")
          expect(error).to match(/already been defined|defined twice/)
        end
      end
    end

    context "with duplicate tables in TOML" do
      it "exits with a configuration parse error" do
        Dir.mktmpdir do |tmpdir|
          config_path = File.join(tmpdir, "duplicate-tables.toml")
          File.write(config_path, <<~TOML)
            [job.rspec]
            cmd = ["echo", "first"]

            [job.rspec]
            cmd = ["echo", "second"]
          TOML

          _, error, status = Dir.chdir(project_fixture("default-ruby")) do
            Open3.capture3(
              {"PLUR_CONFIG_FILE" => config_path},
              plur_binary, "doctor"
            )
          end

          expect(status).not_to be_success
          expect(error).to include("Configuration error:")
          expect(error).to match(/already been defined|duplicated tables|defined twice/)
        end
      end
    end
  end

  describe "resolver edge cases" do
    it "still applies a hyphenated key when a scalar prefix key exists" do
      Dir.mktmpdir do |tmpdir|
        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "test_spec.rb"), "describe('test') { it('works') { expect(1).to eq(1) } }")

        config_path = File.join(tmpdir, "shadowed-dry-run.toml")
        File.write(config_path, <<~TOML)
          use = "custom"
          dry = "shadow"
          dry-run = true

          [job.custom]
          framework = "passthrough"
          cmd = ["echo", "RUN", "{{target}}"]
          target_pattern = "spec/**/*_spec.rb"
        TOML

        _, error, status = Dir.chdir(tmpdir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "spec"
          )
        end

        expect(status).to be_success
        expect(error).to include("[dry-run] Worker 0:")
        expect(error).to include("echo RUN spec/test_spec.rb")
      end
    end

    it "logs unknown nested job keys in debug output" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "unknown-job-key.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]
          cmdd = ["echo", "TYPO"]
        TOML

        _, error, status = Dir.chdir(tmpdir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "--debug", "doctor"
          )
        end

        expect(status).to be_success
        expect(error).to include("unknown config keys")
        expect(error).to include("job.rspec.cmdd")
      end
    end
  end

  describe "TOML 1.1 compatibility" do
    it "supports multiline inline tables for nested job config" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "toml11-multiline-inline.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"
          job = {
            rspec = {
              cmd = ["echo", "MULTILINE", "{{target}}"],
              framework = "rspec"
            }
          }
        TOML

        _, error, status = Dir.chdir(project_fixture("default-ruby")) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "spec", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("echo MULTILINE")
      end
    end

    it "supports trailing commas in inline tables" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "toml11-inline-trailing-comma.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"
          job = { rspec = { cmd = ["echo", "TRAILING", "{{target}}"], framework = "rspec", }, }
        TOML

        _, error, status = Dir.chdir(project_fixture("default-ruby")) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "spec", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("echo TRAILING")
      end
    end
  end
end
