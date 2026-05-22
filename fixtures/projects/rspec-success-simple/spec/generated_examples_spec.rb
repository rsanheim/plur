RSpec.describe "generated examples" do
  generated_values = [1, 2, 3, 4]

  generated_values.each do |value|
    it("runs generated example #{value}") do
      expect(value).to be_positive
    end
  end

  it "runs a unique example one" do
    expect(1).to eq(1)
  end

  it "runs a unique example two" do
    expect(2).to eq(2)
  end

  it "runs a unique example three" do
    expect(3).to eq(3)
  end

  it "runs a unique example four" do
    expect(4).to eq(4)
  end
end
