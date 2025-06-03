require "spec_helper"

RSpec.describe "Backspin multi-command support" do
  let(:backspin_path) { Pathname.new(File.join("tmp", "backspin")) }

  context "recording multiple commands" do
    it "records and replays multiple commands in sequence" do
      dubplate_name = "multi_command_test"

      # First run: record multiple commands
      Backspin.use_dubplate(dubplate_name) do
        stdout1, _, _ = Open3.capture3("echo command1")
        expect(stdout1).to eq("command1\n")

        stdout2, _, _ = Open3.capture3("echo command2")
        expect(stdout2).to eq("command2\n")

        stdout3, _, _ = Open3.capture3("echo command3")
        expect(stdout3).to eq("command3\n")
      end

      # Verify dubplate was created with array
      dubplate_path = backspin_path.join("#{dubplate_name}.yaml")
      expect(dubplate_path).to exist

      dubplate_data = YAML.load_file(dubplate_path)
      expect(dubplate_data).to be_a(Array)
      expect(dubplate_data.size).to eq(3)
      expect(dubplate_data.map { |cmd| cmd["stdout"] }).to eq(["command1\n", "command2\n", "command3\n"])

      # Second run: replay from dubplate
      replay_outputs = []
      Backspin.use_dubplate(dubplate_name) do
        stdout1, _, _ = Open3.capture3("echo command1")
        replay_outputs << stdout1

        stdout2, _, _ = Open3.capture3("echo command2")
        replay_outputs << stdout2

        stdout3, _, _ = Open3.capture3("echo command3")
        replay_outputs << stdout3
      end

      expect(replay_outputs).to eq(["command1\n", "command2\n", "command3\n"])
    end

    it "handles mixed commands with different outputs" do
      dubplate_name = "mixed_commands"

      Backspin.use_dubplate(dubplate_name) do
        # Different commands with different outputs
        stdout1, _, status1 = Open3.capture3("echo success")
        expect(stdout1).to eq("success\n")
        expect(status1.exitstatus).to eq(0)

        # Command that writes to stderr
        _, stderr2, status2 = Open3.capture3("ruby -e 'STDERR.puts \"error message\"'")
        expect(stderr2).to include("error message")
        expect(status2.exitstatus).to eq(0)

        # Command with non-zero exit
        _, _, status3 = Open3.capture3("ruby -e 'exit 42'")
        expect(status3.exitstatus).to eq(42)
      end

      # Verify dubplate structure
      dubplate_path = backspin_path.join("#{dubplate_name}.yaml")
      dubplate_data = YAML.load_file(dubplate_path)
      expect(dubplate_data).to be_a(Array)
      expect(dubplate_data.size).to eq(3)

      expect(dubplate_data[0]["stdout"]).to eq("success\n")
      expect(dubplate_data[0]["status"]).to eq(0)

      expect(dubplate_data[1]["stderr"]).to include("error message")
      expect(dubplate_data[1]["status"]).to eq(0)

      expect(dubplate_data[2]["status"]).to eq(42)
    end

    it "fails gracefully when replaying with too few recordings" do
      dubplate_name = "insufficient_recordings"

      # Record only 2 commands
      Backspin.use_dubplate(dubplate_name) do
        Open3.capture3("echo first")
        Open3.capture3("echo second")
      end

      # Try to replay 3 commands - should fail on the third
      expect {
        Backspin.use_dubplate(dubplate_name) do
          Open3.capture3("echo first")
          Open3.capture3("echo second")
          Open3.capture3("echo third")  # This should fail
        end
      }.to raise_error(Backspin::DubplateNotFoundError, /No more recordings available/)
    end

    it "saves single commands as arrays for consistency" do
      dubplate_name = "single_command_array"

      # Record single command (should save as array)
      Backspin.use_dubplate(dubplate_name) do
        stdout, _, _ = Open3.capture3("echo single")
        expect(stdout).to eq("single\n")
      end

      # Verify dubplate is array even for single command
      dubplate_path = backspin_path.join("#{dubplate_name}.yaml")
      dubplate_data = YAML.load_file(dubplate_path)
      expect(dubplate_data).to be_a(Array)
      expect(dubplate_data.size).to eq(1)
      expect(dubplate_data.first["stdout"]).to eq("single\n")

      # Should still replay correctly
      Backspin.use_dubplate(dubplate_name) do
        stdout, _, _ = Open3.capture3("echo single")
        expect(stdout).to eq("single\n")
      end
    end
  end
end
