# Unhandled exception in test
RSpec.describe "Exceptions" do
  it "raises an unhandled exception" do
    raise "Unhandled error!"
  end
  
  it "passes" do
    expect(1).to eq(1)
  end
end