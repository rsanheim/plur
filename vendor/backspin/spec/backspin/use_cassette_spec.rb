require "spec_helper"

RSpec.describe "Backspin.use_dubplate" do
  let(:dubplate_dir) { Backspin.configuration.backspin_dir }

  describe "VCR-style unified API" do
    it "records on first run, replays on subsequent runs" do
      # First run - should record
      first_result = Backspin.use_dubplate("unified_test") do
        Open3.capture3("echo hello from use_dubplate")
      end

      expect(first_result).to eq(["hello from use_dubplate\n", "", 0])
      expect(dubplate_dir.join("unified_test.yaml")).to exist

      # Second run - should replay without executing
      second_result = Backspin.use_dubplate("unified_test") do
        Open3.capture3("echo this should not run")
      end

      # Should get the recorded output, not "this should not run"
      expect(second_result).to eq(["hello from use_dubplate\n", "", 0])
    end

    it "requires dubplate name" do
      expect {
        Backspin.use_dubplate do
          Open3.capture3("echo auto-named")
        end
      }.to raise_error(ArgumentError, "wrong number of arguments (given 0, expected 1..2)")
    end

    it "raises ArgumentError when dubplate name is nil" do
      expect {
        Backspin.use_dubplate(nil) do
          Open3.capture3("echo test")
        end
      }.to raise_error(ArgumentError, "dubplate_name is required")
    end

    it "raises ArgumentError when dubplate name is empty" do
      expect {
        Backspin.use_dubplate("") do
          Open3.capture3("echo test")
        end
      }.to raise_error(ArgumentError, "dubplate_name is required")
    end

    it "supports record modes" do
      # Clean up any existing file first
      FileUtils.rm_f(dubplate_dir.join("modes_test.yaml"))

      # Record initially
      Backspin.use_dubplate("modes_test") do
        Open3.capture3("echo first")
      end

      # :once mode (default) - use existing dubplate
      result = Backspin.use_dubplate("modes_test", record: :once) do
        Open3.capture3("echo second")
      end
      expect(result[0]).to eq("first\n")

      # :all mode - always re-record
      result = Backspin.use_dubplate("modes_test", record: :all) do
        Open3.capture3("echo third")
      end
      expect(result[0]).to eq("third\n")

      # Verify it was re-recorded
      result = Backspin.use_dubplate("modes_test") do
        Open3.capture3("echo fourth")
      end
      expect(result[0]).to eq("third\n")
    end

    it "supports :none mode - never record" do
      # Ensure dubplate doesn't exist
      FileUtils.rm_f(dubplate_dir.join("none_mode_test.yaml"))

      expect {
        Backspin.use_dubplate("none_mode_test", record: :none) do
          Open3.capture3("echo test")
        end
      }.to raise_error(Backspin::DubplateNotFoundError)
    end

    it "supports :new_episodes mode - appends new recordings" do
      # Record initial command
      Backspin.use_dubplate("episodes_test", record: :new_episodes) do
        Open3.capture3("echo episode1")
      end

      # Different command - should record new episode
      result = Backspin.use_dubplate("episodes_test", record: :new_episodes) do
        Open3.capture3("echo episode2")
      end
      expect(result[0]).to eq("episode2\n")

      # Verify multiple recordings exist
      dubplate_data = YAML.load_file(dubplate_dir.join("episodes_test.yaml"))
      expect(dubplate_data).to be_a(Array)
      expect(dubplate_data.map { |r| r["stdout"] }).to include("episode1\n", "episode2\n")
    end

    it "returns stdout, stderr, and status like capture3" do
      stdout, stderr, status = Backspin.use_dubplate("full_output_test") do
        Open3.capture3("sh -c 'echo stdout; echo stderr >&2; exit 42'")
      end

      expect(stdout).to eq("stdout\n")
      expect(stderr).to eq("stderr\n")
      expect(status).to eq(42)
    end

    it "supports options hash" do
      result = Backspin.use_dubplate("options_test",
        record: :all,
        erb: true,
        preserve_exact_body_bytes: true) do
        Open3.capture3("echo with options")
      end

      expect(result[0]).to eq("with options\n")
    end
  end

  describe "block return values" do
    it "returns the value from the block" do
      result = Backspin.use_dubplate("return_value_test") do
        stdout, _, _ = Open3.capture3("echo test")
        "custom return: #{stdout.strip}"
      end

      expect(result).to eq("custom return: test")
    end
  end

  describe "error handling" do
    it "preserves exceptions from the block" do
      expect {
        Backspin.use_dubplate("exception_test") do
          raise "Something went wrong"
        end
      }.to raise_error("Something went wrong")
    end
  end
end
