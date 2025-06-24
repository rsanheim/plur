# Ruby syntax error in spec file
RSpec.describe "Syntax error" do
  it "has invalid syntax" do
    # Missing closing parenthesis
    expect(1.to eq(1)
  end
end