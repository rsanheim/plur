require "backspin"

RSpec.describe "Backspin.use_cassette" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "backspin") }
  
  describe "VCR-style unified API" do
    it "records on first run, replays on subsequent runs" do
      # First run - should record
      first_result = Backspin.use_cassette("unified_test") do
        Open3.capture3("echo hello from use_cassette")
      end
      
      expect(first_result).to eq(["hello from use_cassette\n", "", 0])
      expect(cassette_dir.join("unified_test.yaml")).to exist
      
      # Second run - should replay without executing
      second_result = Backspin.use_cassette("unified_test") do
        Open3.capture3("echo this should not run")
      end
      
      # Should get the recorded output, not "this should not run"
      expect(second_result).to eq(["hello from use_cassette\n", "", 0])
    end
    
    it "supports auto-generated cassette names" do
      result = Backspin.use_cassette do
        Open3.capture3("echo auto-named")
      end
      
      expect(result[0]).to eq("auto-named\n")
      
      # Verify cassette was created with auto-generated name
      auto_cassette = Dir.glob(cassette_dir.join("**/*.yaml")).find do |f|
        f.include?("supports_auto-generated_cassette_names")
      end
      expect(auto_cassette).not_to be_nil
    end
    
    it "supports record modes" do
      # Record initially
      Backspin.use_cassette("modes_test") do
        Open3.capture3("echo first")
      end
      
      # :once mode (default) - use existing cassette
      result = Backspin.use_cassette("modes_test", record: :once) do
        Open3.capture3("echo second")
      end
      expect(result[0]).to eq("first\n")
      
      # :all mode - always re-record
      result = Backspin.use_cassette("modes_test", record: :all) do
        Open3.capture3("echo third")
      end
      expect(result[0]).to eq("third\n")
      
      # Verify it was re-recorded
      result = Backspin.use_cassette("modes_test") do
        Open3.capture3("echo fourth")
      end
      expect(result[0]).to eq("third\n")
    end
    
    it "supports :none mode - never record" do
      # Ensure cassette doesn't exist
      FileUtils.rm_f(cassette_dir.join("none_mode_test.yaml"))
      
      expect {
        Backspin.use_cassette("none_mode_test", record: :none) do
          Open3.capture3("echo test")
        end
      }.to raise_error(Backspin::CassetteNotFoundError)
    end
    
    it "supports :new_episodes mode - appends new recordings" do
      # Record initial command
      Backspin.use_cassette("episodes_test", record: :new_episodes) do
        Open3.capture3("echo episode1")
      end
      
      # Different command - should record new episode
      result = StayGold.use_cassette("episodes_test", record: :new_episodes) do
        Open3.capture3("echo episode2")
      end
      expect(result[0]).to eq("episode2\n")
      
      # Verify multiple recordings exist
      cassette_data = YAML.load_file(cassette_dir.join("episodes_test.yaml"))
      expect(cassette_data).to be_a(Array)
      expect(cassette_data.map { |r| r["stdout"] }).to include("episode1\n", "episode2\n")
    end
    
    it "returns stdout, stderr, and status like capture3" do
      stdout, stderr, status = Backspin.use_cassette("full_output_test") do
        Open3.capture3("sh -c 'echo stdout; echo stderr >&2; exit 42'")
      end
      
      expect(stdout).to eq("stdout\n")
      expect(stderr).to eq("stderr\n")
      expect(status).to eq(42)
    end
    
    it "supports options hash" do
      result = Backspin.use_cassette("options_test", 
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
      result = Backspin.use_cassette("return_value_test") do
        stdout, stderr, status = Open3.capture3("echo test")
        "custom return: #{stdout.strip}"
      end
      
      expect(result).to eq("custom return: test")
    end
  end
  
  describe "error handling" do
    it "preserves exceptions from the block" do
      expect {
        Backspin.use_cassette("exception_test") do
          raise "Something went wrong"
        end
      }.to raise_error("Something went wrong")
    end
  end
  
  after do
    FileUtils.rm_rf(cassette_dir)
  end
end