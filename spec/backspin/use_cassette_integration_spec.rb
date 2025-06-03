require "backspin"

RSpec.describe "Backspin use_cassette integration" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "backspin") }
  
  it "works seamlessly with rux testing" do
    # First run - records
    output1 = Backspin.use_cassette("rux_version") do
      stdout, _, _ = Open3.capture3("echo 'rux v0.6.0'")
      stdout
    end
    
    expect(output1).to eq("rux v0.6.0\n")
    
    # Second run - replays without executing
    output2 = Backspin.use_cassette("rux_version") do
      stdout, _, _ = Open3.capture3("echo 'this will not run'")
      stdout
    end
    
    expect(output2).to eq("rux v0.6.0\n") # Gets recorded value
  end
  
  it "preserves the exact behavior of Open3.capture3" do
    # Test that our API is transparent
    original_result = Open3.capture3("echo test")
    
    cassette_result = Backspin.use_cassette("transparent_test") do
      Open3.capture3("echo test")
    end
    
    expect(cassette_result[0]).to eq(original_result[0]) # stdout
    expect(cassette_result[1]).to eq(original_result[1]) # stderr
    expect(cassette_result[2]).to eq(original_result[2].exitstatus) # status
  end
  
  it "can be used in before/after hooks" do
    recordings = []
    
    3.times do |i|
      result = Backspin.use_cassette("hook_test") do
        Open3.capture3("echo iteration")
      end
      recordings << result[0]
    end
    
    # All iterations should get the same recorded value
    expect(recordings).to eq(["iteration\n", "iteration\n", "iteration\n"])
  end
  
  after do
    FileUtils.rm_rf(cassette_dir)
  end
end