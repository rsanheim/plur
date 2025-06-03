require "backspin"

RSpec.describe "Backspin simple verify" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "backspin") }
  
  it "verifies matching output" do
    # Record
    Backspin.record("simple_echo") do
      Open3.capture3("echo hello")
    end
    
    # Verify - should pass
    result = Backspin.verify(cassette: "simple_echo") do
      Open3.capture3("echo hello")
    end
    
    expect(result.verified?).to be true
  end
  
  it "detects non-matching output" do
    # Record
    Backspin.record("echo_original") do
      Open3.capture3("echo original")
    end
    
    # Verify - should fail
    result = Backspin.verify(cassette: "echo_original") do
      Open3.capture3("echo different")
    end
    
    expect(result.verified?).to be false
  end
  
  it "raises error when cassette not found" do
    expect {
      Backspin.verify(cassette: "missing") do
        Open3.capture3("echo test")
      end
    }.to raise_error(Backspin::CassetteNotFoundError)
  end
  
  after do
    FileUtils.rm_rf(cassette_dir)
  end
end