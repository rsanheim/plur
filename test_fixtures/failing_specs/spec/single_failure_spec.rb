RSpec.describe "Single Failure" do
  it "fails due to strings not matching" do
    expected = "All work and no play makes Jack a dull boy"
    actual = "All work and no play makes something something something"

    expect(actual).to eq(expected)
  end
end
