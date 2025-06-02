require 'spec_helper'
require 'stay_gold'

RSpec.describe "StayGold multi-command support" do
  let(:staygold_path) { ROOT_PATH.join("tmp", "stay_gold") }

  context "recording multiple commands" do
    it "records and replays multiple commands in sequence" do
      cassette_name = "multi_command_test"
      
      # First run: record multiple commands
      StayGold.use_cassette(cassette_name) do
        stdout1, stderr1, status1 = Open3.capture3("echo command1")
        expect(stdout1).to eq("command1\n")
        
        stdout2, stderr2, status2 = Open3.capture3("echo command2")
        expect(stdout2).to eq("command2\n")
        
        stdout3, stderr3, status3 = Open3.capture3("echo command3")
        expect(stdout3).to eq("command3\n")
      end
      
      # Verify cassette was created with array
      cassette_path = staygold_path.join("#{cassette_name}.yaml")
      expect(cassette_path).to exist
      
      cassette_data = YAML.load_file(cassette_path)
      expect(cassette_data).to be_a(Array)
      expect(cassette_data.size).to eq(3)
      expect(cassette_data.map { |cmd| cmd["stdout"] }).to eq(["command1\n", "command2\n", "command3\n"])
      
      # Second run: replay from cassette
      replay_outputs = []
      StayGold.use_cassette(cassette_name) do
        stdout1, stderr1, status1 = Open3.capture3("echo command1")
        replay_outputs << stdout1
        
        stdout2, stderr2, status2 = Open3.capture3("echo command2") 
        replay_outputs << stdout2
        
        stdout3, stderr3, status3 = Open3.capture3("echo command3")
        replay_outputs << stdout3
      end
      
      expect(replay_outputs).to eq(["command1\n", "command2\n", "command3\n"])
    end
    
    it "handles mixed commands with different outputs" do
      cassette_name = "mixed_commands"
      
      StayGold.use_cassette(cassette_name) do
        # Different commands with different outputs
        stdout1, stderr1, status1 = Open3.capture3("echo success")
        expect(stdout1).to eq("success\n")
        expect(status1.exitstatus).to eq(0)
        
        # Command that writes to stderr
        stdout2, stderr2, status2 = Open3.capture3("ruby -e 'STDERR.puts \"error message\"'")
        expect(stderr2).to include("error message")
        expect(status2.exitstatus).to eq(0)
        
        # Command with non-zero exit
        stdout3, stderr3, status3 = Open3.capture3("ruby -e 'exit 42'")
        expect(status3.exitstatus).to eq(42)
      end
      
      # Verify cassette structure
      cassette_path = staygold_path.join("#{cassette_name}.yaml")
      cassette_data = YAML.load_file(cassette_path)
      expect(cassette_data).to be_a(Array)
      expect(cassette_data.size).to eq(3)
      
      expect(cassette_data[0]["stdout"]).to eq("success\n")
      expect(cassette_data[0]["status"]).to eq(0)
      
      expect(cassette_data[1]["stderr"]).to include("error message")
      expect(cassette_data[1]["status"]).to eq(0)
      
      expect(cassette_data[2]["status"]).to eq(42)
    end
    
    it "fails gracefully when replaying with too few recordings" do
      cassette_name = "insufficient_recordings"
      
      # Record only 2 commands
      StayGold.use_cassette(cassette_name) do
        Open3.capture3("echo first")
        Open3.capture3("echo second")
      end
      
      # Try to replay 3 commands - should fail on the third
      expect {
        StayGold.use_cassette(cassette_name) do
          Open3.capture3("echo first")
          Open3.capture3("echo second")  
          Open3.capture3("echo third")  # This should fail
        end
      }.to raise_error(StayGold::CassetteNotFoundError, /No more recordings available/)
    end
    
    it "saves single commands as arrays for consistency" do
      cassette_name = "single_command_array"
      
      # Record single command (should save as array)
      StayGold.use_cassette(cassette_name) do
        stdout, stderr, status = Open3.capture3("echo single")
        expect(stdout).to eq("single\n")
      end
      
      # Verify cassette is array even for single command
      cassette_path = staygold_path.join("#{cassette_name}.yaml")
      cassette_data = YAML.load_file(cassette_path)
      expect(cassette_data).to be_a(Array)
      expect(cassette_data.size).to eq(1)
      expect(cassette_data.first["stdout"]).to eq("single\n")
      
      # Should still replay correctly
      StayGold.use_cassette(cassette_name) do
        stdout, stderr, status = Open3.capture3("echo single")
        expect(stdout).to eq("single\n")
      end
    end
  end
end