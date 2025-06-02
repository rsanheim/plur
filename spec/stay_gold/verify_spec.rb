require "stay_gold"

RSpec.describe "StayGold verify functionality" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "stay_gold") }
  
  describe "basic verification" do
    before do
      # First, record a command
      StayGold.record(record_as: "echo_verify") do
        Open3.capture3("echo hello")
      end
    end
    
    it "passes when output matches recorded cassette" do
      result = StayGold.verify(cassette: "echo_verify") do
        Open3.capture3("echo hello")
      end
      
      expect(result.verified?).to be true
      expect(result.output).to eq("hello\n")
    end
    
    it "fails when output differs from recorded cassette" do
      result = StayGold.verify(cassette: "echo_verify") do
        Open3.capture3("echo goodbye")
      end
      
      expect(result.verified?).to be false
      expect(result.expected_output).to eq("hello\n")
      expect(result.actual_output).to eq("goodbye\n")
      expect(result.diff).to include("-hello")
      expect(result.diff).to include("+goodbye")
    end
    
    it "verifies stderr and exit status too" do
      # Record a command with stderr
      StayGold.record(record_as: "stderr_test") do
        Open3.capture3("sh -c 'echo error >&2; exit 1'")
      end
      
      # Verify matching stderr and status
      result = StayGold.verify(cassette: "stderr_test") do
        Open3.capture3("sh -c 'echo error >&2; exit 1'")
      end
      
      expect(result.verified?).to be true
      
      # Verify non-matching stderr
      result = StayGold.verify(cassette: "stderr_test") do
        Open3.capture3("sh -c 'echo different >&2; exit 1'")
      end
      
      expect(result.verified?).to be false
      expect(result.stderr_diff).to include("-error")
      expect(result.stderr_diff).to include("+different")
    end
  end
  
  describe "verification modes" do
    it "can run in strict mode (default) - must match exactly" do
      StayGold.record(record_as: "timestamp_test") do
        Open3.capture3("date +%s%N") # nanoseconds since epoch
      end
      
      sleep 0.01 # Small delay to ensure different timestamp
      
      # Running date again will produce different output
      result = StayGold.verify(cassette: "timestamp_test", mode: :strict) do
        Open3.capture3("date +%s%N")
      end
      
      expect(result.verified?).to be false
    end
    
    it "can run in playback mode - returns recorded output without running command" do
      StayGold.record(record_as: "playback_test") do
        Open3.capture3("echo original")
      end
      
      result = StayGold.verify(cassette: "playback_test", mode: :playback) do
        # This would normally output "different" but playback mode should return "original"
        Open3.capture3("echo different")
      end
      
      expect(result.verified?).to be true
      expect(result.output).to eq("original\n")
      expect(result.command_executed?).to be false
    end
    
    it "can use custom matchers for flexible verification" do
      StayGold.record(record_as: "version_test") do
        Open3.capture3("ruby --version")
      end
      
      result = StayGold.verify(cassette: "version_test", 
                              matcher: ->(recorded, actual) { 
                                recorded["stdout"].start_with?("ruby") && actual["stdout"].start_with?("ruby")
                              }) do
        Open3.capture3("ruby --version")
      end
      
      expect(result.verified?).to be true
    end
  end
  
  describe "error handling" do
    it "raises error when cassette doesn't exist" do
      expect {
        StayGold.verify(cassette: "nonexistent") do
          Open3.capture3("echo test")
        end
      }.to raise_error(StayGold::CassetteNotFoundError, /nonexistent.yaml/)
    end
    
    it "provides helpful error messages on verification failure" do
      StayGold.record(record_as: "failure_test") do
        Open3.capture3("echo expected")
      end
      
      result = StayGold.verify(cassette: "failure_test") do
        Open3.capture3("echo actual")
      end
      
      expect(result.error_message).to include("Output verification failed")
      expect(result.error_message).to include("Expected: expected")
      expect(result.error_message).to include("Actual: actual")
    end
  end
  
  describe "auto-cassette naming with verify" do
    it "uses auto-generated cassette name when not specified" do
      # Record with auto-generated name
      StayGold.record do
        Open3.capture3("echo auto")
      end
      
      # Verify using same auto-generated name
      result = StayGold.verify do
        Open3.capture3("echo auto")
      end
      
      expect(result.verified?).to be true
    end
  end
  
  after do
    FileUtils.rm_rf(cassette_dir)
  end
end