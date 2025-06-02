require "stay_gold"

RSpec.describe "StayGold edge cases" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "stay_gold") }
  
  it "handles empty record_as string" do
    result = StayGold.record(record_as: "") do
      Open3.capture3("echo empty")
    end
    
    # Should fall back to auto-generated name
    expect(result.cassette_path.to_s).to match(%r{StayGold_edge_cases/handles_empty_record_as_string\.yaml$})
  end
  
  it "prioritizes explicit record_as over auto-generated name" do
    result = StayGold.record(record_as: "custom_name") do
      Open3.capture3("echo custom")
    end
    
    expect(result.cassette_path.to_s).to end_with("custom_name.yaml")
    expect(result.cassette_path.to_s).not_to include("StayGold_edge_cases")
  end
  
  it "sanitizes very long example descriptions" do
    # This is a very long example description that should be sanitized properly when creating the cassette filename
    result = StayGold.record do
      Open3.capture3("echo long")
    end
    
    filename = File.basename(result.cassette_path.to_s)
    expect(filename).to eq("sanitizes_very_long_example_descriptions.yaml")
  end
  
  describe "with slash in description" do
    it "handles slashes/special/characters properly" do
      result = StayGold.record do
        Open3.capture3("echo slashes")
      end
      
      expect(result.cassette_path.to_s).to match(%r{handles_slashesspecialcharacters_properly\.yaml$})
    end
  end
  
  after do
    # Clean up test cassettes
    FileUtils.rm_rf(cassette_dir.join("StayGold_edge_cases"))
    FileUtils.rm_f(cassette_dir.join("custom_name.yaml"))
  end
end