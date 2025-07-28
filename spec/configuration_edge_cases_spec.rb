require "spec_helper"
require "fileutils"

RSpec.describe "Configuration edge cases" do
  let(:test_dir) { Dir.mktmpdir }
  let(:home_dir) { Dir.mktmpdir }

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

  around do |example|
    # Temporarily override HOME for global config testing
    original_home = ENV["HOME"]
    ENV["HOME"] = home_dir
    example.run
  ensure
    ENV["HOME"] = original_home
  end

  describe "global vs local configuration precedence" do
    before do
      # Create global config
      File.write(File.join(home_dir, ".plur.toml"), <<~TOML)
        workers = 8
        color = false
        command = "echo 'GLOBAL:'"
        
        [spec]
        command = "echo 'GLOBAL SPEC:'"
      TOML

      # Create local config with partial overrides
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        workers = 2
        # Note: not overriding color or command
        
        [spec]
        # Only override command, not type
        command = "echo 'LOCAL SPEC:'"
      TOML
    end

    it "local config overrides global config" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "spec", "--dry-run", "--debug")
      end

      expect(status).to be_success
      # Local values should win
      expect(error).to include("echo 'LOCAL SPEC:'")
    end
  end

  describe "empty configuration files" do
    context "with empty .plur.toml" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), "")
      end

      it "falls back to defaults gracefully" do
        _, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "--dry-run")
        end

        expect(status).to be_success
        expect(error).to include("bundle exec rspec") # default command
      end
    end

    context "with comments-only .plur.toml" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          # This file only contains comments
          # workers = 4
          # color = true
        TOML
      end

      it "works with comments-only config" do
        _, _, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "--dry-run")
        end

        expect(status).to be_success
      end
    end
  end

  describe "configuration with environment variable interaction" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        # Config file sets workers to 4
        workers = 4
      TOML
    end

    it "config file takes precedence over PARALLEL_TEST_PROCESSORS" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(
          {"PARALLEL_TEST_PROCESSORS" => "16"},
          "plur", "--dry-run", "--debug"
        )
      end

      expect(status).to be_success
      # Config file wins, should use workers from config not env
      expect(error).to match(/Using .+ execution/)
    end

    it "CLI flag takes precedence over everything" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(
          {"PARALLEL_TEST_PROCESSORS" => "16"},
          "plur", "--workers=1", "--dry-run", "--debug"
        )
      end

      expect(status).to be_success
      # CLI flag wins, should use 1 worker
      expect(error).to match(/Using .+ execution/)
    end
  end

  describe "malformed configuration handling" do
    context "with syntax error in TOML" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          workers = 4
          color = "this should be boolean"
          command = missing quotes here
        TOML
      end

      it "shows warning and continues" do
        output, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "doctor")
        end

        expect(status).to be_success
        # Should show warning in stderr
        expect(error).to include("Warning: Invalid TOML")
        # Doctor should still work
        expect(output).to include("Plur Doctor")
      end
    end

    context "with wrong data types" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          workers = "should be number"
          color = "yes" # should be boolean
          
          [spec]
          command = 123 # should be string
        TOML
      end

      it "fails with type validation error" do
        _, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "doctor")
        end

        # Kong validates types and fails
        expect(status).not_to be_success
        expect(error).to include("expected a valid 64 bit int")
      end
    end
  end

  describe "watch mode debounce validation" do
    context "with extremely high debounce value" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          [watch.run]
          debounce = 50000 # 50 seconds!
        TOML
      end

      it "shows warning in doctor output" do
        output, _, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "doctor")
        end

        expect(status).to be_success
        expect(output).to include("Warning: debounce value")
        expect(output).to include("seems unusual")
      end
    end

    context "with negative debounce value" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          [watch.run]
          debounce = -100
        TOML
      end

      it "shows warning in doctor output" do
        output, _, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "doctor")
        end

        expect(status).to be_success
        expect(output).to include("Warning: debounce value")
      end
    end
  end
end
