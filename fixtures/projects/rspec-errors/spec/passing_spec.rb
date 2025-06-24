# Normal passing spec for control
RSpec.describe "Passing tests" do
  it "passes" do
    expect(1).to eq(1)
  end
  
  it "also passes" do
    expect(true).to be true
  end
end