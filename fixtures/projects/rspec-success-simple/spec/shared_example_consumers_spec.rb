RSpec.describe "shared example consumers" do
  context "with a small sum" do
    include_examples "an arithmetic operation", 1, 2, 3
  end

  context "with a larger sum" do
    include_examples "an arithmetic operation", 4, 5, 9
  end

  it "runs an ordinary consumer example" do
    expect(10 + 5).to eq(15)
  end
end
