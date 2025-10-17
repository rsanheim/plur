RSpec.describe "Calculator" do
  it "adds two numbers" do
    expect(1 + 1).to eq(2)
  end

  it "subtracts two numbers" do
    expect(5 - 3).to eq(2)
  end

  it "multiplies two numbers" do
    expect(3 * 4).to eq(12)
  end
end
