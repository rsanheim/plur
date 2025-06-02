require "stay_gold"

RSpec.describe "StayGold simple verify" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "stay_gold") }
  
  it "verifies matching output" do
    # Record
    StayGold.record(record_as: "simple_echo") do
      Open3.capture3("echo hello")
    end
    
    # Verify - should pass
    result = StayGold.verify(cassette: "simple_echo") do
      Open3.capture3("echo hello")
    end
    
    expect(result.verified?).to be true
  end
  
  it "detects non-matching output" do
    # Record
    StayGold.record(record_as: "echo_original") do
      Open3.capture3("echo original")
    end
    
    # Verify - should fail
    result = StayGold.verify(cassette: "echo_original") do
      Open3.capture3("echo different")
    end
    
    expect(result.verified?).to be false
  end
  
  it "raises error when cassette not found" do
    expect {
      StayGold.verify(cassette: "missing") do
        Open3.capture3("echo test")
      end
    }.to raise_error(StayGold::CassetteNotFoundError)
  end
  
  after do
    FileUtils.rm_rf(cassette_dir)
  end
end