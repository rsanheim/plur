# Normal test failure - assertion fails
RSpec.describe "Test failures" do
  it "fails an assertion" do
    expect(1).to eq(2)
  end
  
  it "passes" do
    expect(1).to eq(1)
  end
end