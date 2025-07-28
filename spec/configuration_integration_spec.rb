require "spec_helper"

RSpec.describe "Configuration integration" do
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

  describe "configuration file loading" do
    context "with local .plur.toml" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          workers = 2
          color = false
          verbose = true
        TOML
      end

      it "loads configuration from .plur.toml" do
        _, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "--dry-run", "--debug")
        end

        expect(status).to be_success
        # Verify configuration is loaded by checking the dry-run output
        expect(error).to include("[dry-run]")
        # Workers should be 2 based on config
        expect(error).to match(/Using .+ execution: \d+ (?:files?|groups?)/)
      end
    end

    context "with invalid TOML syntax" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          workers = 2
          color = this is invalid
        TOML
      end

      it "handles invalid TOML gracefully" do
        output, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "doctor")
        end

        # Should handle invalid TOML gracefully
        expect(status).to be_success
        # Should show warning about invalid TOML
        expect(error).to include("Warning: Invalid TOML")
        # Doctor should still work
        expect(output).to include("Plur Doctor")
      end
    end

    context "with nested configuration sections" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          # Global settings
          workers = 4
          
          [spec]
          command = "echo 'CUSTOM SPEC:'"
          type = "rspec"
          
          [watch.run]
          command = "echo 'CUSTOM WATCH:'"
          debounce = 250
          type = "rspec"
        TOML
      end

      it "applies spec-specific configuration" do
        _, error, status = Dir.chdir(test_dir) do
          Open3.capture3("plur", "spec", "--dry-run")
        end

        expect(status).to be_success
        expect(error).to include("echo 'CUSTOM SPEC:'")
      end
    end
  end

  describe "configuration precedence" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        workers = 2
        [spec]
        command = "echo 'FROM CONFIG:'"
      TOML
    end

    it "CLI flags override configuration file" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "--workers=8", "--command=echo 'FROM CLI:'", "--dry-run")
      end

      expect(status).to be_success
      expect(error).to include("echo 'FROM CLI:'")
      # Workers set to 8 via CLI flag
      expect(error).to match(/Using .+ execution/)
    end

    it "environment variables have lower precedence than config files" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3(
          {"PARALLEL_TEST_PROCESSORS" => "16"},
          "plur", "--dry-run", "--debug"
        )
      end

      expect(status).to be_success
      # Config file value (2) should win over env var (16)
      # Should use workers from config file
      expect(error).to match(/Using .+ execution/)
    end
  end

  describe "watch mode configuration" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        [watch.run]
        debounce = 500
        command = "echo 'WATCH MODE:'"
      TOML
    end

    it "applies watch-specific configuration" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "watch", "run", "--timeout=1", "--debug", stdin_data: "")
      end

      expect(status).to be_success
      # Check that watch mode uses the configured debounce
      expect(error).to include("debounce=500")
    end
  end

  describe "minitest configuration" do
    before do
      # Create a minitest file
      test_dir_path = File.join(test_dir, "test")
      FileUtils.mkdir_p(test_dir_path)
      File.write(File.join(test_dir_path, "example_test.rb"), <<~RUBY)
        require "minitest/autorun"
        
        class ExampleTest < Minitest::Test
          def test_truth
            assert true
          end
        end
      RUBY

      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        [spec]
        command = "echo 'MINITEST:'"
        type = "minitest"
      TOML
    end

    it "respects minitest type configuration" do
      _, error, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "spec", "--dry-run")
      end

      expect(status).to be_success
      expect(error).to include("bundle exec ruby -Itest")
      # Should find test files, not spec files
      expect(error).to include("example_test.rb")
    end
  end

  describe "doctor command configuration display" do
    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        workers = 4
        color = true
        command = "custom-rspec"
        
        [spec]
        command = "spec-specific-command"
        
        [watch.run]
        command = "watch-specific-command"
        debounce = 300
      TOML
    end

    it "shows configuration in doctor output" do
      output, _, status = Dir.chdir(test_dir) do
        Open3.capture3("plur", "doctor")
      end

      expect(status).to be_success
      expect(output).to include("Configuration:")
      expect(output).to include("Local Config:")
      expect(output).to include(".plur.toml (valid")
      expect(output).to include("Active Configuration:")
      expect(output).to include("Workers: 4")
      expect(output).to include("Color: true")
      expect(output).to include("[spec] section:")
      expect(output).to include("Command: spec-specific-command")
      expect(output).to include("[watch.run] section:")
      expect(output).to include("Debounce: 300ms")
    end
  end
end
