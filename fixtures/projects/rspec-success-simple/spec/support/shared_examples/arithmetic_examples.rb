RSpec.shared_examples "an arithmetic operation" do |left, right, expected|
  it "returns the expected result" do
    expect(left + right).to eq(expected)
  end

  it "keeps the result numeric" do
    expect(left + right).to be_a(Integer)
  end
end
