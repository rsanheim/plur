require "spec_helper"

RSpec.describe "Configuration" do
  let(:config_fixture_dir) { project_fixture("config-test") }

  describe "file loading" do
    context "with valid config file" do
      it "loads configuration from specified file" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "valid.toml"},
            plur_binary, "--dry-run"
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
            plur_binary, "doctor"
          )
        end

        expect(status).not_to be_success
        expect(error).to include("Configuration error:")
      end
    end

    context "with nested configuration sections" do
      it "applies command-specific configuration with use in config" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "command-specific.toml"},
            plur_binary, "spec", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("echo 'SPEC:'")
      end

      it "applies command-specific configuration with --use CLI flag" do
        _, error, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "job-without-use.toml"},
            plur_binary, "spec", "--use=rspec", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("echo 'SPEC:'")
      end
    end
  end

  describe "precedence" do
    it "CLI flags override configuration file" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "valid.toml"},
          plur_binary, "--workers=1", "--color=always", "--dry-run"
        )
      end

      expect(status).to be_success
      expect(error).to match(/--force-color|--color/)
      expect(error).not_to include("Worker 1:")
    end

    # Worker-count env-var precedence lives in configuration_precedence_spec.rb,
    # which asserts via `plur doctor` (not capped by fixture file count).

    it "project config overrides home config" do
      Dir.mktmpdir do |home_dir|
        Dir.mktmpdir do |project_dir|
          FileUtils.mkdir_p(File.join(project_dir, "spec"))
          8.times do |i|
            File.write(File.join(project_dir, "spec", "example_#{i}_spec.rb"), <<~RUBY)
              RSpec.describe "Example#{i}" do
                it "passes" do
                  expect(1).to eq(1)
                end
              end
            RUBY
          end

          File.write(File.join(home_dir, ".plur.toml"), <<~TOML)
            workers = 6
          TOML

          File.write(File.join(project_dir, ".plur.toml"), <<~TOML)
            workers = 3
          TOML

          result = run_plur(
            "-C", project_dir,
            "--dry-run",
            env: {
              "HOME" => home_dir,
              "PLUR_HOME" => File.join(home_dir, ".plur")
            }
          )

          expect(result).to be_success
          expect(result.err).to include("using 3 workers")
          expect(result.err).not_to include("using 6 workers")
        end
      end
    end

    it "PLUR_CONFIG_FILE takes precedence over local config" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          workers = 8
          color = true
        TOML

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
        expect(error).to include("Worker 0:")
        expect(error).not_to include("Worker 1:")
      end
    end

    it "CLI flag takes precedence over everything" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {
            "PLUR_CONFIG_FILE" => "valid.toml",
            "PARALLEL_TEST_PROCESSORS" => "16"
          },
          plur_binary, "--workers=1", "--dry-run"
        )
      end

      expect(status).to be_success
      expect(error).to include("Worker 0:")
      expect(error).not_to include("Worker 1:")
    end
  end

  describe "PLUR_CONFIG_FILE" do
    it "exits immediately when pointing to nonexistent file" do
      _, error, status = Dir.chdir(project_fixture("default-ruby")) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "/no/such/file.toml"},
          plur_binary, "doctor"
        )
      end

      expect(status).not_to be_success
      expect(error).to include("does not exist or is not readable")
    end
  end

  describe "empty configuration files" do
    it "falls back to defaults with empty config file" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "empty.toml"},
          plur_binary, "--dry-run"
        )
      end

      expect(status).to be_success
      expect(error).to include("bundle exec rspec")
    end

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

  describe "malformed configuration" do
    it "exits with error for syntax errors" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "invalid-syntax.toml"},
          plur_binary, "doctor"
        )
      end

      expect(status).not_to be_success
      expect(error).to include("Configuration error:")
    end

    it "fails with type validation error" do
      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "type-mismatch.toml"},
          plur_binary, "doctor"
        )
      end

      expect(status).not_to be_success
      expect(error).to match(/expected a valid 64 bit int|invalid value|must be one of/)
    end

    it "exits with error for duplicate keys" do
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

    it "exits with error for duplicate tables" do
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

  describe "job validation" do
    it "rejects a job without a command" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "no-cmd.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]

          [job.broken]
          framework = "rspec"
        TOML

        _, error, status = Dir.chdir(project_fixture("default-ruby")) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "doctor"
          )
        end

        expect(status).not_to be_success
        expect(error).to include('job "broken" must define a command')
      end
    end

    it "rejects a job with an unknown framework" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "bad-framework.toml")
        File.write(config_path, <<~TOML)
          [job.bad]
          cmd = ["echo"]
          framework = "unknown"
        TOML

        _, error, status = Dir.chdir(tmpdir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "doctor"
          )
        end

        expect(status).not_to be_success
        expect(error).to include('unknown framework "unknown"')
      end
    end

    it "rejects template tokens in job commands" do
      Dir.mktmpdir do |tmpdir|
        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "example_spec.rb"), "RSpec.describe('x'){ it('works'){} }")

        config_path = File.join(tmpdir, "job-command-template.toml")
        File.write(config_path, <<~TOML)
          use = "custom"

          [job.custom]
          framework = "rspec"
          cmd = ["bin/rspec", "{{target}}"]
          target_pattern = "spec/**/*_spec.rb"
        TOML

        _, error, status = Dir.chdir(tmpdir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "doctor"
          )
        end

        expect(status).not_to be_success
        expect(error).to include('job "custom" command must not contain {{target}} tokens')
      end
    end

    it "validates all jobs, not just the selected one" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "all-validated.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]

          [job.broken]
          framework = "rspec"
        TOML

        _, error, status = Dir.chdir(project_fixture("default-ruby")) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "spec", "--dry-run"
          )
        end

        expect(status).not_to be_success
        expect(error).to include('job "broken" must define a command')
      end
    end
  end

  describe "inherited fields in debug output" do
    it "shows which fields were inherited from builtin defaults" do
      Dir.mktmpdir do |tmpdir|
        config_path = File.join(tmpdir, "cmd-only.toml")
        File.write(config_path, <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]
        TOML

        _, error, status = Dir.chdir(project_fixture("default-ruby")) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => config_path},
            plur_binary, "spec", "--debug", "--dry-run"
          )
        end

        expect(status).to be_success
        expect(error).to include("job inherited defaults")
        expect(error).to include("framework=true")
        expect(error).to include("target_pattern=true")
      end
    end
  end

  describe "watch config validation" do
    it "fails early for spec and doctor through the same config path" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]
          target_pattern = "spec/**/*_spec.rb"

          [[watch]]
          name = "bad-watch"
          source = "spec/**/*_spec.rb"
          jobs = ["missing"]
        TOML

        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "example_spec.rb"), "RSpec.describe('x'){ it('works'){ expect(true).to be(true) } }")

        _, spec_err, spec_status = Dir.chdir(tmpdir) { Open3.capture3(plur_binary, "spec", "--dry-run") }
        _, doctor_err, doctor_status = Dir.chdir(tmpdir) { Open3.capture3(plur_binary, "doctor") }

        expect(spec_status).not_to be_success
        expect(doctor_status).not_to be_success
        expect(spec_err).to include("bad-watch")
        expect(doctor_err).to include("bad-watch")
      end
    end

    it "rejects an unknown template token" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]

          [[watch]]
          name = "bad-template"
          source = "lib/**/*.rb"
          targets = ["spec/{{unknown}}_spec.rb"]
          jobs = ["rspec"]
        TOML

        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "example_spec.rb"), "RSpec.describe('x'){ it('works'){} }")

        _, error, status = Dir.chdir(tmpdir) { Open3.capture3(plur_binary, "doctor") }

        expect(status).not_to be_success
        expect(error).to include("bad-template")
        expect(error).to include("invalid target template")
      end
    end

    it "rejects unclosed template braces" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["bin/rspec"]

          [[watch]]
          name = "unclosed"
          source = "lib/**/*.rb"
          targets = ["spec/{{match"]
          jobs = ["rspec"]
        TOML

        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "example_spec.rb"), "RSpec.describe('x'){ it('works'){} }")

        _, error, status = Dir.chdir(tmpdir) { Open3.capture3(plur_binary, "doctor") }

        expect(status).not_to be_success
        expect(error).to include("unclosed")
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
          cmd = ["echo", "RUN"]
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
              cmd = ["echo", "MULTILINE"],
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
          job = { rspec = { cmd = ["echo", "TRAILING"], framework = "rspec", }, }
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

  describe "command-specific job configuration" do
    let(:test_dir) { Dir.mktmpdir }

    before do
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
        expect(error).to include("echo 'SPEC:'")
        expect(error).not_to include("echo 'WATCH:'")
      end

      it "loads watch mappings from config for watch runs" do
        _, error, status = Dir.chdir(test_dir) do
          Open3.capture3(plur_binary, "watch", "run", "--timeout=1", "--debug", stdin_data: "")
        end

        expect(status).to be_success
        expect(error).to include("custom-watch")
      end
    end

    context "with global command setting" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
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
        expect(error).to include("echo 'GLOBAL:'")
      end
    end
  end

  describe "watch mode configuration" do
    it "applies watch-specific configuration" do
      skip "Watch tests require watcher binary" if ENV["CI"]

      _, error, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "command-specific.toml"},
          plur_binary, "watch", "run", "--timeout=1", "--debug", stdin_data: ""
        )
      end

      expect(status).to be_success
      expect(error).to include("custom-watch")
    end
  end

  describe "minitest configuration" do
    it "respects minitest job configuration" do
      minitest_config = config_fixture_dir.join("minitest.toml")

      _, error, status = Dir.chdir(project_fixture("minitest-success")) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => minitest_config.to_s},
          plur_binary, "spec", "--dry-run"
        )
      end

      expect(status).to be_success
      expect(error).to include("bundle exec ruby -Itest")
      expect(error).to match(/calculator_test/)
    end
  end

  describe "doctor command" do
    it "shows configuration from PLUR_CONFIG_FILE" do
      output, _, status = Dir.chdir(config_fixture_dir) do
        Open3.capture3(
          {"PLUR_CONFIG_FILE" => "doctor-test.toml"},
          plur_binary, "doctor"
        )
      end

      expect(status).to be_success
      expect(output).to include("Plur Doctor")
      expect(output).to include("Configuration:")
      expect(output).to include("Active Configuration Files:")
      expect(output).to match(/doctor-test\.toml \(via PLUR_CONFIG_FILE\)/)
      expect(output).to include("Workers:     4")
      expect(output).to include("Color:       true")
      expect(output).not_to include("Using defaults")
    end
  end

  describe "job configuration loading" do
    context "with TOML job definitions" do
      it "applies job-specific run commands from TOML" do
        _, output, status = Dir.chdir(config_fixture_dir) do
          Open3.capture3(
            {"PLUR_CONFIG_FILE" => "with-tasks.toml"},
            plur_binary, "spec", "--dry-run", "--use=custom"
          )
        end

        expect(status).to be_success
        expect(output).to include("echo CUSTOM TASK:")
      end
    end

    context "when explicitly requesting a non-existent job" do
      it "fails with a clear error message for spec command" do
        _, error, status = Open3.capture3(
          {"PLUR_CONFIG_FILE" => "with-tasks.toml"},
          plur_binary, "-C", config_fixture_dir.to_s, "spec", "--use=nonexistent"
        )

        expect(status).not_to be_success
        expect(error).to include("job 'nonexistent' not found")
        expect(error).to include("rspec")
        expect(error).to include("minitest")
        expect(error).to include("custom")
      end

      it "fails with a clear error message for watch command" do
        skip "Watch tests require watcher binary" if ENV["CI"]

        _, error, status = Open3.capture3(
          {"PLUR_CONFIG_FILE" => "with-tasks.toml"},
          plur_binary, "-C", config_fixture_dir.to_s, "watch", "run", "-u", "nonexistent", "--timeout=1"
        )

        expect(status).not_to be_success
        expect(error).to include("job 'nonexistent' not found")
      end
    end
  end
end
